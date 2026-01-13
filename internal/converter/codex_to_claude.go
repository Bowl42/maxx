package converter

import (
	"encoding/json"

	"github.com/Bowl42/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeCodex, domain.ClientTypeClaude, &codexToClaudeRequest{}, &codexToClaudeResponse{})
}

type codexToClaudeRequest struct{}
type codexToClaudeResponse struct{}

func (c *codexToClaudeRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req CodexRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	claudeReq := ClaudeRequest{
		Model:       model,
		Stream:      stream,
		MaxTokens:   req.MaxOutputTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert instructions to system prompt
	if req.Instructions != "" {
		claudeReq.System = req.Instructions
	}

	// Convert input to Claude messages
	switch input := req.Input.(type) {
	case string:
		claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
			Role:    "user",
			Content: input,
		})
	case []interface{}:
		for _, item := range input {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			itemType, _ := m["type"].(string)
			role, _ := m["role"].(string)

			switch itemType {
			case "message":
				if role == "" {
					role = "user"
				}
				claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
					Role:    role,
					Content: m["content"],
				})
			case "function_call":
				// Convert function call to tool_use block
				id, _ := m["id"].(string)
				if id == "" {
					id, _ = m["call_id"].(string)
				}
				name, _ := m["name"].(string)
				argStr, _ := m["arguments"].(string)
				var args interface{}
				json.Unmarshal([]byte(argStr), &args)
				claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
					Role: "assistant",
					Content: []ClaudeContentBlock{{
						Type:  "tool_use",
						ID:    id,
						Name:  name,
						Input: args,
					}},
				})
			case "function_call_output":
				// Convert function call output to tool_result
				callID, _ := m["call_id"].(string)
				outputStr, _ := m["output"].(string)
				claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
					Role: "user",
					Content: []ClaudeContentBlock{{
						Type:      "tool_result",
						ToolUseID: callID,
						Content:   outputStr,
					}},
				})
			}
		}
	}

	// Convert tools
	for _, tool := range req.Tools {
		claudeReq.Tools = append(claudeReq.Tools, ClaudeTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}

	return json.Marshal(claudeReq)
}

func (c *codexToClaudeResponse) Transform(body []byte) ([]byte, error) {
	var resp CodexResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	claudeResp := ClaudeResponse{
		ID:    resp.ID,
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
		Usage: ClaudeUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}

	var hasToolCall bool
	for _, out := range resp.Output {
		switch out.Type {
		case "message":
			contentStr, _ := out.Content.(string)
			claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
				Type: "text",
				Text: contentStr,
			})
		case "function_call":
			hasToolCall = true
			var args interface{}
			json.Unmarshal([]byte(out.Arguments), &args)
			claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
				Type:  "tool_use",
				ID:    out.ID,
				Name:  out.Name,
				Input: args,
			})
		}
	}

	if hasToolCall {
		claudeResp.StopReason = "tool_use"
	} else {
		claudeResp.StopReason = "end_turn"
	}

	return json.Marshal(claudeResp)
}

func (c *codexToClaudeResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		var codexEvent map[string]interface{}
		if err := json.Unmarshal(event.Data, &codexEvent); err != nil {
			continue
		}

		eventType, _ := codexEvent["type"].(string)

		switch eventType {
		case "response.created":
			if resp, ok := codexEvent["response"].(map[string]interface{}); ok {
				state.MessageID, _ = resp["id"].(string)
			}
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

		case "response.output_item.delta":
			if delta, ok := codexEvent["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					claudeDelta := map[string]interface{}{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]interface{}{
							"type": "text_delta",
							"text": text,
						},
					}
					output = append(output, FormatSSE("content_block_delta", claudeDelta)...)
				}
			}

		case "response.done":
			blockStop := map[string]interface{}{
				"type":  "content_block_stop",
				"index": 0,
			}
			output = append(output, FormatSSE("content_block_stop", blockStop)...)

			msgDelta := map[string]interface{}{
				"type": "message_delta",
				"delta": map[string]interface{}{
					"stop_reason": "end_turn",
				},
				"usage": map[string]int{"output_tokens": 0},
			}
			output = append(output, FormatSSE("message_delta", msgDelta)...)
			output = append(output, FormatSSE("message_stop", map[string]string{"type": "message_stop"})...)
		}
	}

	return output, nil
}
