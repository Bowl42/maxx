package converter

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

var errTest = errors.New("test error")

func TestCodexInstructionsGlobalSettings(t *testing.T) {
	SetGlobalSettingsGetter(func() (*GlobalSettings, error) {
		return &GlobalSettings{CodexInstructionsEnabled: true}, nil
	})
	defer SetGlobalSettingsGetter(nil)
	if !GetCodexInstructionsEnabled() {
		t.Fatalf("expected enabled from global settings")
	}
	if settings := GetGlobalSettings(); settings == nil || !settings.CodexInstructionsEnabled {
		t.Fatalf("expected settings")
	}
}

func TestCodexInstructionsNoGlobalSettings(t *testing.T) {
	SetGlobalSettingsGetter(nil)
	SetCodexInstructionsEnabled(false)
	if GetGlobalSettings() != nil {
		t.Fatalf("expected nil settings")
	}
}

func TestCodexInstructionsGlobalSettingsError(t *testing.T) {
	SetGlobalSettingsGetter(func() (*GlobalSettings, error) {
		return nil, errTest
	})
	defer SetGlobalSettingsGetter(nil)
	if GetGlobalSettings() != nil {
		t.Fatalf("expected nil on error")
	}
}

func TestCodexInstructionsBranches(t *testing.T) {
	SetCodexInstructionsEnabled(true)
	defer SetCodexInstructionsEnabled(false)

	_, prompt := codexInstructionsForCodex("codex", "")
	if prompt == "" {
		t.Fatalf("expected codex prompt")
	}
	if ok, empty := codexInstructionsForCodex("codex", prompt); !ok || empty != "" {
		t.Fatalf("expected prefix match")
	}
	if _, v := codexInstructionsForCodex("codex-max", ""); v == "" {
		t.Fatalf("expected codex-max prompt")
	}
	if _, v := codexInstructionsForCodex("5.2-codex", ""); v == "" {
		t.Fatalf("expected 5.2-codex prompt")
	}
	if _, v := codexInstructionsForCodex("gpt-5.1", ""); v == "" {
		t.Fatalf("expected 5.1 prompt")
	}
	if _, v := codexInstructionsForCodex("gpt-5.2", ""); v == "" {
		t.Fatalf("expected 5.2 prompt")
	}
	if _, v := codexInstructionsForCodex("other", ""); v == "" {
		t.Fatalf("expected default prompt")
	}

	if _, v := codexInstructionsForOpenCode(""); v == "" {
		t.Fatalf("expected opencode prompt")
	}
	if ok, v := codexInstructionsForOpenCode(opencodeCodexInstructions); !ok || v != "" {
		t.Fatalf("expected opencode prefix match")
	}

	orig := opencodeCodexInstructions
	opencodeCodexInstructions = ""
	if ok, v := codexInstructionsForOpenCode("sys"); ok || v != "" {
		t.Fatalf("expected empty opencode instructions")
	}
	opencodeCodexInstructions = orig
}

func TestCodexUserAgentHelpers(t *testing.T) {
	raw := []byte(`{"k":"v"}`)
	if got := InjectCodexUserAgent(nil, "ua"); got != nil {
		t.Fatalf("expected nil for empty raw")
	}
	if got := InjectCodexUserAgent(raw, ""); string(got) != string(raw) {
		t.Fatalf("expected no change for empty user agent")
	}
	bad := []byte("{")
	if got := InjectCodexUserAgent(bad, "ua"); string(got) != string(bad) {
		t.Fatalf("expected no change for invalid json")
	}
	if got := ExtractCodexUserAgent(bad); got != "" {
		t.Fatalf("expected empty for invalid json")
	}
	if got := StripCodexUserAgent(bad); string(got) != string(bad) {
		t.Fatalf("expected no change for invalid json")
	}
	if got := StripCodexUserAgent(raw); string(got) != string(raw) {
		t.Fatalf("expected no change when key missing")
	}
	if got := ExtractCodexUserAgent(nil); got != "" {
		t.Fatalf("expected empty for nil")
	}
	if got := StripCodexUserAgent(nil); got != nil {
		t.Fatalf("expected nil for empty raw")
	}
}

func TestOpenAIToCodexSystemMessage(t *testing.T) {
	SetCodexInstructionsEnabled(false)
	req := OpenAIRequest{
		Messages: []OpenAIMessage{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hi"},
		},
	}
	body, _ := json.Marshal(req)
	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !codexInputHasRoleText(codexReq.Input, "developer", "sys") {
		t.Fatalf("expected system message")
	}
	if codexReq.Instructions != "" {
		t.Fatalf("expected no instructions when disabled")
	}
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" || codexReq.Reasoning.Summary != "auto" {
		t.Fatalf("expected default reasoning")
	}
}

func TestOpenAIToCodexReasoningWhitespace(t *testing.T) {
	req := OpenAIRequest{
		ReasoningEffort: " ",
		Messages:        []OpenAIMessage{{Role: "user", Content: "hi"}},
	}
	body, _ := json.Marshal(req)
	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" {
		t.Fatalf("expected reasoning default for whitespace")
	}
}

func TestOpenAIToCodexToolMessageAndArrayContent(t *testing.T) {
	req := OpenAIRequest{
		Messages: []OpenAIMessage{
			{Role: "assistant", ToolCalls: []OpenAIToolCall{{
				ID:       "call_1",
				Type:     "function",
				Function: OpenAIFunctionCall{Name: "tool", Arguments: `{"x":1}`},
			}}},
			{Role: "tool", ToolCallID: "call_1", Content: "ok"},
			{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "hi"}}},
		},
		Tools: []OpenAITool{{Type: "function", Function: OpenAIFunction{Name: "tool"}}},
	}
	body, _ := json.Marshal(req)
	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "function_call_output") {
		t.Fatalf("expected tool output")
	}
	if !strings.Contains(string(out), "function_call") {
		t.Fatalf("expected tool call")
	}
}

func TestOpenAIToCodexInstructionsEnabled(t *testing.T) {
	SetGlobalSettingsGetter(func() (*GlobalSettings, error) {
		return &GlobalSettings{CodexInstructionsEnabled: true}, nil
	})
	defer SetGlobalSettingsGetter(nil)
	req := OpenAIRequest{
		Messages: []OpenAIMessage{{Role: "user", Content: "hi"}},
	}
	body, _ := json.Marshal(req)
	body = InjectCodexUserAgent(body, "opencode/1.0")
	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if strings.TrimSpace(codexReq.Instructions) == "" {
		t.Fatalf("expected instructions")
	}
}

func TestOpenAIToCodexToolNameFallback(t *testing.T) {
	req := OpenAIRequest{
		Messages: []OpenAIMessage{{
			Role: "assistant",
			ToolCalls: []OpenAIToolCall{{
				ID:       "call_1",
				Type:     "function",
				Function: OpenAIFunctionCall{Name: "missing_tool", Arguments: `{"x":1}`},
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "missing_tool") {
		t.Fatalf("expected fallback tool name")
	}
}

func TestClaudeToCodexSystemString(t *testing.T) {
	SetCodexInstructionsEnabled(false)
	req := ClaudeRequest{
		System: "sys",
		Messages: []ClaudeMessage{{
			Role:    "user",
			Content: "hi",
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !codexInputHasRoleText(codexReq.Input, "developer", "sys") {
		t.Fatalf("expected system message")
	}
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" {
		t.Fatalf("expected default reasoning")
	}
}

func TestClaudeToCodexOutputConfigEffort(t *testing.T) {
	req := ClaudeRequest{
		OutputConfig: &ClaudeOutputConfig{Effort: "HIGH"},
		Messages:     []ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "high" {
		t.Fatalf("expected mapped effort")
	}
	if codexReq.Reasoning.Summary != "auto" {
		t.Fatalf("expected summary default")
	}
}

func TestClaudeToCodexOutputConfigEmptyEffort(t *testing.T) {
	req := ClaudeRequest{
		OutputConfig: &ClaudeOutputConfig{Effort: " "},
		Messages:     []ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if codexReq.Reasoning == nil || codexReq.Reasoning.Effort != "medium" {
		t.Fatalf("expected default effort")
	}
	if codexReq.Reasoning.Summary != "auto" {
		t.Fatalf("expected summary default")
	}
}

func TestClaudeToCodexToolBlocks(t *testing.T) {
	req := ClaudeRequest{
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hi"},
				map[string]interface{}{"type": "tool_use", "id": "call_1", "name": "tool", "input": map[string]interface{}{"x": 1}},
				map[string]interface{}{"type": "tool_result", "tool_use_id": "call_1", "content": "ok"},
			},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "function_call") {
		t.Fatalf("expected tool use conversion")
	}
	if !strings.Contains(string(out), "function_call_output") {
		t.Fatalf("expected tool result conversion")
	}
}

func TestClaudeToCodexInstructionsEnabled(t *testing.T) {
	SetGlobalSettingsGetter(func() (*GlobalSettings, error) {
		return &GlobalSettings{CodexInstructionsEnabled: true}, nil
	})
	defer SetGlobalSettingsGetter(nil)
	req := ClaudeRequest{Messages: []ClaudeMessage{{Role: "user", Content: "hi"}}}
	body, _ := json.Marshal(req)
	body = InjectCodexUserAgent(body, "opencode/1.0")
	conv := &claudeToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if strings.TrimSpace(codexReq.Instructions) == "" {
		t.Fatalf("expected instructions")
	}
}

func TestGeminiToCodexInstructionsEnabled(t *testing.T) {
	SetCodexInstructionsEnabled(true)
	defer SetCodexInstructionsEnabled(false)

	req := GeminiRequest{Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}}}
	body, _ := json.Marshal(req)
	body = InjectCodexUserAgent(body, "opencode/1.0")
	conv := &geminiToCodexRequest{}
	out, err := conv.Transform(body, "codex", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var codexReq CodexRequest
	if err := json.Unmarshal(out, &codexReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if strings.TrimSpace(codexReq.Instructions) == "" {
		t.Fatalf("expected instructions")
	}
}
