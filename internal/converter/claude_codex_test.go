package converter

import (
	"encoding/json"
	"testing"
)

func TestClaudeToCodexRequest_Basic(t *testing.T) {
	req := ClaudeRequest{
		System: "sys",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []ClaudeContentBlock{
				{Type: "text", Text: "hi"},
				{Type: "tool_use", ID: "call_1", Name: "do", Input: map[string]interface{}{"a": 1}},
			},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToCodexRequest{}
	out, err := conv.Transform(body, "codex-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var got CodexRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !codexInputHasRoleText(got.Input, "developer", "sys") {
		t.Fatalf("expected system message")
	}
	if got.Input == nil {
		t.Fatalf("expected input")
	}
}

func TestCodexToClaudeResponse_Basic(t *testing.T) {
	resp := CodexResponse{
		ID:     "resp_1",
		Model:  "codex-test",
		Status: "completed",
		Usage:  CodexUsage{InputTokens: 1, OutputTokens: 1},
		Output: []CodexOutput{{
			Type:    "message",
			Content: "hi",
		}, {
			Type:      "function_call",
			ID:        "call_1",
			Name:      "do",
			Arguments: `{"a":1}`,
		}},
	}
	body, _ := json.Marshal(resp)
	conv := &codexToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var got ClaudeResponse
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.StopReason != "tool_use" {
		t.Fatalf("expected tool_use stop_reason")
	}
}

func TestCodexToClaudeResponse_Stream(t *testing.T) {
	conv := &codexToClaudeResponse{}
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
