package converter

import (
	"encoding/json"
	"testing"
)

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
