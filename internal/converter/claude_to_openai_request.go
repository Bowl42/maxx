package converter

import (
	"encoding/json"
	"strings"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeClaude, domain.ClientTypeOpenAI, &claudeToOpenAIRequest{}, &claudeToOpenAIResponse{})
}

type claudeToOpenAIRequest struct{}

func (c *claudeToOpenAIRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req ClaudeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	openaiReq := OpenAIRequest{
		Model:       model,
		Stream:      stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert system to first message
	if req.System != nil {
		switch s := req.System.(type) {
		case string:
			openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
				Role:    "system",
				Content: s,
			})
		case []interface{}:
			var systemText string
			for _, block := range s {
				if m, ok := block.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok {
						systemText += text
					}
				}
			}
			if systemText != "" {
				openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
					Role:    "system",
					Content: systemText,
				})
			}
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		openaiMsg := OpenAIMessage{Role: msg.Role}
		var toolResultMessages []OpenAIMessage
		var reasoningParts []string
		switch content := msg.Content.(type) {
		case string:
			openaiMsg.Content = content
		case []interface{}:
			var parts []OpenAIContentPart
			var toolCalls []OpenAIToolCall
			for _, block := range content {
				if m, ok := block.(map[string]interface{}); ok {
					blockType, _ := m["type"].(string)
					switch blockType {
					case "thinking":
						if msg.Role == "assistant" {
							if thinkingText := extractClaudeThinkingText(m); strings.TrimSpace(thinkingText) != "" {
								reasoningParts = append(reasoningParts, thinkingText)
							}
						}
					case "redacted_thinking":
						// Ignore redacted thinking blocks.
					case "text":
						if text, ok := m["text"].(string); ok {
							parts = append(parts, OpenAIContentPart{Type: "text", Text: text})
						}
					case "tool_use":
						id, _ := m["id"].(string)
						name, _ := m["name"].(string)
						input := m["input"]
						inputJSON, _ := json.Marshal(input)
						toolCalls = append(toolCalls, OpenAIToolCall{
							ID:       id,
							Type:     "function",
							Function: OpenAIFunctionCall{Name: name, Arguments: string(inputJSON)},
						})
					case "tool_result":
						toolUseID, _ := m["tool_use_id"].(string)
						toolContent := convertClaudeToolResultContentToString(m["content"])
						toolResultMessages = append(toolResultMessages, OpenAIMessage{
							Role:       "tool",
							Content:    toolContent,
							ToolCallID: toolUseID,
						})
						continue
					}
				}
			}
			if len(toolCalls) > 0 {
				openaiMsg.ToolCalls = toolCalls
			}
			if len(parts) == 1 && parts[0].Type == "text" {
				openaiMsg.Content = parts[0].Text
			} else if len(parts) > 0 {
				openaiMsg.Content = parts
			}
		}

		if len(reasoningParts) > 0 {
			openaiMsg.ReasoningContent = strings.Join(reasoningParts, "\n\n")
		}

		// Ensure tool results appear before the current message.
		if len(toolResultMessages) > 0 {
			openaiReq.Messages = append(openaiReq.Messages, toolResultMessages...)
		}

		// Only add message if it has actual content (avoid empty user messages)
		if openaiMsg.Content != nil || len(openaiMsg.ToolCalls) > 0 || openaiMsg.ReasoningContent != nil {
			openaiReq.Messages = append(openaiReq.Messages, openaiMsg)
		}
	}

	// Convert tools
	for _, tool := range req.Tools {
		openaiReq.Tools = append(openaiReq.Tools, OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Convert stop sequences
	if len(req.StopSequences) > 0 {
		openaiReq.Stop = req.StopSequences
	}

	// Convert thinking settings to reasoning_effort when present.
	applyClaudeThinkingToOpenAI(&openaiReq, &req)

	return json.Marshal(openaiReq)
}
