package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIToCodex_ReasoningEffort(t *testing.T) {
	req := OpenAIRequest{
		Model:           "gpt-test",
		ReasoningEffort: "high",
		Messages: []OpenAIMessage{{
			Role:    "user",
			Content: "hi",
		}},
	}
	body, _ := json.Marshal(req)

	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	var got CodexRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Reasoning == nil || got.Reasoning.Effort != "high" {
		t.Fatalf("expected reasoning.effort high, got %#v", got.Reasoning)
	}
	if got.ParallelToolCalls == nil || !*got.ParallelToolCalls {
		t.Fatalf("expected parallel_tool_calls true")
	}
	if len(got.Include) == 0 {
		t.Fatalf("expected include to be set")
	}
}

func TestCodexToGemini_ReasoningEffort(t *testing.T) {
	req := CodexRequest{
		Model: "codex-test",
		Reasoning: &CodexReasoning{
			Effort: "high",
		},
		Input: "hi",
	}
	body, _ := json.Marshal(req)

	conv := &codexToGeminiRequest{}
	out, err := conv.Transform(body, "gemini-test", false)
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
	if got.GenerationConfig.ThinkingConfig.ThinkingLevel != "high" {
		t.Fatalf("expected thinkingLevel high, got %q", got.GenerationConfig.ThinkingConfig.ThinkingLevel)
	}
}

func TestOpenAIToCodex_ToolNameShortening(t *testing.T) {
	longName := strings.Repeat("verylongtoolname", 5)
	req := OpenAIRequest{
		Model: "gpt-test",
		Tools: []OpenAITool{{
			Type: "function",
			Function: OpenAIFunction{
				Name:        longName,
				Description: "desc",
			},
		}},
		Messages: []OpenAIMessage{{
			Role: "assistant",
			ToolCalls: []OpenAIToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      longName,
					Arguments: "{}",
				},
			}},
		}},
	}
	body, _ := json.Marshal(req)

	conv := &openaiToCodexRequest{}
	out, err := conv.Transform(body, "codex-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	var got CodexRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Tools) == 0 {
		t.Fatalf("expected tools")
	}
	if len(got.Tools[0].Name) > maxToolNameLen {
		t.Fatalf("tool name not shortened: %s", got.Tools[0].Name)
	}
	found := false
	if items, ok := got.Input.([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "function_call" {
					if name, ok := m["name"].(string); ok && name == got.Tools[0].Name {
						found = true
						break
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected function_call name to match shortened tool name")
	}
}
