package converter

import "encoding/json"

type claudeToGeminiResponse struct{}

func (c *claudeToGeminiResponse) Transform(body []byte) ([]byte, error) {
	var resp ClaudeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	geminiResp := GeminiResponse{
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     resp.Usage.InputTokens,
			CandidatesTokenCount: resp.Usage.OutputTokens,
			TotalTokenCount:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	candidate := GeminiCandidate{
		Content: GeminiContent{Role: "model"},
		Index:   0,
	}

	// Convert content
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{Text: block.Text})
		case "tool_use":
			inputMap, _ := block.Input.(map[string]interface{})
			candidate.Content.Parts = append(candidate.Content.Parts, GeminiPart{
				FunctionCall: &GeminiFunctionCall{
					Name: block.Name,
					Args: inputMap,
					ID:   block.ID,
				},
			})
		}
	}

	// Map stop reason
	switch resp.StopReason {
	case "end_turn":
		candidate.FinishReason = "STOP"
	case "max_tokens":
		candidate.FinishReason = "MAX_TOKENS"
	case "tool_use":
		candidate.FinishReason = "STOP"
	}

	geminiResp.Candidates = []GeminiCandidate{candidate}
	return json.Marshal(geminiResp)
}
