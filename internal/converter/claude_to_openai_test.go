package converter

import (
	"encoding/json"
	"testing"
)

func TestClaudeToOpenAIRequest_ThinkingToReasoning(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-test",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []ClaudeContentBlock{
				{Type: "thinking", Thinking: "step one"},
				{Type: "text", Text: "hello"},
			},
		}},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	conv := &claudeToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	var got OpenAIRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	msg := got.Messages[0]
	if msg.ReasoningContent != "step one" {
		t.Fatalf("expected reasoning_content 'step one', got %#v", msg.ReasoningContent)
	}
	if msg.Content != "hello" {
		t.Fatalf("expected content 'hello', got %#v", msg.Content)
	}
}

func TestClaudeToOpenAIRequest_ToolResultOrder(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-test",
		Messages: []ClaudeMessage{
			{
				Role: "assistant",
				Content: []ClaudeContentBlock{
					{Type: "tool_use", ID: "call-1", Name: "lookup", Input: map[string]interface{}{"q": "foo"}},
				},
			},
			{
				Role: "user",
				Content: []ClaudeContentBlock{
					{Type: "tool_result", ToolUseID: "call-1", Content: "ok"},
					{Type: "text", Text: "next"},
				},
			},
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	conv := &claudeToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	var got OpenAIRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got.Messages))
	}
	if got.Messages[1].Role != "tool" {
		t.Fatalf("expected tool message at index 1, got role %q", got.Messages[1].Role)
	}
	if got.Messages[2].Role != "user" {
		t.Fatalf("expected user message at index 2, got role %q", got.Messages[2].Role)
	}
	if got.Messages[1].ToolCallID != "call-1" {
		t.Fatalf("expected tool_call_id 'call-1', got %q", got.Messages[1].ToolCallID)
	}
	if got.Messages[1].Content != "ok" {
		t.Fatalf("expected tool content 'ok', got %#v", got.Messages[1].Content)
	}
}
