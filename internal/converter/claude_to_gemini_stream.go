package converter

import "encoding/json"

func (c *claudeToGeminiResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		if event.Event == "done" {
			continue
		}

		var claudeEvent ClaudeStreamEvent
		if err := json.Unmarshal(event.Data, &claudeEvent); err != nil {
			continue
		}

		switch claudeEvent.Type {
		case "content_block_delta":
			if claudeEvent.Delta != nil && claudeEvent.Delta.Type == "text_delta" {
				geminiChunk := GeminiStreamChunk{
					Candidates: []GeminiCandidate{{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: claudeEvent.Delta.Text}},
						},
						Index: 0,
					}},
				}
				output = append(output, FormatSSE("", geminiChunk)...)
			}

		case "message_delta":
			if claudeEvent.Usage != nil {
				if state.Usage == nil {
					state.Usage = &Usage{}
				}
				state.Usage.OutputTokens = claudeEvent.Usage.OutputTokens
			}

		case "message_stop":
			inputTokens, outputTokens := 0, 0
			if state.Usage != nil {
				inputTokens = state.Usage.InputTokens
				outputTokens = state.Usage.OutputTokens
			}
			geminiChunk := GeminiStreamChunk{
				Candidates: []GeminiCandidate{{
					FinishReason: "STOP",
					Index:        0,
				}},
				UsageMetadata: &GeminiUsageMetadata{
					PromptTokenCount:     inputTokens,
					CandidatesTokenCount: outputTokens,
					TotalTokenCount:      inputTokens + outputTokens,
				},
			}
			output = append(output, FormatSSE("", geminiChunk)...)
		}
	}

	return output, nil
}
