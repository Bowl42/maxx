package converter

import (
	"encoding/json"
	"testing"
)

func TestClaudeToGeminiRequest_ThinkingDisabledDowngrade(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Thinking: map[string]interface{}{
			"type": "enabled",
		},
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "thinking", "thinking": "t1", "signature": "signature_12345"},
				map[string]interface{}{"type": "text", "text": "hi"},
			},
		}},
	}
	body, _ := json.Marshal(req)
	conv := &claudeToGeminiRequest{}
	out, err := conv.Transform(body, "gpt-4o", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	var got GeminiRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	foundThought := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.Thought {
				foundThought = true
			}
		}
	}
	if foundThought {
		t.Fatalf("expected thinking downgraded to text when target doesn't support thinking")
	}
}

func TestClaudeToGeminiRequest_EmptyThinkingToPlaceholder(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Thinking: map[string]interface{}{
			"type": "enabled",
		},
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "thinking", "thinking": "", "signature": "signature_12345"},
			},
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
	foundPlaceholder := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.Text == "..." {
				foundPlaceholder = true
			}
		}
	}
	if !foundPlaceholder {
		t.Fatalf("expected placeholder text for empty thinking")
	}
}

func TestClaudeToGeminiRequest_ToolResultEmptyIsError(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Messages: []ClaudeMessage{{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "tool_result", "tool_use_id": "call_1", "content": "", "is_error": true},
			},
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
	found := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil && p.FunctionResponse.ID == "call_1" {
				if resp, ok := p.FunctionResponse.Response.(map[string]interface{}); ok {
					if resp["result"] == "Tool execution failed with no output." {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected error placeholder result")
	}
}

func TestClaudeToGeminiRequest_ToolResultEmptySuccess(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Messages: []ClaudeMessage{{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "tool_result", "tool_use_id": "call_1", "content": ""},
			},
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
	found := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil && p.FunctionResponse.ID == "call_1" {
				if resp, ok := p.FunctionResponse.Response.(map[string]interface{}); ok {
					if resp["result"] == "Command executed successfully." {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected success placeholder result")
	}
}

func TestClaudeToGeminiRequest_ThinkingBudgetCapFlash(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Thinking: map[string]interface{}{
			"type":          "enabled",
			"budget_tokens": float64(999999),
		},
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
	if got.GenerationConfig == nil || got.GenerationConfig.ThinkingConfig == nil {
		t.Fatalf("expected thinkingConfig")
	}
	if got.GenerationConfig.ThinkingConfig.ThinkingBudget == 0 {
		t.Fatalf("expected thinking budget set")
	}
}
