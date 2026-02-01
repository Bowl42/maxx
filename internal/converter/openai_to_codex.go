package converter

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/awsl-project/maxx/internal/domain"
)

func init() {
	RegisterConverter(domain.ClientTypeOpenAI, domain.ClientTypeCodex, &openaiToCodexRequest{}, &openaiToCodexResponse{})
}

type openaiToCodexRequest struct{}
type openaiToCodexResponse struct{}

func (c *openaiToCodexRequest) Transform(body []byte, model string, stream bool) ([]byte, error) {
	userAgent := ExtractCodexUserAgent(body)
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

	if req.ReasoningEffort != "" {
		effort := strings.TrimSpace(req.ReasoningEffort)
		codexReq.Reasoning = &CodexReasoning{
			Effort: effort,
		}
	}
	trueVal := true
	codexReq.ParallelToolCalls = &trueVal
	codexReq.Include = []string{"reasoning.encrypted_content"}

	// Convert messages to input
	shortMap := map[string]string{}
	if len(req.Tools) > 0 {
		var names []string
		for _, tool := range req.Tools {
			if tool.Type == "function" && tool.Function.Name != "" {
				names = append(names, tool.Function.Name)
			}
		}
		if len(names) > 0 {
			shortMap = buildShortNameMap(names)
		}
	}

	var input []CodexInputItem
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "system" {
			role = "developer"
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
			Role: role,
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
			name := tc.Function.Name
			if short, ok := shortMap[name]; ok {
				name = short
			} else {
				name = shortenNameIfNeeded(name)
			}
			input = append(input, CodexInputItem{
				Type:      "function_call",
				ID:        tc.ID,
				CallID:    tc.ID,
				Name:      name,
				Role:      "assistant",
				Arguments: tc.Function.Arguments,
			})
		}
	}
	codexReq.Input = input

	// Convert tools
	for _, tool := range req.Tools {
		name := tool.Function.Name
		if short, ok := shortMap[name]; ok {
			name = short
		} else {
			name = shortenNameIfNeeded(name)
		}
		codexReq.Tools = append(codexReq.Tools, CodexTool{
			Type:        "function",
			Name:        name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	_, instructions := CodexInstructionsForModel(model, "", userAgent)
	if GetCodexInstructionsEnabled() {
		codexReq.Instructions = instructions
	}
	if codexReq.Reasoning == nil {
		codexReq.Reasoning = &CodexReasoning{Effort: "medium"}
	}
	if codexReq.Reasoning.Effort == "" {
		codexReq.Reasoning.Effort = "medium"
	}
	if codexReq.Reasoning.Summary == "" {
		codexReq.Reasoning.Summary = "auto"
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
