package converter

import (
	"encoding/json"
	"testing"
)

func TestGeminiToClaudeResponse_RemapArgs(t *testing.T) {
	resp := GeminiResponse{
		Candidates: []GeminiCandidate{{
			Content: GeminiContent{
				Role: "model",
				Parts: []GeminiPart{{
					FunctionCall: &GeminiFunctionCall{
						Name: "grep",
						Args: map[string]interface{}{"query": "foo", "paths": []interface{}{"x"}},
					},
				}},
			},
			Index: 0,
		}},
	}
	body, _ := json.Marshal(resp)
	conv := &geminiToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var got ClaudeResponse
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Content) == 0 || got.Content[0].Type != "tool_use" {
		t.Fatalf("expected tool_use")
	}
	if _, ok := got.Content[0].Input.(map[string]interface{})["pattern"]; !ok {
		t.Fatalf("expected pattern remap")
	}
	if _, ok := got.Content[0].Input.(map[string]interface{})["path"]; !ok {
		t.Fatalf("expected path remap")
	}
}
