package converter

import (
	"encoding/json"
	"time"

	"github.com/Bowl42/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeOpenAI, domain.ClientTypeCodex, &openaiToCodexRequest{}, &openaiToCodexResponse{})
}

type openaiToCodexRequest struct{}
type openaiToCodexResponse struct{}

func (c *openaiToCodexRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	var req OpenAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	codexReq := CodexRequest{
		Model:           model,
		Stream:          stream,
		MaxOutputTokens: req.MaxTokens,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
	}

	if req.MaxCompletionTokens > 0 && req.MaxTokens == 0 {
		codexReq.MaxOutputTokens = req.MaxCompletionTokens
	}

	// Convert messages to input
	var input []CodexInputItem
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Convert to instructions
			if content, ok := msg.Content.(string); ok {
				codexReq.Instructions = content
			}
			continue
		}

		if msg.Role == "tool" {
			// Tool response
			contentStr, _ := msg.Content.(string)
			input = append(input, CodexInputItem{
				Type:   "function_call_output",
				CallID: msg.ToolCallID,
				Output: contentStr,
			})
			continue
		}

		item := CodexInputItem{
			Type: "message",
			Role: msg.Role,
		}

		switch content := msg.Content.(type) {
		case string:
			item.Content = content
		case []interface{}:
			var textContent string
			for _, part := range content {
				if m, ok := part.(map[string]interface{}); ok {
					if m["type"] == "text" {
						if text, ok := m["text"].(string); ok {
							textContent += text
						}
					}
				}
			}
			item.Content = textContent
		}

		input = append(input, item)

		// Handle tool calls
		for _, tc := range msg.ToolCalls {
			input = append(input, CodexInputItem{
				Type:      "function_call",
				ID:        tc.ID,
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Role:      "assistant",
				Arguments: tc.Function.Arguments,
			})
		}
	}
	codexReq.Input = input

	// Convert tools
	for _, tool := range req.Tools {
		codexReq.Tools = append(codexReq.Tools, CodexTool{
			Type:        "function",
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	return json.Marshal(codexReq)
}

func (c *openaiToCodexResponse) Transform(body []byte) ([]byte, error) {
	var resp OpenAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	codexResp := CodexResponse{
		ID:        resp.ID,
		Object:    "response",
		CreatedAt: resp.Created,
		Model:     resp.Model,
		Status:    "completed",
		Usage: CodexUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.Message != nil {
			if content, ok := choice.Message.Content.(string); ok && content != "" {
				codexResp.Output = append(codexResp.Output, CodexOutput{
					Type:    "message",
					Role:    "assistant",
					Content: content,
				})
			}
			for _, tc := range choice.Message.ToolCalls {
				codexResp.Output = append(codexResp.Output, CodexOutput{
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

func (c *openaiToCodexResponse) TransformChunk(chunk []byte, state *TransformState) ([]byte, error) {
	events, remaining := ParseSSE(state.Buffer + string(chunk))
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
			output = append(output, FormatSSE("", codexEvent)...)
			continue
		}

		var openaiChunk OpenAIStreamChunk
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
			output = append(output, FormatSSE("", codexEvent)...)
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
					output = append(output, FormatSSE("", codexEvent)...)
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
				output = append(output, FormatSSE("", codexEvent)...)
			}
		}
	}

	return output, nil
}
