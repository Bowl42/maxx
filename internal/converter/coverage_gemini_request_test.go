package converter

import (
	"encoding/json"
	"testing"
)

func TestCodexToGeminiRequest(t *testing.T) {
	req := CodexRequest{
		Instructions: "sys",
		Input: []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "hi"},
			map[string]interface{}{"type": "function_call", "name": "tool", "call_id": "call_1", "arguments": `{"x":1}`},
			map[string]interface{}{"type": "function_call_output", "call_id": "call_1", "output": "ok"},
		},
		Reasoning: &CodexReasoning{Effort: "auto"},
		Tools: []CodexTool{{
			Type:        "function",
			Name:        "tool",
			Description: "d",
			Parameters:  map[string]interface{}{"type": "object"},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &codexToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var geminiReq GeminiRequest
	if err := json.Unmarshal(out, &geminiReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if geminiReq.SystemInstruction == nil {
		t.Fatalf("systemInstruction missing")
	}
	if len(geminiReq.Contents) == 0 {
		t.Fatalf("contents missing")
	}
	if geminiReq.GenerationConfig == nil || geminiReq.GenerationConfig.ThinkingConfig == nil {
		t.Fatalf("thinking config missing")
	}
}
