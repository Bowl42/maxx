package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeToCodexRequestDetails(t *testing.T) {
	req := ClaudeRequest{
		System: "sys",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{map[string]interface{}{
				"type":  "tool_use",
				"id":    "call_1",
				"name":  "tool",
				"input": map[string]interface{}{"x": 1},
			}},
		}, {
			Role: "user",
			Content: []interface{}{map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": "call_1",
				"content":     "ok",
			}},
		}},
		Tools: []ClaudeTool{{Name: "tool", InputSchema: map[string]interface{}{"type": "object"}}},
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
	items, ok := codexReq.Input.([]interface{})
	if !ok {
		t.Fatalf("expected input array")
	}
	foundCall := false
	foundOutput := false
	for _, item := range items {
		m, _ := item.(map[string]interface{})
		if m["type"] == "function_call" {
			foundCall = true
		}
		if m["type"] == "function_call_output" {
			foundOutput = true
		}
	}
	if !foundCall || !foundOutput {
		t.Fatalf("missing tool items")
	}
}

func TestGeminiToClaudeRequestGenerationConfig(t *testing.T) {
	topK := 5
	req := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{TopK: &topK, StopSequences: []string{"x"}},
		Contents:         []GeminiContent{{Role: "model", Parts: []GeminiPart{{Text: "hi"}}}},
	}
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
	if claudeReq.TopK == nil || len(claudeReq.StopSequences) != 1 {
		t.Fatalf("generation config missing")
	}
}

func TestClaudeToGeminiRequestToolResultAndThinking(t *testing.T) {
	sig := strings.Repeat("a", MinSignatureLength)
	req := ClaudeRequest{
		Model: "claude-opus-4-5",
		Thinking: map[string]interface{}{
			"type":          "enabled",
			"budget_tokens": float64(99999),
		},
		Tools: []ClaudeTool{{Type: "web_search_20250305"}},
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{map[string]interface{}{
				"type":      "thinking",
				"thinking":  "t",
				"signature": sig,
			}, map[string]interface{}{
				"type":  "tool_use",
				"id":    "call_1",
				"name":  "tool",
				"input": map[string]interface{}{"x": 1},
			}, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": "call_1",
				"is_error":    true,
				"content":     "",
			}, map[string]interface{}{
				"type": "image",
				"source": map[string]interface{}{
					"type":       "base64",
					"media_type": "image/png",
					"data":       "Zm9v",
				},
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-thinking", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var geminiReq GeminiRequest
	if err := json.Unmarshal(out, &geminiReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if geminiReq.GenerationConfig == nil || geminiReq.GenerationConfig.ThinkingConfig == nil {
		t.Fatalf("thinking config missing")
	}
	if geminiReq.GenerationConfig.ThinkingConfig.ThinkingBudget != 24576 {
		t.Fatalf("expected capped budget")
	}
	if !strings.Contains(string(out), "functionResponse") || !strings.Contains(string(out), "inlineData") {
		t.Fatalf("expected tool_result and image parts")
	}
}

func TestClaudeToGeminiRequestBlocksAndTools(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-3-7-sonnet",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{map[string]interface{}{
				"type":          "text",
				"text":          "hi",
				"cache_control": "cache",
			}, map[string]interface{}{
				"type":      "thinking",
				"thinking":  "", // empty -> downgraded
				"signature": strings.Repeat("a", MinSignatureLength),
			}, map[string]interface{}{
				"type":  "tool_use",
				"id":    "call_1",
				"name":  "tool",
				"input": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			}, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": "call_1",
				"content":     []interface{}{map[string]interface{}{"text": "a"}, map[string]interface{}{"text": "b"}},
			}, map[string]interface{}{
				"type": "document",
				"source": map[string]interface{}{
					"type":       "base64",
					"media_type": "application/pdf",
					"data":       "Zg==",
				},
			}, map[string]interface{}{
				"type": "redacted_thinking",
				"data": "secret",
			}, map[string]interface{}{
				"type": "server_tool_use",
			}, map[string]interface{}{
				"type": "web_search_tool_result",
			}},
		}},
		Tools: []ClaudeTool{{Name: "tool", InputSchema: map[string]interface{}{"type": "object"}}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini-1.5", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var geminiReq GeminiRequest
	if err := json.Unmarshal(out, &geminiReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(geminiReq.Contents) == 0 {
		t.Fatalf("contents missing")
	}
	if geminiReq.Tools == nil || geminiReq.ToolConfig == nil {
		t.Fatalf("tools missing")
	}
	if strings.Contains(string(out), "cache_control") {
		t.Fatalf("expected cache_control removed")
	}
}

func TestClaudeToGeminiRequestGoogleSearchOnly(t *testing.T) {
	req := ClaudeRequest{
		Tools:    []ClaudeTool{{Type: "web_search_20250305"}},
		Messages: []ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "googleSearch") {
		t.Fatalf("expected googleSearch tool")
	}
}

func TestClaudeToGeminiRequestThinkingDisabledByTarget(t *testing.T) {
	req := ClaudeRequest{
		Model: "opus-4.5-thinking",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{map[string]interface{}{
				"type":      "thinking",
				"thinking":  "t",
				"signature": strings.Repeat("a", MinSignatureLength),
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini-1.5", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if strings.Contains(string(out), "thought") {
		t.Fatalf("expected thinking disabled for target")
	}
}

func TestClaudeToGeminiRequestEffortLevel(t *testing.T) {
	req := ClaudeRequest{
		OutputConfig: &ClaudeOutputConfig{Effort: "low"},
		Messages:     []ClaudeMessage{{Role: "user", Content: "hi"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-thinking", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "LOW") {
		t.Fatalf("expected effort level LOW")
	}
}

func TestClaudeToGeminiRequestDisableThinkingDueToHistory(t *testing.T) {
	req := ClaudeRequest{
		Thinking: map[string]interface{}{"type": "enabled", "budget_tokens": float64(10)},
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{map[string]interface{}{
				"type":  "tool_use",
				"id":    "call_1",
				"name":  "tool",
				"input": map[string]interface{}{"x": 1},
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-thinking", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if strings.Contains(string(out), "thought") {
		t.Fatalf("expected thinking cleared due to history")
	}
}

func TestClaudeToGeminiRequestSkipEmptyMessage(t *testing.T) {
	req := ClaudeRequest{
		System: "sys",
		Messages: []ClaudeMessage{{
			Role:    "user",
			Content: "(no content)",
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if strings.Contains(string(out), "(no content)") {
		t.Fatalf("expected content skipped")
	}
}

func TestClaudeToGeminiRequestToolResultSuccessFallback(t *testing.T) {
	req := ClaudeRequest{
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{map[string]interface{}{
				"type":  "tool_use",
				"id":    "call_1",
				"name":  "tool",
				"input": map[string]interface{}{},
			}, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": "call_1",
				"is_error":    false,
				"content":     "",
			}},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-thinking", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "Command executed successfully") {
		t.Fatalf("expected success fallback")
	}
}

func TestCodexToClaudeRequestInputString(t *testing.T) {
	req := CodexRequest{Input: "hi"}
	body, _ := json.Marshal(req)
	conv := &codexToClaudeRequest{}
	out, err := conv.Transform(body, "claude", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "hi") {
		t.Fatalf("expected content")
	}
}

func TestCodexToClaudeRequestFunctionOutput(t *testing.T) {
	req := CodexRequest{Input: []interface{}{map[string]interface{}{"type": "function_call_output", "call_id": "call_1", "output": "ok"}}}
	body, _ := json.Marshal(req)
	conv := &codexToClaudeRequest{}
	out, err := conv.Transform(body, "claude", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "tool_result") {
		t.Fatalf("expected tool_result")
	}
}

func TestClaudeToGeminiRequestToolsDefaultSchema(t *testing.T) {
	req := ClaudeRequest{
		Messages: []ClaudeMessage{{Role: "user", Content: "hi"}},
		Tools:    []ClaudeTool{{Name: "tool"}, {Name: "", Type: "web_search_20250305"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "functionDeclarations") {
		t.Fatalf("expected functionDeclarations")
	}
}

func TestClaudeToGeminiRequestToolSkipMissingName(t *testing.T) {
	req := ClaudeRequest{
		Messages: []ClaudeMessage{{Role: "user", Content: "hi"}},
		Tools:    []ClaudeTool{{Type: "custom"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if strings.Contains(string(out), "functionDeclarations") {
		t.Fatalf("expected no functionDeclarations")
	}
}

func TestCodexToClaudeRequestFunctionCallIDFallback(t *testing.T) {
	req := CodexRequest{Input: []interface{}{map[string]interface{}{"type": "function_call", "call_id": "call_1", "name": "tool", "arguments": "{}"}}}
	body, _ := json.Marshal(req)
	conv := &codexToClaudeRequest{}
	out, err := conv.Transform(body, "claude", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "tool_use") {
		t.Fatalf("expected tool_use")
	}
}

func TestClaudeToGeminiRequestMergeAdjacentRoles(t *testing.T) {
	req := ClaudeRequest{
		Messages: []ClaudeMessage{{Role: "user", Content: "hi"}, {Role: "user", Content: "there"}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var geminiReq GeminiRequest
	if err := json.Unmarshal(out, &geminiReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(geminiReq.Contents) != 1 {
		t.Fatalf("expected merged contents")
	}
}

func TestClaudeToGeminiRequestUnknownRoleAndToolResultString(t *testing.T) {
	req := ClaudeRequest{Messages: []ClaudeMessage{{
		Role: "unknown",
		Content: []interface{}{map[string]interface{}{
			"type": "text",
			"text": "hi",
		}},
	}, {
		Role: "assistant",
		Content: []interface{}{map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": "call_1",
			"content":     "ok",
		}},
	}}}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gemini", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if !strings.Contains(string(out), "tool_result") && !strings.Contains(string(out), "functionResponse") {
		t.Fatalf("expected functionResponse")
	}
}

func TestClaudeToGeminiRequestSignatureDisableAndConfig(t *testing.T) {
	temp := 0.2
	topP := 0.7
	topK := 7
	req := ClaudeRequest{
		Model: "claude-3-5-haiku",
		Messages: []ClaudeMessage{{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "tool_use", "id": "tool_1", "name": "calc", "input": map[string]interface{}{"x": 1}},
			},
		}},
		Thinking:    map[string]interface{}{"type": "enabled", "budget_tokens": float64(123)},
		Temperature: &temp,
		TopP:        &topP,
		TopK:        &topK,
		OutputConfig: &ClaudeOutputConfig{
			Effort: "high",
		},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-3-5-haiku", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var gemReq GeminiRequest
	if err := json.Unmarshal(out, &gemReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gemReq.GenerationConfig.EffortLevel != "HIGH" {
		t.Fatalf("expected HIGH effort")
	}
	if gemReq.GenerationConfig.Temperature == nil || *gemReq.GenerationConfig.Temperature != temp {
		t.Fatalf("expected temperature")
	}
	if gemReq.GenerationConfig.TopP == nil || *gemReq.GenerationConfig.TopP != topP {
		t.Fatalf("expected top_p")
	}
	if gemReq.GenerationConfig.TopK == nil || *gemReq.GenerationConfig.TopK != topK {
		t.Fatalf("expected top_k")
	}
	if gemReq.GenerationConfig.ThinkingConfig != nil {
		t.Fatalf("expected thinking disabled")
	}

	req.OutputConfig = &ClaudeOutputConfig{Effort: "medium"}
	body, _ = json.Marshal(req)
	out, err = conv.Transform(body, "claude-3-5-haiku", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if err := json.Unmarshal(out, &gemReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gemReq.GenerationConfig.EffortLevel != "MEDIUM" {
		t.Fatalf("expected MEDIUM effort")
	}

	req.OutputConfig = &ClaudeOutputConfig{Effort: "weird"}
	body, _ = json.Marshal(req)
	out, err = conv.Transform(body, "claude-3-5-haiku", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if err := json.Unmarshal(out, &gemReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gemReq.GenerationConfig.EffortLevel != "HIGH" {
		t.Fatalf("expected default HIGH effort")
	}
}

func TestClaudeToGeminiRequestThinkingNotFirst(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-3-5-haiku",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "first"},
				"bad",
				map[string]interface{}{"type": "thinking", "thinking": "idea", "signature": "signature123"},
			},
		}},
		Thinking: map[string]interface{}{"type": "enabled", "budget_tokens": float64(10)},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "claude-3-5-haiku", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var gemReq GeminiRequest
	if err := json.Unmarshal(out, &gemReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(gemReq.Contents) == 0 || len(gemReq.Contents[0].Parts) == 0 {
		t.Fatalf("expected parts")
	}
	for _, part := range gemReq.Contents[0].Parts {
		if part.Text == "idea" && part.Thought {
			t.Fatalf("expected downgraded thinking")
		}
	}
}

func TestClaudeToGeminiRequestGoogleSearchTools(t *testing.T) {
	cases := []ClaudeTool{
		{Type: "web_search_20250305"},
		{Name: "web_search"},
	}
	for _, tool := range cases {
		req := ClaudeRequest{
			Model:    "claude-3-5-haiku",
			Messages: []ClaudeMessage{{Role: "user", Content: "hi"}},
			Tools: []ClaudeTool{
				tool,
				{},
			},
		}
		body, _ := json.Marshal(req)
		conv := &claudeToGeminiRequest{}
		out, err := conv.Transform(body, "claude-3-5-haiku", false)
		if err != nil {
			t.Fatalf("Transform: %v", err)
		}
		var gemReq GeminiRequest
		if err := json.Unmarshal(out, &gemReq); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(gemReq.Tools) == 0 || gemReq.Tools[0].GoogleSearch == nil {
			t.Fatalf("expected google search tool")
		}
	}
}

func TestGeminiToClaudeRequestTools(t *testing.T) {
	req := GeminiRequest{
		Tools: []GeminiTool{{
			FunctionDeclarations: []GeminiFunctionDecl{{
				Name:        "tool",
				Description: "desc",
				Parameters:  map[string]interface{}{"type": "object"},
			}},
		}},
		Contents: []GeminiContent{{
			Role:  "user",
			Parts: []GeminiPart{{Text: "hi"}},
		}},
	}
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
	if len(claudeReq.Tools) != 1 {
		t.Fatalf("expected tool conversion")
	}
}

func TestCodexToClaudeRequestRoleDefault(t *testing.T) {
	req := CodexRequest{
		Input: []interface{}{
			"skip",
			map[string]interface{}{
				"type":    "message",
				"content": "hi",
			},
		},
	}
	body, _ := json.Marshal(req)
	conv := &codexToClaudeRequest{}
	out, err := conv.Transform(body, "claude", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var claudeReq ClaudeRequest
	if err := json.Unmarshal(out, &claudeReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(claudeReq.Messages) == 0 || claudeReq.Messages[0].Role != "user" {
		t.Fatalf("expected default role user")
	}
}
