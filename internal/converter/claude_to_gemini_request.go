package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeClaude, domain.ClientTypeGemini, &claudeToGeminiRequest{}, &claudeToGeminiResponse{})
}

type claudeToGeminiRequest struct{}

func (c *claudeToGeminiRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req ClaudeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	// [CRITICAL FIX] Clean cache_control from all messages before processing
	// This prevents "Extra inputs are not permitted" errors from VS Code and other clients
	cleanCacheControlFromMessages(req.Messages)

	// [CRITICAL FIX] Filter invalid thinking blocks BEFORE processing
	// (like Antigravity-Manager's filter_invalid_thinking_blocks)
	// - Converts thinking with invalid signature to TEXT (preserves content)
	// - Handles both 'assistant' and 'model' roles
	FilterInvalidThinkingBlocks(req.Messages)

	// [CRITICAL FIX] Remove trailing unsigned thinking blocks
	// (like Antigravity-Manager's remove_trailing_unsigned_thinking)
	RemoveTrailingUnsignedThinking(req.Messages)

	// Detect web search tool presence
	hasWebSearch := hasWebSearchTool(req.Tools)

	// Track tool_use id -> name mapping (critical for tool_result handling)
	toolIDToName := make(map[string]string)

	// Track last thought signature for backfill
	var lastThoughtSignature string

	// Determine if thinking is enabled (like Antigravity-Manager)
	isThinkingEnabled := false
	var thinkingBudget int
	if req.Thinking != nil {
		if enabled, ok := req.Thinking["type"].(string); ok && enabled == "enabled" {
			isThinkingEnabled = true
			if budget, ok := req.Thinking["budget_tokens"].(float64); ok {
				thinkingBudget = int(budget)
			}
		}
	} else {
		// [Claude Code v2.0.67+] Default thinking enabled for Opus 4.5
		isThinkingEnabled = shouldEnableThinkingByDefault(req.Model)
	}

	// [NEW FIX] Check if target model supports thinking
	if isThinkingEnabled && !targetModelSupportsThinking(model) {
		isThinkingEnabled = false
	}

	// Check if thinking should be disabled due to history
	if isThinkingEnabled && shouldDisableThinkingDueToHistory(req.Messages) {
		isThinkingEnabled = false
	}

	// [FIX #295 & #298] Signature validation for function calls
	// If thinking enabled but no valid signature and has function calls, disable thinking
	if isThinkingEnabled {
		hasThinkingHist := hasThinkingHistory(req.Messages)
		hasFuncCalls := hasFunctionCalls(req.Messages)

		// Only enforce strict signature checks when function calls are involved
		if hasFuncCalls && !hasThinkingHist {
			// Get global signature (empty string if not available)
			globalSig := "" // TODO: integrate with signature cache
			if !hasValidSignatureForFunctionCalls(req.Messages, globalSig) {
				isThinkingEnabled = false
			}
		}
	}

	// Build generation config (like Antigravity-Manager)
	genConfig := &GeminiGenerationConfig{
		MaxOutputTokens: 64000, // Fixed value like Antigravity-Manager
		StopSequences:   defaultStopSequences(),
	}

	if req.Temperature != nil {
		genConfig.Temperature = req.Temperature
	}
	if req.TopP != nil {
		genConfig.TopP = req.TopP
	}
	if req.TopK != nil {
		genConfig.TopK = req.TopK
	}

	// Effort level mapping (Claude API v2.0.67+)
	if req.OutputConfig != nil && req.OutputConfig.Effort != "" {
		effort := strings.ToLower(req.OutputConfig.Effort)
		switch effort {
		case "high":
			genConfig.EffortLevel = "HIGH"
		case "medium":
			genConfig.EffortLevel = "MEDIUM"
		case "low":
			genConfig.EffortLevel = "LOW"
		default:
			genConfig.EffortLevel = "HIGH"
		}
	}

	// Add thinking config if enabled
	if isThinkingEnabled {
		genConfig.ThinkingConfig = &GeminiThinkingConfig{
			IncludeThoughts: true,
		}
		if thinkingBudget > 0 {
			// Cap at 24576 for flash models or web search
			if (strings.Contains(strings.ToLower(model), "flash") || hasWebSearch) && thinkingBudget > 24576 {
				thinkingBudget = 24576
			}
			genConfig.ThinkingConfig.ThinkingBudget = thinkingBudget
		}
	}

	geminiReq := GeminiRequest{
		GenerationConfig: genConfig,
		SafetySettings:   defaultSafetySettings(),
	}

	// Build system instruction with multiple parts (like Antigravity-Manager)
	var systemParts []GeminiPart
	systemParts = append(systemParts, GeminiPart{Text: buildIdentityPatch(model)})

	if req.System != nil {
		switch s := req.System.(type) {
		case string:
			if s != "" {
				systemParts = append(systemParts, GeminiPart{Text: s})
			}
		case []interface{}:
			for _, block := range s {
				if m, ok := block.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok && text != "" {
						systemParts = append(systemParts, GeminiPart{Text: text})
					}
				}
			}
		}
	}

	systemParts = append(systemParts, GeminiPart{Text: "\n--- [SYSTEM_PROMPT_END] ---"})
	// [FIX] Set role to "user" for systemInstruction (like CLIProxyAPI commit 67985d8)
	geminiReq.SystemInstruction = &GeminiContent{Role: "user", Parts: systemParts}

	// Convert messages to contents
	var contents []GeminiContent
	for _, msg := range req.Messages {
		geminiContent := GeminiContent{}

		// Map role
		switch msg.Role {
		case "user":
			geminiContent.Role = "user"
		case "assistant":
			geminiContent.Role = "model"
		default:
			geminiContent.Role = msg.Role
		}

		var parts []GeminiPart

		switch content := msg.Content.(type) {
		case string:
			if content != "(no content)" && strings.TrimSpace(content) != "" {
				parts = append(parts, GeminiPart{Text: strings.TrimSpace(content)})
			}

		case []interface{}:
			for _, block := range content {
				m, ok := block.(map[string]interface{})
				if !ok {
					continue
				}

				blockType, _ := m["type"].(string)

				switch blockType {
				case "text":
					text, _ := m["text"].(string)
					if text != "(no content)" && text != "" {
						parts = append(parts, GeminiPart{Text: text})
					}

				case "thinking":
					thinking, _ := m["thinking"].(string)
					signature, _ := m["signature"].(string)

					// If thinking is disabled, convert to text
					if !isThinkingEnabled {
						if thinking != "" {
							parts = append(parts, GeminiPart{Text: thinking})
						}
						continue
					}

					// Thinking block must be first in the message
					if len(parts) > 0 {
						// Downgrade to text
						if thinking != "" {
							parts = append(parts, GeminiPart{Text: thinking})
						}
						continue
					}

					// Empty thinking blocks -> downgrade to text
					if thinking == "" {
						parts = append(parts, GeminiPart{Text: "..."})
						continue
					}

					part := GeminiPart{
						Text:    thinking,
						Thought: true,
					}
					if signature != "" {
						part.ThoughtSignature = signature
						lastThoughtSignature = signature
					}
					parts = append(parts, part)

				case "tool_use":
					id, _ := m["id"].(string)
					name, _ := m["name"].(string)
					input, _ := m["input"].(map[string]interface{})

					// Clean input schema
					if input != nil {
						cleanJSONSchema(input)
					}

					// Store id -> name mapping
					if id != "" && name != "" {
						toolIDToName[id] = name
					}

					part := GeminiPart{
						FunctionCall: &GeminiFunctionCall{
							Name: name,
							Args: input,
							ID:   id, // Include ID (like Antigravity-Manager)
						},
					}

					// Backfill thoughtSignature if available
					if lastThoughtSignature != "" {
						part.ThoughtSignature = lastThoughtSignature
					}

					parts = append(parts, part)

				case "tool_result":
					toolUseID, _ := m["tool_use_id"].(string)

					// Handle content: can be string or array
					var resultContent string
					switch c := m["content"].(type) {
					case string:
						resultContent = c
					case []interface{}:
						var textParts []string
						for _, block := range c {
							if blockMap, ok := block.(map[string]interface{}); ok {
								if text, ok := blockMap["text"].(string); ok {
									textParts = append(textParts, text)
								}
							}
						}
						resultContent = strings.Join(textParts, "\n")
					}

					// Handle empty content
					if strings.TrimSpace(resultContent) == "" {
						isError, _ := m["is_error"].(bool)
						if isError {
							resultContent = "Tool execution failed with no output."
						} else {
							resultContent = "Command executed successfully."
						}
					}

					// Use stored function name, fallback to tool_use_id
					funcName := toolUseID
					if name, ok := toolIDToName[toolUseID]; ok {
						funcName = name
					}

					part := GeminiPart{
						FunctionResponse: &GeminiFunctionResponse{
							Name:     funcName,
							Response: map[string]string{"result": resultContent},
							ID:       toolUseID, // Include ID (like Antigravity-Manager)
						},
					}

					// Backfill thoughtSignature if available
					if lastThoughtSignature != "" {
						part.ThoughtSignature = lastThoughtSignature
					}

					// tool_result sets role to user
					geminiContent.Role = "user"
					parts = append(parts, part)

				case "image":
					source, _ := m["source"].(map[string]interface{})
					if source != nil {
						sourceType, _ := source["type"].(string)
						if sourceType == "base64" {
							mediaType, _ := source["media_type"].(string)
							data, _ := source["data"].(string)
							parts = append(parts, GeminiPart{
								InlineData: &GeminiInlineData{
									MimeType: mediaType,
									Data:     data,
								},
							})
						}
					}

				case "document":
					// Document block (PDF, etc) - convert to inline data
					source, _ := m["source"].(map[string]interface{})
					if source != nil {
						sourceType, _ := source["type"].(string)
						if sourceType == "base64" {
							mediaType, _ := source["media_type"].(string)
							data, _ := source["data"].(string)
							parts = append(parts, GeminiPart{
								InlineData: &GeminiInlineData{
									MimeType: mediaType,
									Data:     data,
								},
							})
						}
					}

				case "redacted_thinking":
					// RedactedThinking block - downgrade to text (like Antigravity-Manager)
					data, _ := m["data"].(string)
					parts = append(parts, GeminiPart{
						Text: fmt.Sprintf("[Redacted Thinking: %s]", data),
					})

				case "server_tool_use", "web_search_tool_result":
					// Server tool blocks should not be sent to upstream
					continue
				}
			}
		}

		// Skip empty messages
		if len(parts) == 0 {
			continue
		}

		geminiContent.Parts = parts
		contents = append(contents, geminiContent)
	}

	// Merge adjacent messages with same role (like Antigravity-Manager)
	contents = mergeAdjacentRoles(contents)

	// Clean thinking fields if thinking is disabled
	if !isThinkingEnabled {
		for i := range contents {
			for j := range contents[i].Parts {
				contents[i].Parts[j].Thought = false
				contents[i].Parts[j].ThoughtSignature = ""
			}
		}
	}

	geminiReq.Contents = contents

	// Convert tools (like Antigravity-Manager's build_tools)
	if len(req.Tools) > 0 {
		var funcDecls []GeminiFunctionDecl
		hasGoogleSearch := hasWebSearch

		for _, tool := range req.Tools {
			// 1. Detect server tools / built-in tools like web_search
			if tool.IsWebSearch() {
				hasGoogleSearch = true
				continue
			}

			// 2. Client tools require name and input_schema
			if tool.Name == "" {
				continue
			}

			inputSchema := tool.InputSchema
			if inputSchema == nil {
				inputSchema = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}

			// Clean input schema
			if schemaMap, ok := inputSchema.(map[string]interface{}); ok {
				cleanJSONSchema(schemaMap)
			}

			funcDecls = append(funcDecls, GeminiFunctionDecl{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  inputSchema,
			})
		}

		// [FIX] Gemini v1internal does not support mixing Google Search with function declarations
		if len(funcDecls) > 0 {
			// If has local tools, use local tools only, skip Google Search injection
			geminiReq.Tools = []GeminiTool{{FunctionDeclarations: funcDecls}}
			geminiReq.ToolConfig = &GeminiToolConfig{
				FunctionCallingConfig: &GeminiFunctionCallingConfig{
					Mode: "VALIDATED",
				},
			}
		} else if hasGoogleSearch {
			// Only inject Google Search if no local tools
			geminiReq.Tools = []GeminiTool{{
				GoogleSearch: &struct{}{},
			}}
		}
	}

	return json.Marshal(geminiReq)
}

// mergeAdjacentRoles merges adjacent messages with the same role
