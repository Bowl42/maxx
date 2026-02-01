package converter

import (
	"encoding/json"
	"fmt"
)

type geminiToClaudeResponse struct{}

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
