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
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var req OpenAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

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

	instructions := ""
	if rawInstructions, ok := raw["instructions"].(string); ok && strings.TrimSpace(rawInstructions) != "" {
		instructions = rawInstructions
	}

	messages := req.Messages
	if instructions == "" {
		for i, msg := range messages {
			if msg.Role == "system" || msg.Role == "developer" {
				instructions = stringifyContent(msg.Content)
				messages = append(messages[:i], messages[i+1:]...)
				break
			}
		}
	}
	if instructions == "" {
		instructions = CodexInstructionsForModel(model, userAgent)
	}

	var inputItems []map[string]interface{}
	for _, msg := range messages {
		role := msg.Role
		if role == "system" {
			role = "developer"
		}

		if msg.Role == "tool" {
			output := stringifyContent(msg.Content)
			inputItems = append(inputItems, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": msg.ToolCallID,
				"output":  output,
			})
			continue
		}

		if parts := codexContentParts(role, msg.Content); len(parts) > 0 {
			inputItems = append(inputItems, map[string]interface{}{
				"type":    "message",
				"role":    role,
				"content": parts,
			})
		}

		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			if short, ok := shortMap[name]; ok {
				name = short
			} else {
				name = shortenNameIfNeeded(name)
			}
			callID := tc.ID
			inputItems = append(inputItems, map[string]interface{}{
				"type":      "function_call",
				"id":        callID,
				"call_id":   callID,
				"name":      name,
				"arguments": tc.Function.Arguments,
			})
		}
	}

	// Convert tools
	tools := []CodexTool{}
	for _, tool := range req.Tools {
		name := tool.Function.Name
		if short, ok := shortMap[name]; ok {
			name = short
		} else {
			name = shortenNameIfNeeded(name)
		}
		tools = append(tools, CodexTool{
			Type:        "function",
			Name:        name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	maxOutputTokens := req.MaxTokens
	if req.MaxCompletionTokens > 0 && req.MaxTokens == 0 {
		maxOutputTokens = req.MaxCompletionTokens
	}

	reasoningEffort := strings.TrimSpace(req.ReasoningEffort)
	if reasoningEffort == "" {
		reasoningEffort = "high"
	}

	toolChoice := req.ToolChoice
	if toolChoice == nil {
		toolChoice = "auto"
	}

	payload := map[string]interface{}{
		"model":               model,
		"instructions":        instructions,
		"input":               inputItems,
		"tools":               tools,
		"tool_choice":         toolChoice,
		"parallel_tool_calls": true,
		"reasoning": map[string]interface{}{
			"effort":  reasoningEffort,
			"summary": "auto",
		},
		"store":  false,
		"stream": stream,
		"include": []string{
			"reasoning.encrypted_content",
		},
	}

	if maxOutputTokens > 0 {
		payload["max_output_tokens"] = maxOutputTokens
	}
	if req.TopP != nil {
		payload["top_p"] = *req.TopP
	}
	if prevID, ok := raw["previous_response_id"].(string); ok && prevID != "" {
		payload["previous_response_id"] = prevID
	}
	if cacheKey, ok := raw["prompt_cache_key"].(string); ok && cacheKey != "" {
		payload["prompt_cache_key"] = cacheKey
	}

	return json.Marshal(payload)
}

func codexContentParts(role string, content interface{}) []map[string]interface{} {
	partType := "input_text"
	if role == "assistant" {
		partType = "output_text"
	}
	switch c := content.(type) {
	case string:
		if strings.TrimSpace(c) == "" {
			return nil
		}
		return []map[string]interface{}{
			{
				"type": partType,
				"text": c,
			},
		}
	case []interface{}:
		var parts []map[string]interface{}
		for _, part := range c {
			if m, ok := part.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok && strings.TrimSpace(text) != "" {
					if typ, _ := m["type"].(string); typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
						continue
					}
					parts = append(parts, map[string]interface{}{
						"type": partType,
						"text": text,
					})
				}
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return parts
	case map[string]interface{}:
		if text, ok := c["text"].(string); ok && strings.TrimSpace(text) != "" {
			return []map[string]interface{}{
				{
					"type": partType,
					"text": text,
				},
			}
		}
	}
	return nil
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
