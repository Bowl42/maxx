package converter

import "encoding/json"

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
		}

		if len(geminiChunk.Candidates) > 0 {
			candidate := geminiChunk.Candidates[0]
			for _, part := range candidate.Content.Parts {
				// Handle thinking blocks (thought: true)
				if part.Thought && part.Text != "" {
					// Close text block if needed
					if state.CurrentBlockType == "text" {
						blockStop := map[string]interface{}{
							"type":  "content_block_stop",
							"index": state.CurrentIndex,
						}
						output = append(output, FormatSSE("content_block_stop", blockStop)...)
						state.CurrentIndex++
						state.CurrentBlockType = ""
					}
					// Start thinking block if needed
					if state.CurrentBlockType != "thinking" {
						blockStart := map[string]interface{}{
							"type":  "content_block_start",
							"index": state.CurrentIndex,
							"content_block": map[string]interface{}{
								"type":     "thinking",
								"thinking": "",
							},
						}
						output = append(output, FormatSSE("content_block_start", blockStart)...)
						state.CurrentBlockType = "thinking"
					}
					// Send thinking content as thinking_delta
					delta := map[string]interface{}{
						"type":  "content_block_delta",
						"index": state.CurrentIndex,
						"delta": map[string]interface{}{
							"type":     "thinking_delta",
							"thinking": part.Text,
						},
					}
					output = append(output, FormatSSE("content_block_delta", delta)...)
					continue
				}
				if part.Text != "" {
					if state.CurrentBlockType == "thinking" {
						blockStop := map[string]interface{}{
							"type":  "content_block_stop",
							"index": state.CurrentIndex,
						}
						output = append(output, FormatSSE("content_block_stop", blockStop)...)
						state.CurrentIndex++
						state.CurrentBlockType = ""
					}
					if state.CurrentBlockType != "text" {
						blockStart := map[string]interface{}{
							"type":  "content_block_start",
							"index": state.CurrentIndex,
							"content_block": map[string]interface{}{
								"type": "text",
								"text": "",
							},
						}
						output = append(output, FormatSSE("content_block_start", blockStart)...)
						state.CurrentBlockType = "text"
					}
					delta := map[string]interface{}{
						"type":  "content_block_delta",
						"index": state.CurrentIndex,
						"delta": map[string]interface{}{
							"type": "text_delta",
							"text": part.Text,
						},
					}
					output = append(output, FormatSSE("content_block_delta", delta)...)
				}
				if part.FunctionCall != nil {
					if state.CurrentBlockType == "text" || state.CurrentBlockType == "thinking" {
						blockStop := map[string]interface{}{
							"type":  "content_block_stop",
							"index": state.CurrentIndex,
						}
						output = append(output, FormatSSE("content_block_stop", blockStop)...)
						state.CurrentIndex++
						state.CurrentBlockType = ""
					}
					blockStart := map[string]interface{}{
						"type":  "content_block_start",
						"index": state.CurrentIndex,
						"content_block": map[string]interface{}{
							"type":  "tool_use",
							"id":    "call_" + part.FunctionCall.Name,
							"name":  part.FunctionCall.Name,
							"input": part.FunctionCall.Args,
						},
					}
					output = append(output, FormatSSE("content_block_start", blockStart)...)
					blockStop := map[string]interface{}{
						"type":  "content_block_stop",
						"index": state.CurrentIndex,
					}
					output = append(output, FormatSSE("content_block_stop", blockStop)...)
					state.CurrentIndex++
					state.CurrentBlockType = ""
				}
			}

			if candidate.FinishReason != "" {
				if state.CurrentBlockType != "" {
					blockStop := map[string]interface{}{
						"type":  "content_block_stop",
						"index": state.CurrentIndex,
					}
					output = append(output, FormatSSE("content_block_stop", blockStop)...)
					state.CurrentBlockType = ""
				}

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
