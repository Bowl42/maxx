package converter

import (
	"encoding/json"
	"testing"
)

func TestClaudeToOpenAIResponse_Stream(t *testing.T) {
	conv := &claudeToOpenAIResponse{}
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
	delta := ClaudeStreamEvent{
		Type:  "content_block_delta",
		Delta: &ClaudeStreamDelta{Type: "text_delta", Text: "hi"},
	}
	if _, err := conv.TransformChunk(FormatSSE("", delta), state); err != nil {
		t.Fatalf("TransformChunk delta: %v", err)
	}
	stop := ClaudeStreamEvent{
		Type:  "message_stop",
		Delta: &ClaudeStreamDelta{StopReason: "end_turn"},
	}
	if _, err := conv.TransformChunk(FormatSSE("", stop), state); err != nil {
		t.Fatalf("TransformChunk stop: %v", err)
	}
}

func TestCodexToGeminiResponse_Stream(t *testing.T) {
	conv := &codexToGeminiResponse{}
	state := NewTransformState()

	chunk := GeminiStreamChunk{
		Candidates: []GeminiCandidate{{
			Content: GeminiContent{
				Role: "model",
				Parts: []GeminiPart{{
					Text: "hi",
				}},
			},
		}},
		UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 1, CandidatesTokenCount: 1},
	}
	b, _ := json.Marshal(chunk)
	if _, err := conv.TransformChunk(FormatSSE("", b), state); err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if _, err := conv.TransformChunk(FormatDone(), state); err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}
}
