package converter

import (
	"strings"
	"testing"
)

func TestOpenAIToClaudeResponse_StreamThinking(t *testing.T) {
	conv := &openaiToClaudeResponse{}
	state := NewTransformState()

	chunk1 := FormatSSE("", []byte(`{"id":"resp-1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"reasoning_content":"think"}}]}`))
	out1, err := conv.TransformChunk(chunk1, state)
	if err != nil {
		t.Fatalf("TransformChunk 1: %v", err)
	}
	out1Str := string(out1)
	if !strings.Contains(out1Str, `"type":"thinking"`) {
		t.Fatalf("expected thinking block start, got: %s", out1Str)
	}
	if !strings.Contains(out1Str, `"thinking_delta"`) {
		t.Fatalf("expected thinking delta, got: %s", out1Str)
	}

	chunk2 := FormatSSE("", []byte(`{"id":"resp-1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"content":"hello"}}]}`))
	out2, err := conv.TransformChunk(chunk2, state)
	if err != nil {
		t.Fatalf("TransformChunk 2: %v", err)
	}
	out2Str := string(out2)
	if !strings.Contains(out2Str, `"type":"text"`) {
		t.Fatalf("expected text block start, got: %s", out2Str)
	}
	if !strings.Contains(out2Str, `"text_delta"`) {
		t.Fatalf("expected text delta, got: %s", out2Str)
	}

	out3, err := conv.TransformChunk(FormatDone(), state)
	if err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}
	out3Str := string(out3)
	if !strings.Contains(out3Str, `"message_stop"`) {
		t.Fatalf("expected message_stop, got: %s", out3Str)
	}
}
