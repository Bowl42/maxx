package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/awsl-project/maxx/internal/domain"
)

// remapFunctionCallArgs remaps Gemini's function call arguments to Claude Code expected format
// This is critical for Claude Code compatibility as Gemini sometimes uses different parameter names
func remapFunctionCallArgs(toolName string, args map[string]interface{}) {
	if args == nil {
		return
	}

	toolNameLower := strings.ToLower(toolName)

	switch toolNameLower {
	case "grep":
		// Gemini uses "query", Claude Code expects "pattern"
		if query, ok := args["query"]; ok {
			if _, hasPattern := args["pattern"]; !hasPattern {
				args["pattern"] = query
				delete(args, "query")
			}
		}
		// Claude Code uses "path" (string), NOT "paths" (array)
		if _, hasPath := args["path"]; !hasPath {
			if paths, ok := args["paths"]; ok {
				pathStr := extractFirstPath(paths)
				args["path"] = pathStr
				delete(args, "paths")
			} else {
				args["path"] = "."
			}
		}

	case "glob":
		// Gemini uses "query", Claude Code expects "pattern"
		if query, ok := args["query"]; ok {
			if _, hasPattern := args["pattern"]; !hasPattern {
				args["pattern"] = query
				delete(args, "query")
			}
		}
		// Claude Code uses "path" (string), NOT "paths" (array)
		if _, hasPath := args["path"]; !hasPath {
			if paths, ok := args["paths"]; ok {
				pathStr := extractFirstPath(paths)
				args["path"] = pathStr
				delete(args, "paths")
			} else {
				args["path"] = "."
			}
		}

	case "read":
		// Gemini might use "path" vs "file_path"
		if path, ok := args["path"]; ok {
			if _, hasFilePath := args["file_path"]; !hasFilePath {
				args["file_path"] = path
				delete(args, "path")
			}
		}

	case "ls":
		// LS tool: ensure "path" parameter exists
		if _, hasPath := args["path"]; !hasPath {
			args["path"] = "."
		}
	}
}

// extractFirstPath extracts the first path from various input formats
func extractFirstPath(paths interface{}) string {
	switch v := paths.(type) {
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
		return "."
	case string:
		return v
	default:
		return "."
	}
}

func init() {
	RegisterConverter(domain.ClientTypeGemini, domain.ClientTypeClaude, &geminiToClaudeRequest{}, &geminiToClaudeResponse{})
}

type geminiToClaudeRequest struct{}
type geminiToClaudeResponse struct{}

func (c *geminiToClaudeRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req GeminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	claudeReq := ClaudeRequest{
		Model:  model,
		Stream: stream,
	}

	if req.GenerationConfig != nil {
		claudeReq.MaxTokens = req.GenerationConfig.MaxOutputTokens
		claudeReq.Temperature = req.GenerationConfig.Temperature
		claudeReq.TopP = req.GenerationConfig.TopP
		claudeReq.TopK = req.GenerationConfig.TopK
		claudeReq.StopSequences = req.GenerationConfig.StopSequences
	}

	// Convert systemInstruction
	if req.SystemInstruction != nil {
		var systemText string
		for _, part := range req.SystemInstruction.Parts {
			systemText += part.Text
		}
		if systemText != "" {
			claudeReq.System = systemText
		}
	}

	// Convert contents to messages
	toolCallCounter := 0
	for _, content := range req.Contents {
		claudeMsg := ClaudeMessage{}
		// Map role
		switch content.Role {
		case "user":
			claudeMsg.Role = "user"
		case "model":
			claudeMsg.Role = "assistant"
		default:
			claudeMsg.Role = "user"
		}

		var blocks []ClaudeContentBlock
		for _, part := range content.Parts {
			if part.Text != "" {
				blocks = append(blocks, ClaudeContentBlock{Type: "text", Text: part.Text})
			}
			if part.FunctionCall != nil {
				toolCallCounter++
				blocks = append(blocks, ClaudeContentBlock{
					Type:  "tool_use",
					ID:    fmt.Sprintf("call_%d", toolCallCounter),
					Name:  part.FunctionCall.Name,
					Input: part.FunctionCall.Args,
				})
			}
			if part.FunctionResponse != nil {
				respJSON, _ := json.Marshal(part.FunctionResponse.Response)
				blocks = append(blocks, ClaudeContentBlock{
					Type:      "tool_result",
					ToolUseID: part.FunctionResponse.Name,
					Content:   string(respJSON),
				})
			}
		}

		if len(blocks) == 1 && blocks[0].Type == "text" {
			claudeMsg.Content = blocks[0].Text
		} else if len(blocks) > 0 {
			claudeMsg.Content = blocks
		}

		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	// Convert tools
	for _, tool := range req.Tools {
		for _, decl := range tool.FunctionDeclarations {
			claudeReq.Tools = append(claudeReq.Tools, ClaudeTool{
				Name:        decl.Name,
				Description: decl.Description,
				InputSchema: decl.Parameters,
			})
		}
	}

	return json.Marshal(claudeReq)
}

func (c *geminiToClaudeResponse) Transform(body []byte) ([]byte, error) {
	var resp GeminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	claudeResp := ClaudeResponse{
		ID:   "msg_gemini",
		Type: "message",
		Role: "assistant",
	}

	if resp.UsageMetadata != nil {
		claudeResp.Usage = ClaudeUsage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		}
	}

	hasToolUse := false
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		toolCallCounter := 0
		for _, part := range candidate.Content.Parts {
			// Handle thinking blocks (thought: true)
			if part.Thought && part.Text != "" {
				claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
					Type:      "thinking",
					Thinking:  part.Text,
					Signature: part.ThoughtSignature,
				})
				continue
			}
			if part.Text != "" {
				claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
					Type: "text",
					Text: part.Text,
				})
			}
			if part.FunctionCall != nil {
				hasToolUse = true
				toolCallCounter++
				// Apply argument remapping for Claude Code compatibility
				args := part.FunctionCall.Args
				remapFunctionCallArgs(part.FunctionCall.Name, args)
				claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
					Type:  "tool_use",
					ID:    fmt.Sprintf("call_%d", toolCallCounter),
					Name:  part.FunctionCall.Name,
					Input: args,
				})
			}
		}

		// Map finish reason
		switch candidate.FinishReason {
		case "STOP":
			if hasToolUse {
				claudeResp.StopReason = "tool_use"
			} else {
				claudeResp.StopReason = "end_turn"
			}
		case "MAX_TOKENS":
			claudeResp.StopReason = "max_tokens"
		default:
			claudeResp.StopReason = "end_turn"
		}
	}

	return json.Marshal(claudeResp)
}

func (c *geminiToClaudeResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		var geminiChunk GeminiStreamChunk
		if err := json.Unmarshal(event.Data, &geminiChunk); err != nil {
			continue
		}

		// First chunk - send message_start
		if state.MessageID == "" {
			state.MessageID = "msg_gemini"
			msgStart := map[string]interface{}{
				"type": "message_start",
				"message": map[string]interface{}{
					"id":    state.MessageID,
					"type":  "message",
					"role":  "assistant",
					"usage": map[string]int{"input_tokens": 0, "output_tokens": 0},
				},
			}
			output = append(output, FormatSSE("message_start", msgStart)...)

			blockStart := map[string]interface{}{
				"type":  "content_block_start",
				"index": 0,
				"content_block": map[string]interface{}{
					"type": "text",
					"text": "",
				},
			}
			output = append(output, FormatSSE("content_block_start", blockStart)...)
		}

		if len(geminiChunk.Candidates) > 0 {
			candidate := geminiChunk.Candidates[0]
			for _, part := range candidate.Content.Parts {
				// Handle thinking blocks (thought: true)
				if part.Thought && part.Text != "" {
					// Send thinking content as thinking_delta
					delta := map[string]interface{}{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]interface{}{
							"type":     "thinking_delta",
							"thinking": part.Text,
						},
					}
					output = append(output, FormatSSE("content_block_delta", delta)...)
					continue
				}
				if part.Text != "" {
					delta := map[string]interface{}{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]interface{}{
							"type": "text_delta",
							"text": part.Text,
						},
					}
					output = append(output, FormatSSE("content_block_delta", delta)...)
				}
			}

			if candidate.FinishReason != "" {
				blockStop := map[string]interface{}{
					"type":  "content_block_stop",
					"index": 0,
				}
				output = append(output, FormatSSE("content_block_stop", blockStop)...)

				stopReason := "end_turn"
				if candidate.FinishReason == "MAX_TOKENS" {
					stopReason = "max_tokens"
				}

				msgDelta := map[string]interface{}{
					"type": "message_delta",
					"delta": map[string]interface{}{
						"stop_reason": stopReason,
					},
					"usage": map[string]int{"output_tokens": state.Usage.OutputTokens},
				}
				output = append(output, FormatSSE("message_delta", msgDelta)...)
				output = append(output, FormatSSE("message_stop", map[string]string{"type": "message_stop"})...)
			}
		}

		if geminiChunk.UsageMetadata != nil {
			state.Usage.InputTokens = geminiChunk.UsageMetadata.PromptTokenCount
			state.Usage.OutputTokens = geminiChunk.UsageMetadata.CandidatesTokenCount
		}
	}

	return output, nil
}
