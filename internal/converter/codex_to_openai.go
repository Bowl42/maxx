package converter

import (
	"encoding/json"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeCodex, domain.ClientTypeOpenAI, &codexToOpenAIRequest{}, &codexToOpenAIResponse{})
}

type codexToOpenAIRequest struct{}
type codexToOpenAIResponse struct{}

func (c *codexToOpenAIRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req CodexRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	openaiReq := OpenAIRequest{
		Model:       model,
		Stream:      stream,
		MaxTokens:   req.MaxOutputTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		openaiReq.ReasoningEffort = req.Reasoning.Effort
	}

	// Convert instructions to system message
	if req.Instructions != "" {
		openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
			Role:    "system",
			Content: req.Instructions,
		})
	}

	// Convert input to messages
	switch input := req.Input.(type) {
	case string:
		openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
			Role:    "user",
			Content: input,
		})
	case []interface{}:
		for _, item := range input {
			if m, ok := item.(map[string]interface{}); ok {
				itemType, _ := m["type"].(string)
				role, _ := m["role"].(string)
				switch itemType {
				case "message":
					if role == "" {
						role = "user"
					}
					openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
						Role:    role,
						Content: m["content"],
					})
				case "function_call":
					id, _ := m["id"].(string)
					if id == "" {
						id, _ = m["call_id"].(string)
					}
					name, _ := m["name"].(string)
					args, _ := m["arguments"].(string)
					openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
						Role: "assistant",
						ToolCalls: []OpenAIToolCall{{
							ID:   id,
							Type: "function",
							Function: OpenAIFunctionCall{
								Name:      name,
								Arguments: args,
							},
						}},
					})
				case "function_call_output":
					callID, _ := m["call_id"].(string)
					outputStr, _ := m["output"].(string)
					openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
						Role:       "tool",
						Content:    outputStr,
						ToolCallID: callID,
					})
				}
			}
		}
	}

	// Convert tools
	for _, tool := range req.Tools {
		openaiReq.Tools = append(openaiReq.Tools, OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	return json.Marshal(openaiReq)
}

func (c *codexToOpenAIResponse) Transform(body []byte) ([]byte, error) {
	var resp CodexResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	openaiResp := OpenAIResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.CreatedAt,
		Model:   resp.Model,
		Usage: OpenAIUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	msg := OpenAIMessage{Role: "assistant"}
	var textContent string
	var toolCalls []OpenAIToolCall

	for _, out := range resp.Output {
		switch out.Type {
		case "message":
			if s, ok := out.Content.(string); ok {
				textContent += s
			}
		case "function_call":
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   out.ID,
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      out.Name,
					Arguments: out.Arguments,
				},
			})
		}
	}

	if textContent != "" {
		msg.Content = textContent
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	openaiResp.Choices = []OpenAIChoice{{
		Index:        0,
		Message:      &msg,
		FinishReason: finishReason,
	}}

	return json.Marshal(openaiResp)
}

func (c *codexToOpenAIResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
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
			openaiChunk := OpenAIStreamChunk{
				ID:      state.MessageID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Choices: []OpenAIChoice{{
					Index: 0,
					Delta: &OpenAIMessage{Role: "assistant", Content: ""},
				}},
			}
			output = append(output, FormatSSE("", openaiChunk)...)

		case "response.output_item.delta":
			if delta, ok := codexEvent["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					openaiChunk := OpenAIStreamChunk{
						ID:      state.MessageID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Choices: []OpenAIChoice{{
							Index: 0,
							Delta: &OpenAIMessage{Content: text},
						}},
					}
					output = append(output, FormatSSE("", openaiChunk)...)
				}
			}

		case "response.done":
			openaiChunk := OpenAIStreamChunk{
				ID:      state.MessageID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Choices: []OpenAIChoice{{
					Index:        0,
					Delta:        &OpenAIMessage{},
					FinishReason: "stop",
				}},
			}
			output = append(output, FormatSSE("", openaiChunk)...)
			output = append(output, FormatDone()...)
		}
	}

	return output, nil
}
