package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeGeminiHelperCoverage(t *testing.T) {
	msgs := []ClaudeMessage{{
		Role: "assistant",
		Content: []interface{}{map[string]interface{}{
			"type":      "thinking",
			"signature": strings.Repeat("a", MinSignatureLength),
		}},
	}}
	if !hasValidSignatureForFunctionCalls(msgs, "") {
		t.Fatalf("expected valid signature from messages")
	}
	if shouldEnableThinkingByDefault("claude-opus-4-5-20250101") != true {
		t.Fatalf("expected thinking enabled for opus 4.5")
	}
	if shouldEnableThinkingByDefault("claude-opus-4-6-20260205") != true {
		t.Fatalf("expected thinking enabled for opus 4.6")
	}
	if shouldEnableThinkingByDefault("model-thinking") != true {
		t.Fatalf("expected thinking enabled for -thinking")
	}
	if shouldEnableThinkingByDefault("claude-haiku") != false {
		t.Fatalf("expected thinking disabled for non-thinking")
	}
}

func TestClaudeToGeminiHelpersDeepClean(t *testing.T) {
	data := map[string]interface{}{
		"a": "[undefined]",
		"b": map[string]interface{}{"c": "[undefined]"},
		"d": []interface{}{map[string]interface{}{"e": "[undefined]"}},
	}
	deepCleanUndefined(data)
	if _, ok := data["a"]; ok {
		t.Fatalf("expected removal")
	}
	if nested, ok := data["b"].(map[string]interface{}); ok {
		if _, ok := nested["c"]; ok {
			t.Fatalf("expected nested removal")
		}
	}
	arr := data["d"].([]interface{})
	if nested, ok := arr[0].(map[string]interface{}); ok {
		if _, ok := nested["e"]; ok {
			t.Fatalf("expected array removal")
		}
	}
}

func TestClaudeToCodexSystemArray(t *testing.T) {
	req := ClaudeRequest{System: []interface{}{map[string]interface{}{"text": "sys"}}, Messages: []ClaudeMessage{{Role: "user", Content: "hi"}}}
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
		t.Fatalf("expected system message in input")
	}
}

func TestApplyClaudeThinkingDisabled(t *testing.T) {
	openaiReq := &OpenAIRequest{}
	claudeReq := &ClaudeRequest{Thinking: map[string]interface{}{"type": "disabled"}}
	applyClaudeThinkingToOpenAI(openaiReq, claudeReq)
	if openaiReq.ReasoningEffort != "none" {
		t.Fatalf("expected none")
	}
}

func TestExtractClaudeThinkingTextEmpty(t *testing.T) {
	if extractClaudeThinkingText(map[string]interface{}{}) != "" {
		t.Fatalf("expected empty")
	}
}

func TestApplyClaudeThinkingNilCases(t *testing.T) {
	applyClaudeThinkingToOpenAI(nil, &ClaudeRequest{})
	applyClaudeThinkingToOpenAI(&OpenAIRequest{}, nil)
}

func TestClaudeToGeminiHelpersExtra(t *testing.T) {
	schema := map[string]interface{}{
		"items": map[string]interface{}{
			"type": "string",
		},
	}
	cleanJSONSchema(schema)
	if _, ok := schema["items"]; !ok {
		t.Fatalf("expected items to remain")
	}

	msgs := []ClaudeMessage{{Role: "assistant", Content: "plain"}}
	if count := FilterInvalidThinkingBlocks(msgs); count != 0 {
		t.Fatalf("unexpected filtered count")
	}

	msgs = []ClaudeMessage{{Role: "assistant", Content: []interface{}{
		map[string]interface{}{"type": "thinking", "thinking": ""},
	}}}
	FilterInvalidThinkingBlocks(msgs)
	if blocks, ok := msgs[0].Content.([]interface{}); !ok || len(blocks) == 0 {
		t.Fatalf("expected fallback block")
	}

	msgs = []ClaudeMessage{{Role: "assistant", Content: "text"}}
	RemoveTrailingUnsignedThinking(msgs)

	msgs = []ClaudeMessage{{Role: "assistant", Content: []interface{}{"bad"}}}
	RemoveTrailingUnsignedThinking(msgs)

	if hasValidSignatureForFunctionCalls([]ClaudeMessage{{Role: "assistant", Content: []interface{}{"bad"}}}, "") {
		t.Fatalf("expected no valid signature")
	}
	if hasThinkingHistory([]ClaudeMessage{{Role: "assistant", Content: "plain"}}) {
		t.Fatalf("expected no thinking history")
	}
	if hasFunctionCalls([]ClaudeMessage{{Role: "user", Content: "plain"}}) {
		t.Fatalf("expected no function calls")
	}
	if shouldDisableThinkingDueToHistory([]ClaudeMessage{{Role: "assistant", Content: "plain"}}) {
		t.Fatalf("expected no history disable")
	}
}
