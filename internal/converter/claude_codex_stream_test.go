package converter

import "testing"

func TestClaudeToCodexResponse_StreamDoneEvent(t *testing.T) {
	conv := &claudeToCodexResponse{}
	state := NewTransformState()
	if _, err := conv.TransformChunk(FormatDone(), state); err != nil {
		t.Fatalf("TransformChunk done: %v", err)
	}
}
