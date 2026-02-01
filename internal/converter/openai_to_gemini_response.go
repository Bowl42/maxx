package converter

import (
	"encoding/json"
	"strings"
)

type openaiToGeminiResponse struct{}

func (c *openaiToGeminiResponse) Transform(body []byte) ([]byte, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	geminiResp := GeminiResponse{
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     resp.Usage.PromptTokens,
			CandidatesTokenCount: resp.Usage.CompletionTokens,
			TotalTokenCount:      resp.Usage.TotalTokens,
		},
	}

	candidate := GeminiCandidate{
		Content: GeminiContent{Role: "model"},
		Index:   0,
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message != nil {
			if reasoningText := collectReasoningText(choice.Message.ReasoningContent); strings.TrimSpace(reasoningText) != "" {
				candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{
					Text:    reasoningText,
					Thought: true,
				})
			}
			switch content := choice.Message.Content.(type) {
			case string:
				if content != "" {
					candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{Text: content})
				}
			case []interface{}:
				for _, part := range content {
					if m, ok := part.(map[string]interface{}); ok {
						if m["type"] == "text" {
							if text, ok := m["text"].(string); ok && text != "" {
								candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{Text: text})
							}
						}
					}
				}
			}
			for _, tc := range choice.Message.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return nil, err
				}
				candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{
					FunctionCall: &GeminiFunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}

			switch choice.FinishReason {
			case "stop":
				candidate.FinishReason = "STOP"
			case "length":
				candidate.FinishReason = "MAX_TOKENS"
			case "tool_calls":
				candidate.FinishReason = "STOP"
			}
		}
	}

	geminiResp.Candidates = []GeminiCandidate{candidate}
	return json.Marshal(geminiResp)
}
