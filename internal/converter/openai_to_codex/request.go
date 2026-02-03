package openai_to_codex

import (
	"encoding/json"
	"strings"

	"github.com/awsl-project/maxx/internal/converter"
)

type Request struct{}

func (c *Request) Transform(body []byte, model string, stream bool) ([]byte, error) {
	userAgent := converter.ExtractCodexUserAgent(body)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var req converter.OpenAIRequest
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
			shortMap = converter.BuildShortNameMap(names)
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
				instructions = converter.StringifyContent(msg.Content)
				messages = append(messages[:i], messages[i+1:]...)
				break
			}
		}
	}
	if instructions == "" {
		instructions = converter.CodexInstructionsForModel(model, userAgent)
	}

	var inputItems []map[string]interface{}
	for _, msg := range messages {
		role := msg.Role
		if role == "system" {
			role = "developer"
		}

		if msg.Role == "tool" {
			output := converter.StringifyContent(msg.Content)
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
				name = converter.ShortenNameIfNeeded(name)
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
	tools := []converter.CodexTool{}
	for _, tool := range req.Tools {
		name := tool.Function.Name
		if short, ok := shortMap[name]; ok {
			name = short
		} else {
			name = converter.ShortenNameIfNeeded(name)
		}
		tools = append(tools, converter.CodexTool{
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
