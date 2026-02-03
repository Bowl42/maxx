package openai_to_codex

import (
	"encoding/json"

	"github.com/awsl-project/maxx/internal/converter"
)

type Response struct{}

func (c *Response) Transform(body []byte) ([]byte, error) {
	var resp converter.OpenAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	codexResp := converter.CodexResponse{
		ID:        resp.ID,
		Object:    "response",
		CreatedAt: resp.Created,
		Model:     resp.Model,
		Status:    "completed",
		Usage: converter.CodexUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message != nil {
			if content, ok := choice.Message.Content.(string); ok && content != "" {
				codexResp.Output = append(codexResp.Output, converter.CodexOutput{
					Type:    "message",
					Role:    "assistant",
					Content: content,
				})
			}
			for _, tc := range choice.Message.ToolCalls {
				codexResp.Output = append(codexResp.Output, converter.CodexOutput{
					Type:      "function_call",
					ID:        tc.ID,
					CallID:    tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
					Status:    "completed",
				})
			}
		}
	}

	return json.Marshal(codexResp)
}
