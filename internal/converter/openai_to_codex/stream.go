package openai_to_codex

import (
	"encoding/json"
	"time"

	"github.com/awsl-project/maxx/internal/converter"
)

func (c *Response) TransformChunk(chunk []byte, state *converter.TransformState) ([]byte, error) {
	events, remaining := converter.ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		if event.Event == "done" {
			codexEvent := map[string]interface{}{
				"type": "response.done",
				"response": map[string]interface{}{
					"id":     state.MessageID,
					"status": "completed",
				},
			}
			output = append(output, converter.FormatSSE("", codexEvent)...)
			continue
		}

		var openaiChunk converter.OpenAIStreamChunk
		if err := json.Unmarshal(event.Data, &openaiChunk); err != nil {
			continue
		}

		if state.MessageID == "" {
			state.MessageID = openaiChunk.ID
			codexEvent := map[string]interface{}{
				"type": "response.created",
				"response": map[string]interface{}{
					"id":         openaiChunk.ID,
					"model":      openaiChunk.Model,
					"status":     "in_progress",
					"created_at": time.Now().Unix(),
				},
			}
			output = append(output, converter.FormatSSE("", codexEvent)...)
		}

		if len(openaiChunk.Choices) > 0 {
			choice := openaiChunk.Choices[0]
			if choice.Delta != nil {
				if content, ok := choice.Delta.Content.(string); ok && content != "" {
					codexEvent := map[string]interface{}{
						"type": "response.output_item.delta",
						"delta": map[string]interface{}{
							"type": "text",
							"text": content,
						},
					}
					output = append(output, converter.FormatSSE("", codexEvent)...)
				}
			}

			if choice.FinishReason != "" {
				codexEvent := map[string]interface{}{
					"type": "response.done",
					"response": map[string]interface{}{
						"id":     state.MessageID,
						"status": "completed",
					},
				}
				output = append(output, converter.FormatSSE("", codexEvent)...)
			}
		}
	}

	return output, nil
}
