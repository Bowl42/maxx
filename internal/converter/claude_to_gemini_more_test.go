package converter

import (
	"encoding/json"
	"testing"
)

func TestMergeAdjacentRoles(t *testing.T) {
	in := []GeminiContent{
		{Role: "user", Parts: []GeminiPart{{Text: "a"}}},
		{Role: "user", Parts: []GeminiPart{{Text: "b"}}},
		{Role: "model", Parts: []GeminiPart{{Text: "c"}}},
	}
	out := mergeAdjacentRoles(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(out))
	}
	if len(out[0].Parts) != 2 {
		t.Fatalf("expected merged parts")
	}
}

func TestClaudeToGeminiRequest_ToolResultNameFallback(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Messages: []ClaudeMessage{{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "tool_result", "tool_use_id": "call_123", "content": "ok"},
			},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-opus-4-5-thinking", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var got GeminiRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil && p.FunctionResponse.Name == "call_123" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected fallback name to tool_use_id")
	}
}
