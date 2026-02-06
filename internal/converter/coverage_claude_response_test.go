package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeToGeminiResponse(t *testing.T) {
	resp := ClaudeResponse{
		Usage: ClaudeUsage{InputTokens: 1, OutputTokens: 2},
		Content: []ClaudeContentBlock{{
			Type: "text",
			Text: "hello",
		}, {
			Type:  "tool_use",
			ID:    "call_1",
			Name:  "tool",
			Input: map[string]interface{}{"x": 1},
		}},
		StopReason: "max_tokens",
	}
	body, _ := json.Marshal(resp)
	conv := &claudeToGeminiResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var geminiResp GeminiResponse
	if err := json.Unmarshal(out, &geminiResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(geminiResp.Candidates) == 0 || geminiResp.Candidates[0].FinishReason != "MAX_TOKENS" {
		t.Fatalf("finish reason missing")
	}
	if len(geminiResp.Candidates[0].Content.Parts) < 2 {
		t.Fatalf("parts missing")
	}
}

func TestGeminiToClaudeResponseToolUseStop(t *testing.T) {
	resp := GeminiResponse{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{FunctionCall: &GeminiFunctionCall{Name: "tool", Args: map[string]interface{}{"x": 1}}}}},
		FinishReason: "STOP",
	}}}
	body, _ := json.Marshal(resp)
	conv := &geminiToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(out, &claudeResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if claudeResp.StopReason != "tool_use" {
		t.Fatalf("expected tool_use stop reason")
	}
}

func TestGeminiToClaudeRequestRolesAndResponses(t *testing.T) {
	req := GeminiRequest{Contents: []GeminiContent{{
		Role:  "unknown",
		Parts: []GeminiPart{{FunctionResponse: &GeminiFunctionResponse{Name: "tool", Response: map[string]interface{}{"ok": true}}}},
	}}}
	body, _ := json.Marshal(req)
	conv := &geminiToClaudeRequest{}
	out, err := conv.Transform(body, "claude", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var claudeReq ClaudeRequest
	if err := json.Unmarshal(out, &claudeReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(claudeReq.Messages) == 0 {
		t.Fatalf("messages missing")
	}
}

func TestGeminiToClaudeResponseMaxTokens2(t *testing.T) {
	resp := GeminiResponse{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "MAX_TOKENS",
	}}}
	body, _ := json.Marshal(resp)
	conv := &geminiToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(out, &claudeResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if claudeResp.StopReason != "max_tokens" {
		t.Fatalf("expected max_tokens")
	}
}

func TestClaudeToGeminiResponseToolUse(t *testing.T) {
	resp := ClaudeResponse{Content: []ClaudeContentBlock{{
		Type:  "tool_use",
		ID:    "call_1",
		Name:  "tool",
		Input: map[string]interface{}{"x": 1},
	}}, StopReason: "tool_use"}
	body, _ := json.Marshal(resp)
	conv := &claudeToGeminiResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "functionCall") {
		t.Fatalf("expected functionCall")
	}
}

func TestClaudeToGeminiResponseEndTurn(t *testing.T) {
	resp := ClaudeResponse{Content: []ClaudeContentBlock{{Type: "text", Text: "hi"}}, StopReason: "end_turn"}
	body, _ := json.Marshal(resp)
	conv := &claudeToGeminiResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "STOP") {
		t.Fatalf("expected STOP finish reason")
	}
}

func TestGeminiToClaudeResponseStopNoTool(t *testing.T) {
	resp := GeminiResponse{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "STOP",
	}}}
	body, _ := json.Marshal(resp)
	conv := &geminiToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "end_turn") {
		t.Fatalf("expected end_turn")
	}
}

func TestGeminiToClaudeResponseMaxTokens(t *testing.T) {
	resp := GeminiResponse{Candidates: []GeminiCandidate{{
		Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "hi"}}},
		FinishReason: "MAX_TOKENS",
	}}}
	body, _ := json.Marshal(resp)
	conv := &geminiToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "max_tokens") {
		t.Fatalf("expected max_tokens")
	}
}

func TestCodexToClaudeResponseFunctionCall(t *testing.T) {
	resp := CodexResponse{Model: "m", Usage: CodexUsage{InputTokens: 1, OutputTokens: 1}, Output: []CodexOutput{{
		Type:      "function_call",
		ID:        "call_1",
		Name:      "tool",
		Arguments: `{"x":1}`,
		Status:    "completed",
	}}}
	body, _ := json.Marshal(resp)
	conv := &codexToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "tool_use") {
		t.Fatalf("expected tool_use")
	}
}

func TestCodexToClaudeResponseMessage(t *testing.T) {
	resp := CodexResponse{Model: "m", Usage: CodexUsage{InputTokens: 1, OutputTokens: 1}, Output: []CodexOutput{{
		Type:    "message",
		Role:    "assistant",
		Content: "hi",
	}}}
	body, _ := json.Marshal(resp)
	conv := &codexToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "\"type\":\"text\"") {
		t.Fatalf("expected text block")
	}
}

func TestGeminiToClaudeResponseUsage(t *testing.T) {
	resp := GeminiResponse{
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     1,
			CandidatesTokenCount: 2,
		},
		Candidates: []GeminiCandidate{{
			Content: GeminiContent{Parts: []GeminiPart{{Text: "hi"}}},
		}},
	}
	body, _ := json.Marshal(resp)
	conv := &geminiToClaudeResponse{}
	out, err := conv.Transform(body)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "\"input_tokens\":1") {
		t.Fatalf("expected usage metadata")
	}
}

func TestCodexToClaudeResponseInvalidJSON(t *testing.T) {
	out, err := (&codexToClaudeResponse{}).Transform([]byte("{"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected empty output")
	}
}
