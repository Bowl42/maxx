package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeminiToCodexTransformFunctionResponseCallID2(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{{Role: "model", Parts: []GeminiPart{{
			FunctionCall: &GeminiFunctionCall{Name: "tool_call_1", Args: map[string]interface{}{"x": 1}},
		}, {
			FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_1", Response: map[string]interface{}{"ok": true}},
		}}}},
	}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := codexReq.Input.([]interface{})
	if !ok || len(items) < 2 {
		t.Fatalf("expected input items")
	}
}

func TestGeminiToCodexFunctionResponseOutput(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "model", Parts: []GeminiPart{{
		FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_9", Response: map[string]interface{}{"ok": true}},
	}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := codexReq.Input.([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("expected input items")
	}
}

func TestGeminiToCodexFunctionResponseResultString(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{
		FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_1", Response: map[string]interface{}{"result": "ok"}},
	}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := codexReq.Input.([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("expected input items")
	}
	item, ok := items[0].(map[string]interface{})
	if !ok || item["output"] != "ok" {
		t.Fatalf("expected result string output")
	}
}

func TestGeminiToCodexFunctionResponseResultObject(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{
		FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_1", Response: map[string]interface{}{"result": map[string]interface{}{"a": 1}}},
	}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := codexReq.Input.([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("expected input items")
	}
	item, ok := items[0].(map[string]interface{})
	if !ok || !strings.Contains(item["output"].(string), "\"a\":1") {
		t.Fatalf("expected result object output")
	}
}

func TestGeminiToCodexFunctionResponseString(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{
		FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_1", Response: "ok"},
	}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items, ok := codexReq.Input.([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("expected input items")
	}
	item, ok := items[0].(map[string]interface{})
	if !ok || item["output"] == "" {
		t.Fatalf("expected string output")
	}
}

func TestGeminiToCodexTransformFunctionResponseCallID(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{
		FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_9", Response: map[string]interface{}{"ok": true}},
	}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "function_call_output") {
		t.Fatalf("expected function_call_output")
	}
}

func TestGeminiToCodexResponseBranches(t *testing.T) {
	resp := CodexResponse{
		Status: "incomplete",
		Usage:  CodexUsage{InputTokens: 1, OutputTokens: 2, TotalTokens: 3},
		Output: []CodexOutput{{
			Type:    "message",
			Content: []interface{}{map[string]interface{}{"text": "hi"}},
		}, {
			Type:      "function_call",
			Name:      "tool",
			CallID:    "call_9",
			Arguments: `{"x":1}`,
		}},
	}
	body, _ := json.Marshal(resp)
	conv := &geminiToCodexResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "\"STOP\"") {
		t.Fatalf("expected STOP when function_call present")
	}
	if !strings.Contains(string(out), "tool_call_9") {
		t.Fatalf("expected embedded call id")
	}
}
