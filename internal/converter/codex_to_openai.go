package converter

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/awsl-project/maxx/internal/debug"
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

	converted, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, err
	}

	if debug.Enabled() {
		_, _ = os.Stdout.Write([]byte("=========Codex 转 OpenAI 请求>>>>>>>>\n"))
		_, _ = os.Stdout.Write(converted)
	}

	return converted, nil
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
		if eventType == "" && event.Event != "" {
			eventType = event.Event
		}

		switch eventType {
		case "response.created":
			if resp, ok := codexEvent["response"].(map[string]interface{}); ok {
				state.MessageID, _ = resp["id"].(string)
				if model, ok := resp["model"].(string); ok && model != "" {
					state.Model = model
				}
				if created := parseUnixSeconds(resp["created_at"]); created > 0 {
					state.CreatedAt = created
				}
			}
			output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
				"role":    "assistant",
				"content": "",
			}, nil)...)

		case "response.output_text.delta", "response.output_item.delta":
			var text string
			switch delta := codexEvent["delta"].(type) {
			case string:
				text = delta
			case map[string]interface{}:
				if t, ok := delta["text"].(string); ok {
					text = t
				}
				if args, ok := delta["arguments"].(string); ok && args != "" {
					toolIndex := -1
					if id, ok := codexEvent["item_id"].(string); ok && id != "" {
						toolIndex = findToolCallIndex(state, id)
					} else if id, ok := codexEvent["id"].(string); ok && id != "" {
						toolIndex = findToolCallIndex(state, id)
					} else if id, ok := codexEvent["call_id"].(string); ok && id != "" {
						toolIndex = findToolCallIndex(state, id)
					}
					if toolIndex < 0 {
						if state.CurrentIndex > 0 {
							toolIndex = state.CurrentIndex - 1
						} else {
							toolIndex = 0
						}
					}
					var id string
					var name string
					if tc, ok := state.ToolCalls[toolIndex]; ok && tc != nil {
						id = tc.ID
						name = tc.Name
					}
					fn := map[string]interface{}{
						"arguments": args,
					}
					if name != "" {
						fn["name"] = name
					}
					tool := map[string]interface{}{
						"index":    toolIndex,
						"type":     "function",
						"function": fn,
					}
					if id != "" {
						tool["id"] = id
					}
					output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
						"tool_calls": []interface{}{tool},
					}, nil)...)
				}
			}
			if text != "" {
				output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
					"content": text,
				}, nil)...)
			}

		case "response.output_item.added":
			if item, ok := codexEvent["item"].(map[string]interface{}); ok {
				itemType, _ := item["type"].(string)
				if itemType == "function_call" {
					id, _ := item["id"].(string)
					if id == "" {
						id, _ = item["call_id"].(string)
					}
					name, _ := item["name"].(string)
					args, _ := item["arguments"].(string)
					if state.ToolCalls == nil {
						state.ToolCalls = make(map[int]*ToolCallState)
					}
					state.ToolCalls[state.CurrentIndex] = &ToolCallState{ID: id, Name: name}
					tool := map[string]interface{}{
						"index": state.CurrentIndex,
						"type":  "function",
						"function": map[string]interface{}{
							"name":      name,
							"arguments": args,
						},
					}
					if id != "" {
						tool["id"] = id
					}
					output = append(output, formatOpenAIStreamChunk(state, map[string]interface{}{
						"tool_calls": []interface{}{tool},
					}, nil)...)
					state.CurrentIndex++
				}
			}

		case "response.done", "response.completed":
			if resp, ok := codexEvent["response"].(map[string]interface{}); ok {
				updateCodexUsageFromResponse(state, resp)
			}
			usage := buildOpenAIUsageFromState(state)
			stop := "stop"
			output = append(output, formatOpenAIStreamChunkWithUsage(state, map[string]interface{}{}, &stop, usage)...)
			output = append(output, FormatDone()...)
		}
	}

	return output, nil
}

func findToolCallIndex(state *TransformState, id string) int {
	if state == nil || id == "" || state.ToolCalls == nil {
		return -1
	}
	for idx, tc := range state.ToolCalls {
		if tc != nil && tc.ID == id {
			return idx
		}
	}
	return -1
}

func updateCodexUsageFromResponse(state *TransformState, resp map[string]interface{}) {
	if state == nil || resp == nil {
		return
	}
	if model, ok := resp["model"].(string); ok && model != "" {
		state.Model = model
	}
	if created := parseUnixSeconds(resp["created_at"]); created > 0 {
		state.CreatedAt = created
	}
	usage, ok := resp["usage"].(map[string]interface{})
	if !ok {
		return
	}
	if state.Usage == nil {
		state.Usage = &Usage{}
	}
	if v := parseIntValue(usage["input_tokens"]); v > 0 {
		state.Usage.InputTokens = v
	}
	if v := parseIntValue(usage["output_tokens"]); v > 0 {
		state.Usage.OutputTokens = v
	}
	if v := parseIntValue(usage["cache_read_input_tokens"]); v > 0 {
		state.Usage.CacheRead = v
	}
	if details, ok := usage["input_tokens_details"].(map[string]interface{}); ok {
		if v := parseIntValue(details["cached_tokens"]); v > 0 {
			state.Usage.CacheRead = v
		}
	}
	if details, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
		if v := parseIntValue(details["cached_tokens"]); v > 0 {
			state.Usage.CacheRead = v
		}
	}
}

func buildOpenAIUsageFromState(state *TransformState) map[string]interface{} {
	if state == nil || state.Usage == nil {
		return nil
	}
	inputTokens := state.Usage.InputTokens
	outputTokens := state.Usage.OutputTokens
	cacheRead := state.Usage.CacheRead
	if inputTokens == 0 && outputTokens == 0 && cacheRead == 0 {
		return nil
	}
	usage := map[string]interface{}{
		"prompt_tokens":     inputTokens,
		"completion_tokens": outputTokens,
		"total_tokens":      inputTokens + outputTokens,
		"input_tokens":      inputTokens,
		"output_tokens":     outputTokens,
	}
	if cacheRead > 0 {
		usage["cache_read_input_tokens"] = cacheRead
		usage["prompt_tokens_details"] = map[string]interface{}{"cached_tokens": cacheRead}
		usage["input_tokens_details"] = map[string]interface{}{"cached_tokens": cacheRead}
	}
	return usage
}

func formatOpenAIStreamChunkWithUsage(state *TransformState, delta map[string]interface{}, finishReason *string, usage map[string]interface{}) []byte {
	if delta == nil {
		delta = map[string]interface{}{}
	}
	id, model, created := ensureOpenAIStreamMeta(state)
	choice := map[string]interface{}{
		"index":         0,
		"delta":         delta,
		"finish_reason": nil,
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	}
	chunk := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []interface{}{choice},
	}
	if usage != nil {
		chunk["usage"] = usage
	}
	return FormatSSE("", chunk)
}

func parseIntValue(v interface{}) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	case string:
		if n, err := strconv.Atoi(t); err == nil {
			return n
		}
	}
	return 0
}
