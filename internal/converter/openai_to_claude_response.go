package converter

import (
	"encoding/json"
	"strings"
)

type openaiToClaudeResponse struct{}

func (c *openaiToClaudeResponse) Transform(body []byte) ([]byte, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	claudeResp := ClaudeResponse{
		ID:    resp.ID,
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
		Usage: ClaudeUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message != nil {
			// Convert reasoning_content to thinking blocks
			if reasoningText := collectReasoningText(choice.Message.ReasoningContent); strings.TrimSpace(reasoningText) != "" {
				claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
					Type:     "thinking",
					Thinking: reasoningText,
				})
			}

			// Convert content
			switch content := choice.Message.Content.(type) {
			case string:
				if content != "" {
					claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
						Type: "text",
						Text: content,
					})
				}
			case []interface{}:
				for _, part := range content {
					if m, ok := part.(map[string]interface{}); ok {
						if m["type"] == "text" {
							if text, ok := m["text"].(string); ok && text != "" {
								claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
									Type: "text",
									Text: text,
								})
							}
						}
					}
				}
			}

			// Convert tool calls
			for _, tc := range choice.Message.ToolCalls {
				var input interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				claudeResp.Content = append(claudeResp.Content, ClaudeContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}

			// Map finish reason
			switch choice.FinishReason {
			case "stop":
				claudeResp.StopReason = "end_turn"
			case "length":
				claudeResp.StopReason = "max_tokens"
			case "tool_calls":
				claudeResp.StopReason = "tool_use"
			}
		}
	}

	return json.Marshal(claudeResp)
}
