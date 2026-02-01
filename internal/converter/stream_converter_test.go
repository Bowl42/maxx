package converter

import (
	"encoding/json"
	"testing"
)

func TestCodexToOpenAIRequest_Basic(t *testing.T) {
	req := CodexRequest{
		Model: "codex-test",
		Input: []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "hi"},
			map[string]interface{}{"type": "function_call", "id": "call_1", "name": "do", "arguments": "{}"},
			map[string]interface{}{"type": "function_call_output", "call_id": "call_1", "output": "ok"},
		},
	}
	body, _ := json.Marshal(req)
	conv := &codexToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var got OpenAIRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got.Messages))
	}
}

func TestOpenAIToCodexResponse_Stream(t *testing.T) {
	conv := &openaiToCodexResponse{}
	state := NewTransformState()

	chunk1 := FormatSSE("", []byte(`{"id":"resp_1","object":"chat.completion.chunk","created":1,"model":"gpt","choices":[{"index":0,"delta":{"content":"hi"}}]}`))
	if _, err := conv.TransformChunk(chunk1, state); err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	chunk2 := FormatDone()
	if _, err := conv.TransformChunk(chunk2, state); err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}
}

func TestCodexToOpenAIResponse_Stream(t *testing.T) {
	conv := &codexToOpenAIResponse{}
	state := NewTransformState()

	created := map[string]interface{}{
		"type": "response.created",
		"response": map[string]interface{}{
			"id": "resp_1",
		},
	}
	if _, err := conv.TransformChunk(FormatSSE("", created), state); err != nil {
		t.Fatalf("TransformChunk created: %v", err)
	}
	delta := map[string]interface{}{
		"type": "response.output_item.delta",
		"delta": map[string]interface{}{
			"text": "hi",
		},
	}
	if _, err := conv.TransformChunk(FormatSSE("", delta), state); err != nil {
		t.Fatalf("TransformChunk delta: %v", err)
	}
	done := map[string]interface{}{
		"type": "response.done",
	}
	if _, err := conv.TransformChunk(FormatSSE("", done), state); err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}
}
