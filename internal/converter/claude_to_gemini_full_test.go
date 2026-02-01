package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeToGeminiRequest_FullFlow(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		System: []interface{}{
			map[string]interface{}{"type": "text", "text": "sys"},
		},
		Thinking: map[string]interface{}{
			"type":          "enabled",
			"budget_tokens": float64(1024),
		},
		Messages: []ClaudeMessage{
			{
				Role: "assistant",
				Content: []interface{}{
					map[string]interface{}{"type": "thinking", "thinking": "t1", "signature": "signature_12345"},
					map[string]interface{}{"type": "tool_use", "id": "call_1", "name": "do", "input": map[string]interface{}{"a": 1, "type": "string"}},
				},
			},
			{
				Role: "user",
				Content: []interface{}{
					map[string]interface{}{"type": "tool_result", "tool_use_id": "call_1", "content": []interface{}{
						map[string]interface{}{"type": "text", "text": "ok"},
					}},
				},
			},
		},
		Tools: []ClaudeTool{{
			Type: "web_search_20250305",
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
	if got.SystemInstruction == nil || got.SystemInstruction.Role != "user" {
		t.Fatalf("expected systemInstruction user role")
	}
	if got.GenerationConfig == nil || got.GenerationConfig.ThinkingConfig == nil {
		t.Fatalf("expected thinkingConfig")
	}
	if len(got.Contents) == 0 {
		t.Fatalf("expected contents")
	}
	foundToolResp := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil && p.FunctionResponse.ID == "call_1" {
				if respMap, ok := p.FunctionResponse.Response.(map[string]interface{}); ok {
					if result, ok := respMap["result"].(string); ok {
						if !strings.Contains(result, "ok") {
							t.Fatalf("unexpected tool result")
						}
					}
				}
				foundToolResp = true
			}
		}
	}
	if !foundToolResp {
		t.Fatalf("expected function response")
	}
}
