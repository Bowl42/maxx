package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIToGeminiHelpersMisc(t *testing.T) {
	if got := stringifyContent(map[string]interface{}{"a": 1}); !strings.Contains(got, "a") {
		t.Fatalf("stringifyContent json")
	}
	if parseInlineImage("data:;base64,Zm9v") == nil {
		t.Fatalf("expected inline data even without mime")
	}
}

func TestOpenAIToCodexLongToolNameShortening(t *testing.T) {
	longName := strings.Repeat("tool", 30)
	req := OpenAIRequest{Tools: []OpenAITool{{Type: "function", Function: OpenAIFunction{Name: longName}}}, Messages: []OpenAIMessage{{Role: "user", Content: "hi"}}}
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
	if len(codexReq.Tools) == 0 || len(codexReq.Tools[0].Name) > maxToolNameLen {
		t.Fatalf("tool name not shortened")
	}
}

func TestCodexToOpenAIInputString(t *testing.T) {
	req := CodexRequest{Input: "hi"}
	body, _ := json.Marshal(req)
	conv := &codexToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var openaiReq OpenAIRequest
	if err := json.Unmarshal(out, &openaiReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(openaiReq.Messages) == 0 {
		t.Fatalf("messages missing")
	}
}

func TestOpenAIToCodexContentArray(t *testing.T) {
	req := OpenAIRequest{Messages: []OpenAIMessage{{Role: "user", Content: []interface{}{map[string]interface{}{"type": "text", "text": "hi"}}}}}
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
	if codexReq.Input == nil {
		t.Fatalf("input missing")
	}
}

func TestGeminiToOpenAITransformReasoningAndImage(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{
		Role:  "model",
		Parts: []GeminiPart{{Thought: true, Text: "think"}, {Text: "hi"}, {InlineData: &GeminiInlineData{MimeType: "image/png", Data: "Zm9v"}}},
	}}}
	body, _ := json.Marshal(req)
	conv := &geminiToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "reasoning_content") {
		t.Fatalf("expected reasoning_content")
	}
	if !strings.Contains(string(out), "image_url") {
		t.Fatalf("expected image_url")
	}
}

func TestGeminiToOpenAITransformStopSequences(t *testing.T) {
	req := GeminiRequest{GenerationConfig: &GeminiGenerationConfig{StopSequences: []string{"s"}}, Contents: []GeminiContent{{Role: "user", Parts: []GeminiPart{{Text: "hi"}}}}}
	body, _ := json.Marshal(req)
	conv := &geminiToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "stop") {
		t.Fatalf("expected stop sequences")
	}
}

func TestOpenAIToGeminiHelpersExtra(t *testing.T) {
	if got := stringifyContent([]interface{}{"bad", map[string]interface{}{"text": "hi"}}); got != "hi" {
		t.Fatalf("unexpected stringify content")
	}
	if got := stringifyContent(func() {}); got != "" {
		t.Fatalf("expected empty stringify result")
	}

	if parseFilePart(map[string]interface{}{}) != nil {
		t.Fatalf("expected nil file part")
	}
	if parseFilePart(map[string]interface{}{"file": map[string]interface{}{"filename": "", "file_data": "x"}}) != nil {
		t.Fatalf("expected nil for empty filename")
	}
	if parseFilePart(map[string]interface{}{"file": map[string]interface{}{"filename": "a.unknown", "file_data": "x"}}) != nil {
		t.Fatalf("expected nil for unknown ext")
	}
	if got := parseFilePart(map[string]interface{}{"file": map[string]interface{}{"filename": "a.txt", "file_data": "x"}}); got == nil {
		t.Fatalf("expected parsed file part")
	}

	if mimeFromExt("exe") != "" {
		t.Fatalf("expected empty mime")
	}

	if cfg := parseToolChoice("none"); cfg == nil || cfg.FunctionCallingConfig.Mode != "NONE" {
		t.Fatalf("expected NONE mode")
	}
	if cfg := parseToolChoice(" auto "); cfg == nil || cfg.FunctionCallingConfig.Mode != "AUTO" {
		t.Fatalf("expected AUTO mode")
	}
	if cfg := parseToolChoice("required"); cfg == nil || cfg.FunctionCallingConfig.Mode != "ANY" {
		t.Fatalf("expected ANY mode")
	}
	if cfg := parseToolChoice(map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name": "tool",
		},
	}); cfg == nil || cfg.FunctionCallingConfig.Mode != "ANY" {
		t.Fatalf("expected function tool config")
	}
	if cfg := parseToolChoice(map[string]interface{}{
		"type":     "function",
		"function": map[string]interface{}{"name": ""},
	}); cfg != nil {
		t.Fatalf("expected nil tool config")
	}
}

func TestClaudeToOpenAIHelpersExtra(t *testing.T) {
	if got := convertClaudeToolResultContentToString([]interface{}{map[string]interface{}{"text": "a"}, "bad"}); got != "a" {
		t.Fatalf("unexpected tool result content")
	}
	if got := convertClaudeToolResultContentToString(func() {}); got != "" {
		t.Fatalf("expected empty tool result")
	}

	openaiReq := &OpenAIRequest{}
	applyClaudeThinkingToOpenAI(openaiReq, &ClaudeRequest{OutputConfig: &ClaudeOutputConfig{Effort: "high"}})
	if openaiReq.ReasoningEffort != "high" {
		t.Fatalf("expected effort")
	}

	openaiReq = &OpenAIRequest{}
	applyClaudeThinkingToOpenAI(openaiReq, &ClaudeRequest{})
	if openaiReq.ReasoningEffort != "" {
		t.Fatalf("expected no effort")
	}

	openaiReq = &OpenAIRequest{}
	applyClaudeThinkingToOpenAI(openaiReq, &ClaudeRequest{Thinking: map[string]interface{}{"type": "enabled"}})
	if openaiReq.ReasoningEffort != "auto" {
		t.Fatalf("expected auto effort")
	}

	openaiReq = &OpenAIRequest{}
	applyClaudeThinkingToOpenAI(openaiReq, &ClaudeRequest{Thinking: map[string]interface{}{"type": "enabled", "budget_tokens": 2000}})
	if openaiReq.ReasoningEffort == "" {
		t.Fatalf("expected mapped effort")
	}

	openaiReq = &OpenAIRequest{}
	applyClaudeThinkingToOpenAI(openaiReq, &ClaudeRequest{Thinking: map[string]interface{}{"type": "disabled"}})
	if openaiReq.ReasoningEffort != "none" {
		t.Fatalf("expected none effort")
	}
}
