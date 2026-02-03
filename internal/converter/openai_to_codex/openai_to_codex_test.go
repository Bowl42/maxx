package openai_to_codex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/awsl-project/maxx/internal/converter"
)

func TestRequestTransform_SystemInstructionsAndTools(t *testing.T) {
	longName := "tool_" + strings.Repeat("x", 80)
	topP := 0.9
	req := converter.OpenAIRequest{
		Messages: []converter.OpenAIMessage{
			{Role: "system", Content: "system instructions"},
			{Role: "user", Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hello"},
				map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "x"}},
			}},
			{Role: "assistant", Content: map[string]interface{}{"text": "assistant reply"}, ToolCalls: []converter.OpenAIToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: converter.OpenAIFunctionCall{
					Name:      longName,
					Arguments: `{"x":1}`,
				},
			}}},
			{Role: "tool", Content: "tool result", ToolCallID: "call_1"},
		},
		Tools: []converter.OpenAITool{{
			Type: "function",
			Function: converter.OpenAIFunction{
				Name:        longName,
				Description: "desc",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
		MaxCompletionTokens: 321,
		TopP:                &topP,
	}
	body, _ := json.Marshal(req)
	out, err := (&Request{}).Transform(body, "gpt-5.2-codex", true)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	payload := decodeMap(t, out)
	if payload["instructions"] != "system instructions" {
		t.Fatalf("expected system instructions, got: %#v", payload["instructions"])
	}
	if payload["model"] != "gpt-5.2-codex" {
		t.Fatalf("expected model passthrough")
	}
	if payload["stream"] != true {
		t.Fatalf("expected stream true")
	}
	if payload["tool_choice"] != "auto" {
		t.Fatalf("expected default tool_choice auto")
	}
	if payload["max_output_tokens"].(float64) != 321 {
		t.Fatalf("expected max_output_tokens from max_completion_tokens")
	}
	if payload["top_p"].(float64) != topP {
		t.Fatalf("expected top_p passthrough")
	}

	reasoning := payload["reasoning"].(map[string]interface{})
	if reasoning["effort"] != "high" || reasoning["summary"] != "auto" {
		t.Fatalf("expected default reasoning config, got: %#v", reasoning)
	}

	tools := payload["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got: %d", len(tools))
	}
	tool := tools[0].(map[string]interface{})
	shortName := converter.ShortenNameIfNeeded(longName)
	if tool["name"] != shortName {
		t.Fatalf("expected shortened tool name, got: %#v", tool["name"])
	}

	input := payload["input"].([]interface{})
	if len(input) != 4 {
		t.Fatalf("expected 4 input items, got: %d", len(input))
	}

	userMsg := input[0].(map[string]interface{})
	if userMsg["type"] != "message" || userMsg["role"] != "user" {
		t.Fatalf("expected user message item, got: %#v", userMsg)
	}
	userContent := userMsg["content"].([]interface{})
	if len(userContent) != 1 {
		t.Fatalf("expected filtered user content parts, got: %#v", userContent)
	}
	userPart := userContent[0].(map[string]interface{})
	if userPart["type"] != "input_text" || userPart["text"] != "hello" {
		t.Fatalf("unexpected user content part: %#v", userPart)
	}

	assistantMsg := input[1].(map[string]interface{})
	if assistantMsg["type"] != "message" || assistantMsg["role"] != "assistant" {
		t.Fatalf("expected assistant message item, got: %#v", assistantMsg)
	}
	assistantContent := assistantMsg["content"].([]interface{})
	assistantPart := assistantContent[0].(map[string]interface{})
	if assistantPart["type"] != "output_text" || assistantPart["text"] != "assistant reply" {
		t.Fatalf("unexpected assistant content part: %#v", assistantPart)
	}

	call := input[2].(map[string]interface{})
	if call["type"] != "function_call" || call["name"] != shortName || call["call_id"] != "call_1" {
		t.Fatalf("unexpected function_call item: %#v", call)
	}
	if call["arguments"] != `{"x":1}` {
		t.Fatalf("unexpected function_call arguments: %#v", call["arguments"])
	}

	output := input[3].(map[string]interface{})
	if output["type"] != "function_call_output" || output["call_id"] != "call_1" || output["output"] != "tool result" {
		t.Fatalf("unexpected function_call_output item: %#v", output)
	}
}

func TestRequestTransform_RawInstructionsAndParams(t *testing.T) {
	req := converter.OpenAIRequest{
		Messages: []converter.OpenAIMessage{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hi"},
		},
		MaxTokens:       50,
		ReasoningEffort: "  medium  ",
		ToolChoice:      "required",
	}
	body, _ := json.Marshal(req)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	raw["instructions"] = "raw instructions"
	raw["previous_response_id"] = "resp_1"
	raw["prompt_cache_key"] = "cache_1"
	body, _ = json.Marshal(raw)

	out, err := (&Request{}).Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	payload := decodeMap(t, out)

	if payload["instructions"] != "raw instructions" {
		t.Fatalf("expected raw instructions")
	}
	if payload["tool_choice"] != "required" {
		t.Fatalf("expected tool_choice passthrough")
	}
	if payload["max_output_tokens"].(float64) != 50 {
		t.Fatalf("expected max_output_tokens from max_tokens")
	}
	reasoning := payload["reasoning"].(map[string]interface{})
	if reasoning["effort"] != "medium" {
		t.Fatalf("expected trimmed reasoning effort, got: %#v", reasoning["effort"])
	}
	if payload["previous_response_id"] != "resp_1" {
		t.Fatalf("expected previous_response_id passthrough")
	}
	if payload["prompt_cache_key"] != "cache_1" {
		t.Fatalf("expected prompt_cache_key passthrough")
	}

	input := payload["input"].([]interface{})
	if len(input) != 2 {
		t.Fatalf("expected system message retained, got: %#v", input)
	}
	systemMsg := input[0].(map[string]interface{})
	if systemMsg["role"] != "developer" {
		t.Fatalf("expected system message role remap, got: %#v", systemMsg["role"])
	}
}

func TestResponseTransform_Basic(t *testing.T) {
	resp := converter.OpenAIResponse{
		ID:      "resp_1",
		Created: 123,
		Model:   "gpt-test",
		Usage:   converter.OpenAIUsage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
		Choices: []converter.OpenAIChoice{{
			Index: 0,
			Message: &converter.OpenAIMessage{
				Content: "hello",
				ToolCalls: []converter.OpenAIToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: converter.OpenAIFunctionCall{
						Name:      "tool",
						Arguments: `{"a":1}`,
					},
				}},
			},
		}},
	}
	body, _ := json.Marshal(resp)
	out, err := (&Response{}).Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexResp converter.CodexResponse
	if err := json.Unmarshal(out, &codexResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if codexResp.ID != "resp_1" || codexResp.Object != "response" || codexResp.Status != "completed" {
		t.Fatalf("unexpected response metadata: %#v", codexResp)
	}
	if codexResp.Usage.InputTokens != 1 || codexResp.Usage.OutputTokens != 2 || codexResp.Usage.TotalTokens != 3 {
		t.Fatalf("unexpected usage: %#v", codexResp.Usage)
	}
	if len(codexResp.Output) != 2 {
		t.Fatalf("expected 2 output items, got: %d", len(codexResp.Output))
	}
	if codexResp.Output[0].Type != "message" || codexResp.Output[0].Content != "hello" {
		t.Fatalf("unexpected message output: %#v", codexResp.Output[0])
	}
	if codexResp.Output[1].Type != "function_call" || codexResp.Output[1].Name != "tool" || codexResp.Output[1].CallID != "call_1" {
		t.Fatalf("unexpected function_call output: %#v", codexResp.Output[1])
	}
}

func TestResponseTransformChunk_DeltasAndDone(t *testing.T) {
	conv := &Response{}
	state := converter.NewTransformState()

	chunk := converter.FormatSSE("", []byte(`{"id":"resp-1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"content":"hi"}}]}`))
	out, err := conv.TransformChunk(chunk, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	events, _ := converter.ParseSSE(string(out))
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got: %d", len(events))
	}
	types := eventTypes(t, events)
	if !contains(types, "response.created") || !contains(types, "response.output_item.delta") {
		t.Fatalf("unexpected event types: %#v", types)
	}
	deltaEvent := findEvent(t, events, "response.output_item.delta")
	delta := deltaEvent["delta"].(map[string]interface{})
	if delta["text"] != "hi" || delta["type"] != "text" {
		t.Fatalf("unexpected delta payload: %#v", delta)
	}

	doneChunk := converter.FormatSSE("", []byte(`{"id":"resp-1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"finish_reason":"stop"}]}`))
	out2, err := conv.TransformChunk(doneChunk, state)
	if err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}
	events2, _ := converter.ParseSSE(string(out2))
	if !contains(eventTypes(t, events2), "response.done") {
		t.Fatalf("expected response.done event")
	}
}

func decodeMap(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func eventTypes(t *testing.T, events []converter.SSEEvent) []string {
	t.Helper()
	types := make([]string, 0, len(events))
	for _, evt := range events {
		payload := decodeEvent(t, evt)
		typ, _ := payload["type"].(string)
		types = append(types, typ)
	}
	return types
}

func findEvent(t *testing.T, events []converter.SSEEvent, want string) map[string]interface{} {
	t.Helper()
	for _, evt := range events {
		payload := decodeEvent(t, evt)
		if typ, _ := payload["type"].(string); typ == want {
			return payload
		}
	}
	t.Fatalf("event %q not found", want)
	return nil
}

func decodeEvent(t *testing.T, evt converter.SSEEvent) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(evt.Data, &payload); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return payload
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
