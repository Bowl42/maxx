package converter

import (
	"encoding/json"
	"time"
)

func (c *claudeToOpenAIResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
	state.Buffer = remaining

	var output []byte
	for _, event := range events {
		if event.Event == "done" {
			output = append(output, FormatDone()...)
			continue
		}

		var claudeEvent ClaudeStreamEvent
		if err := json.Unmarshal(event.Data, &claudeEvent); err != nil {
			continue
		}

		switch claudeEvent.Type {
		case "message_start":
			if claudeEvent.Message != nil {
				state.MessageID = claudeEvent.Message.ID
			}
			chunk := OpenAIStreamChunk{
				ID:      state.MessageID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Choices: []OpenAIChoice{{
					Index: 0,
					Delta: &OpenAIMessage{Role: "assistant", Content: ""},
				}},
			}
			output = append(output, FormatSSE("", chunk)...)

		case "content_block_start":
			if claudeEvent.ContentBlock != nil {
				state.CurrentBlockType = claudeEvent.ContentBlock.Type
				state.CurrentIndex = claudeEvent.Index
				if claudeEvent.ContentBlock.Type == "tool_use" {
					if state.ToolCalls == nil {
						state.ToolCalls = make(map[int]*ToolCallState)
					}
					state.ToolCalls[claudeEvent.Index] = &ToolCallState{
						ID:   claudeEvent.ContentBlock.ID,
						Name: claudeEvent.ContentBlock.Name,
					}
				}
			}

		case "content_block_delta":
			if claudeEvent.Delta != nil {
				switch claudeEvent.Delta.Type {
				case "text_delta":
					chunk := OpenAIStreamChunk{
						ID:      state.MessageID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Choices: []OpenAIChoice{{
							Index: 0,
							Delta: &OpenAIMessage{Content: claudeEvent.Delta.Text},
						}},
					}
					output = append(output, FormatSSE("", chunk)...)
				case "thinking_delta":
					if claudeEvent.Delta.Thinking != "" {
						chunk := OpenAIStreamChunk{
							ID:      state.MessageID,
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Choices: []OpenAIChoice{{
								Index: 0,
								Delta: &OpenAIMessage{ReasoningContent: claudeEvent.Delta.Thinking},
							}},
						}
						output = append(output, FormatSSE("", chunk)...)
					}
				case "input_json_delta":
					if tc, ok := state.ToolCalls[state.CurrentIndex]; ok {
						tc.Arguments += claudeEvent.Delta.PartialJSON
						chunk := OpenAIStreamChunk{
							ID:      state.MessageID,
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Choices: []OpenAIChoice{{
								Index: 0,
								Delta: &OpenAIMessage{
									ToolCalls: []OpenAIToolCall{{
										Index:    state.CurrentIndex,
										ID:       tc.ID,
										Type:     "function",
										Function: OpenAIFunctionCall{Name: tc.Name, Arguments: claudeEvent.Delta.PartialJSON},
									}},
								},
							}},
						}
						output = append(output, FormatSSE("", chunk)...)
					}
				}
			}

		case "message_delta":
			if claudeEvent.Delta != nil {
				state.StopReason = claudeEvent.Delta.StopReason
			}
			if claudeEvent.Usage != nil {
				if state.Usage == nil {
					state.Usage = &Usage{}
				}
				state.Usage.OutputTokens = claudeEvent.Usage.OutputTokens
			}

		case "message_stop":
			finishReason := "stop"
			switch state.StopReason {
			case "end_turn":
				finishReason = "stop"
			case "max_tokens":
				finishReason = "length"
			case "tool_use":
				finishReason = "tool_calls"
			}
			chunk := OpenAIStreamChunk{
				ID:      state.MessageID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Choices: []OpenAIChoice{{
					Index:        0,
					Delta:        &OpenAIMessage{},
					FinishReason: finishReason,
				}},
			}
			output = append(output, FormatSSE("", chunk)...)
			output = append(output, FormatDone()...)
		}
	}

	return output, nil
}
