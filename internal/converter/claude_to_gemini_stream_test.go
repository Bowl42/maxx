package converter

import (
	"encoding/json"
	"testing"
)

func TestClaudeToGeminiResponse_Stream(t *testing.T) {
	conv := &claudeToGeminiResponse{}
	state := NewTransformState()

	start := ClaudeStreamEvent{
		Type: "message_start",
		Message: &ClaudeResponse{
			ID: "msg_1",
		},
	}
	if _, err := conv.TransformChunk(FormatSSE("", start), state); err != nil {
		t.Fatalf("TransformChunk start: %v", err)
	}

	blockStart := ClaudeStreamEvent{
		Type:  "content_block_start",
		Index: 0,
		ContentBlock: &ClaudeContentBlock{
			Type: "tool_use",
			Name: "do",
			ID:   "call_1",
		},
	}
	if _, err := conv.TransformChunk(FormatSSE("", blockStart), state); err != nil {
		t.Fatalf("TransformChunk block start: %v", err)
	}

	delta := ClaudeStreamEvent{
		Type:  "content_block_delta",
		Delta: &ClaudeStreamDelta{Type: "input_json_delta", PartialJSON: `{"a":1}`},
	}
	if _, err := conv.TransformChunk(FormatSSE("", delta), state); err != nil {
		t.Fatalf("TransformChunk delta: %v", err)
	}

	stop := ClaudeStreamEvent{
		Type:  "content_block_stop",
		Index: 0,
	}
	if _, err := conv.TransformChunk(FormatSSE("", stop), state); err != nil {
		t.Fatalf("TransformChunk stop: %v", err)
	}

	done := ClaudeStreamEvent{
		Type:  "message_stop",
		Usage: &ClaudeUsage{OutputTokens: 1},
	}
	if _, err := conv.TransformChunk(FormatSSE("", done), state); err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}

	// Non-stream response path
	resp := GeminiResponse{
		Candidates: []GeminiCandidate{{
			Content: GeminiContent{
				Role: "model",
				Parts: []GeminiPart{{
					FunctionCall: &GeminiFunctionCall{Name: "do", Args: map[string]interface{}{"a": 1}},
				}},
			},
			Index: 0,
		}},
	}
	b, _ := json.Marshal(resp)
	if _, err := conv.Transform(b); err != nil {
		t.Fatalf("Transform: %v", err)
	}
}

func TestClaudeToGeminiResponse_StreamThinking(t *testing.T) {
	conv := &claudeToGeminiResponse{}
	state := NewTransformState()

	start := ClaudeStreamEvent{
		Type: "message_start",
		Message: &ClaudeResponse{
			ID: "msg_1",
		},
	}
	if _, err := conv.TransformChunk(FormatSSE("", start), state); err != nil {
		t.Fatalf("TransformChunk start: %v", err)
	}

	thinkingDelta := ClaudeStreamEvent{
		Type:  "content_block_delta",
		Delta: &ClaudeStreamDelta{Type: "thinking_delta", Thinking: "think"},
	}
	if _, err := conv.TransformChunk(FormatSSE("", thinkingDelta), state); err != nil {
		t.Fatalf("TransformChunk thinking: %v", err)
	}
}
