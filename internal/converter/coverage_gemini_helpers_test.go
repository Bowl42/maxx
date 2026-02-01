package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodexGeminiAndGeminiCodex(t *testing.T) {
	if mapCodexRoleToGemini("system") != "model" {
		t.Fatalf("map codex role")
	}

	geminiResp := GeminiResponse{
		UsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 1, CandidatesTokenCount: 2, TotalTokenCount: 3},
		Candidates: []GeminiCandidate{{
			Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}, {
				FunctionCall: &GeminiFunctionCall{Name: "tool_call_1", Args: map[string]interface{}{"x": 1}},
			}}},
		}},
	}
	geminiBody, _ := json.Marshal(geminiResp)
	conv := &codexToGeminiResponse{}
	codexOut, err := conv.Transform(geminiBody)
	if err != nil {
		t.Fatalf("Transform codex: %v", err)
	}
	var codexResp CodexResponse
	if err := json.Unmarshal(codexOut, &codexResp); err != nil {
		t.Fatalf("unmarshal codex: %v", err)
	}
	if len(codexResp.Output) == 0 {
		t.Fatalf("codex output missing")
	}

	geminiReq := GeminiRequest{
		GenerationConfig:  &GeminiGenerationConfig{ThinkingConfig: &GeminiThinkingConfig{ThinkingLevel: "low"}},
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: "sys"}}},
		Contents: []GeminiContent{{
			Role: "user",
			Parts: []GeminiPart{{Text: "hi"}, {
				FunctionCall: &GeminiFunctionCall{Name: "tool_call_1", Args: map[string]interface{}{"x": 1}},
			}, {
				FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_1", Response: map[string]interface{}{"ok": true}},
			}},
		}},
		Tools: []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{Name: "tool_call_1"}}}},
	}
	geminiReqBody, _ := json.Marshal(geminiReq)
	g2c := &geminiToCodexRequest{}
	codexReqBody, err := g2c.Transform(geminiReqBody, "codex", false)
	if err != nil {
		t.Fatalf("Transform gemini->codex: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(codexReqBody, &codexReq); err != nil {
		t.Fatalf("unmarshal codex req: %v", err)
	}
	if !codexInputHasRoleTextParts(codexReq.Input, "developer", "sys") {
		t.Fatalf("expected system instruction in input")
	}

	codexResp2 := CodexResponse{
		Status: "completed",
		Usage:  CodexUsage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		Output: []CodexOutput{{Type: "message", Content: "hello"}, {Type: "function_call", Name: "tool", CallID: "call_9", Arguments: `{"a":1}`}},
	}
	codexRespBody, _ := json.Marshal(codexResp2)
	c2g := &geminiToCodexResponse{}
	geminiOut, err := c2g.Transform(codexRespBody)
	if err != nil {
		t.Fatalf("Transform codex->gemini: %v", err)
	}
	var geminiOutResp GeminiResponse
	if err := json.Unmarshal(geminiOut, &geminiOutResp); err != nil {
		t.Fatalf("unmarshal gemini resp: %v", err)
	}
	if len(geminiOutResp.Candidates) == 0 {
		t.Fatalf("candidates missing")
	}

	state := NewTransformState()
	streamEvent := CodexStreamEvent{Type: "response.output_text.delta", Delta: &CodexDelta{Type: "output_text_delta", Text: "hi"}}
	streamBody, _ := json.Marshal(streamEvent)
	streamOut, err := c2g.TransformChunk(FormatSSE("", json.RawMessage(streamBody)), state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if len(streamOut) == 0 || !strings.Contains(string(streamOut), "\"text\"") {
		t.Fatalf("stream output missing")
	}
}

func TestGeminiToCodexSingleTextInput(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}}}
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
	if s, ok := codexReq.Input.(string); !ok || s != "hi" {
		t.Fatalf("expected string input")
	}
}

func TestMapCodexRoleToGeminiUnknown(t *testing.T) {
	if mapCodexRoleToGemini("other") != "user" {
		t.Fatalf("expected user for unknown")
	}
}

func TestGeminiToCodexThinkingBudget(t *testing.T) {
	budget := 0
	req := GeminiRequest{GenerationConfig: &GeminiGenerationConfig{ThinkingConfig: &GeminiThinkingConfig{ThinkingBudget: budget}}, Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}}}
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
	if codexReq.Reasoning == nil {
		t.Fatalf("reasoning missing")
	}
}

func TestGeminiToCodexTransformShortName(t *testing.T) {
	long := strings.Repeat("tool", 30)
	req := GeminiRequest{
		Contents: []GeminiContent{{Role: "model", Parts: []GeminiPart{{FunctionCall: &GeminiFunctionCall{Name: long, Args: map[string]interface{}{"x": 1}}}}}},
		Tools:    []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{Name: long}}}},
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
	if len(codexReq.Tools) == 0 || len(codexReq.Tools[0].Name) > maxToolNameLen {
		t.Fatalf("expected shortened tool name")
	}
}

func TestGeminiToCodexTransformBranches(t *testing.T) {
	req := GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: "sys"}}},
		GenerationConfig:  &GeminiGenerationConfig{ThinkingConfig: &GeminiThinkingConfig{ThinkingLevel: "high"}},
		Contents: []GeminiContent{{
			Role: "model",
			Parts: []GeminiPart{{Text: "out"}, {
				FunctionCall: &GeminiFunctionCall{Name: "tool_call_1", Args: map[string]interface{}{"x": 1}},
			}, {
				FunctionResponse: &GeminiFunctionResponse{Name: "tool", ID: "call_2", Response: map[string]interface{}{"ok": true}},
			}},
		}},
		Tools: []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{Name: "tool_call_1"}}}},
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
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort == "" {
		t.Fatalf("reasoning missing")
	}
	if !codexInputHasRoleTextParts(codexReq.Input, "developer", "sys") {
		t.Fatalf("expected system instruction in input")
	}
}

func TestGeminiToCodexTransformRoleAssistantOutput(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "model", Parts: []GeminiPart{{Text: "hi"}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "output_text") {
		t.Fatalf("expected output_text for assistant")
	}
}

func TestGeminiToCodexTransformCallIDExtraction(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{
		FunctionCall: &GeminiFunctionCall{Name: "tool_call_123", Args: map[string]interface{}{"x": 1}},
	}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "call_123") {
		t.Fatalf("expected call_id extraction")
	}
}

func TestGeminiToCodexTransformBranchesMore(t *testing.T) {
	topP := 0.9
	maxTokens := 9
	req := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			TopP:            &topP,
			ThinkingConfig:  &GeminiThinkingConfig{ThinkingLevel: "high"},
		},
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: ""}, {Text: "sys"}}},
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: "in"}, {FunctionCall: &GeminiFunctionCall{Name: "tool_name", Args: map[string]interface{}{"x": 1}}}},
		}, {
			Role:  "model",
			Parts: []GeminiPart{{Text: "out"}},
		}},
		Tools: []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{Name: ""}, {Name: "tool_name"}}}},
	}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "output_text") {
		t.Fatalf("expected output_text for assistant role")
	}
	if !strings.Contains(string(out), "function_call") {
		t.Fatalf("expected function_call")
	}
}

func TestGeminiToCodexNoReasoningEffort(t *testing.T) {
	req := GeminiRequest{GenerationConfig: &GeminiGenerationConfig{ThinkingConfig: &GeminiThinkingConfig{ThinkingBudget: -2}}, Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}}}
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
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" {
		t.Fatalf("expected default reasoning")
	}
}

func TestGeminiToCodexTransformShortMapAndCallID(t *testing.T) {
	long := strings.Repeat("tool", 30)
	req := GeminiRequest{
		Tools: []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{Name: long}}}},
		Contents: []GeminiContent{{
			Role: "user",
			Parts: []GeminiPart{{
				FunctionCall: &GeminiFunctionCall{Name: long + "_call_77", Args: map[string]interface{}{"x": 1}},
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "function_call") {
		t.Fatalf("expected function_call")
	}
}

func TestGeminiToCodexTransformExhaustive(t *testing.T) {
	temp := 0.2
	req := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: 7,
			Temperature:     &temp,
			ThinkingConfig:  &GeminiThinkingConfig{ThinkingBudget: 10},
		},
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: "sys"}}},
		Tools:             []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{Name: "tool_call_1", Description: "d"}}}},
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: "in"}, {FunctionCall: &GeminiFunctionCall{Name: "tool_call_1", Args: map[string]interface{}{"x": 1}}}, {FunctionResponse: &GeminiFunctionResponse{Name: "tool_call_1", Response: map[string]interface{}{"ok": true}}}},
		}},
	}
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

func TestGeminiToCodexTransformNoGenConfig(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "model", Parts: []GeminiPart{{Text: "out"}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "output_text") {
		t.Fatalf("expected output_text")
	}
}

func TestGeminiToCodexTransformCallIDSuffix(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{FunctionCall: &GeminiFunctionCall{Name: "tool_call_5", Args: map[string]interface{}{"x": 1}}}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "call_5") {
		t.Fatalf("expected call_5")
	}
}

func TestGeminiToCodexTransformUnknownRole(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{Role: "unknown", Parts: []GeminiPart{{Text: "hi"}}}}}
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
	switch v := codexReq.Input.(type) {
	case string:
		if v != "hi" {
			t.Fatalf("expected input string")
		}
	case []interface{}:
		if len(v) == 0 {
			t.Fatalf("expected input items")
		}
		item, _ := v[0].(map[string]interface{})
		if item["role"] != "user" {
			t.Fatalf("expected role user")
		}
	default:
		t.Fatalf("unexpected input type")
	}
}

func TestGeminiToCodexCallIDExtraction(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{{
			Role: "model",
			Parts: []GeminiPart{{
				FunctionCall: &GeminiFunctionCall{
					Name: "tool_call_123",
					Args: map[string]interface{}{"x": 1},
				},
			}},
		}, {
			Role: "user",
			Parts: []GeminiPart{{
				FunctionResponse: &GeminiFunctionResponse{
					Name:     "tool_call_456",
					Response: map[string]interface{}{"result": "ok"},
				},
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	input, ok := raw["input"].([]interface{})
	if !ok {
		t.Fatalf("expected input array")
	}
	var callID string
	var outputID string
	for _, item := range input {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		switch typ {
		case "function_call":
			if v, ok := m["call_id"].(string); ok {
				callID = v
			}
		case "function_call_output":
			if v, ok := m["call_id"].(string); ok {
				outputID = v
			}
		}
	}
	if callID == "" || outputID == "" || callID != outputID {
		t.Fatalf("expected paired call ids")
	}
}

func TestGeminiToCodexDefaultsAndToolCleaning(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}},
		Tools: []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{
			Name: "tool",
			Parameters: map[string]interface{}{
				"$schema":              "x",
				"type":                 "object",
				"additionalProperties": true,
			},
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
	if codexReq.Stream != true || codexReq.Store != false {
		t.Fatalf("expected stream/store defaults")
	}
	if codexReq.ToolChoice != "auto" {
		t.Fatalf("expected tool_choice auto")
	}
	if codexReq.ParallelToolCalls == nil || !*codexReq.ParallelToolCalls {
		t.Fatalf("expected parallel_tool_calls true")
	}
	if len(codexReq.Include) != 1 || codexReq.Include[0] != "reasoning.encrypted_content" {
		t.Fatalf("expected include defaults")
	}
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" || codexReq.Reasoning.Summary != "auto" {
		t.Fatalf("expected reasoning defaults")
	}
	if len(codexReq.Tools) == 0 {
		t.Fatalf("expected tools")
	}
	params, ok := codexReq.Tools[0].Parameters.(map[string]interface{})
	if !ok {
		t.Fatalf("expected params map")
	}
	if _, ok := params["$schema"]; ok {
		t.Fatalf("expected $schema removed")
	}
	if v, ok := params["additionalProperties"].(bool); !ok || v {
		t.Fatalf("expected additionalProperties false")
	}
}

func TestGeminiToCodexToolParamsNonMap(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}},
		Tools: []GeminiTool{{FunctionDeclarations: []GeminiFunctionDecl{{
			Name:                 "tool",
			ParametersJsonSchema: []interface{}{"bad"},
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
	if len(codexReq.Tools) == 0 || codexReq.Tools[0].Parameters == nil {
		t.Fatalf("expected parameters")
	}
}

func TestGeminiToCodexReasoningSummaryDefault(t *testing.T) {
	req := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{ThinkingConfig: &GeminiThinkingConfig{ThinkingLevel: "low"}},
		Contents:         []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}},
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
	if codexReq.Reasoning == nil || codexReq.Reasoning.Summary != "auto" {
		t.Fatalf("expected summary default")
	}
}

func TestGeminiToCodexReasoningEffortTrim(t *testing.T) {
	req := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{ThinkingConfig: &GeminiThinkingConfig{ThinkingLevel: "  "}},
		Contents:         []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}},
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
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" {
		t.Fatalf("expected trimmed effort default")
	}
}

func codexInputHasRoleTextParts(input interface{}, role string, text string) bool {
	items, ok := input.([]interface{})
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok || m["type"] != "message" || m["role"] != role {
			continue
		}
		parts, ok := m["content"].([]interface{})
		if !ok {
			continue
		}
		for _, part := range parts {
			pm, ok := part.(map[string]interface{})
			if ok && pm["text"] == text {
				return true
			}
		}
	}
	return false
}
