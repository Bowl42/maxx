package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIToGeminiRequest_ReasoningEffort(t *testing.T) {
	req := OpenAIRequest{
		Model:           "gpt-test",
		ReasoningEffort: "medium",
		Messages: []OpenAIMessage{{
			Role:    "user",
			Content: "hi",
		}},
	}
	body, _ := json.Marshal(req)

	conv := &openaiToGeminiRequest{}
	out, err := conv.Transform(body, "gemini-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	var got GeminiRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GenerationConfig == nil || got.GenerationConfig.ThinkingConfig == nil {
		t.Fatal("expected thinkingConfig to be set")
	}
	if got.GenerationConfig.ThinkingConfig.ThinkingLevel != "medium" {
		t.Fatalf("expected thinkingLevel medium, got %q", got.GenerationConfig.ThinkingConfig.ThinkingLevel)
	}
}

func TestGeminiToOpenAIRequest_ThinkingBudget(t *testing.T) {
	req := GeminiRequest{
		GenerationConfig: &GeminiGenerationConfig{
			ThinkingConfig: &GeminiThinkingConfig{
				ThinkingBudget: 1024,
			},
		},
		Contents: []GeminiContent{{
			Role: "user",
			Parts: []GeminiPart{{
				Text: "hi",
			}},
		}},
	}
	body, _ := json.Marshal(req)

	conv := &geminiToOpenAIRequest{}
	out, err := conv.Transform(body, "gpt-test", false)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	var got OpenAIRequest
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ReasoningEffort != "low" {
		t.Fatalf("expected reasoning_effort low, got %q", got.ReasoningEffort)
	}
}

func TestOpenAIToGeminiResponse_StreamReasoning(t *testing.T) {
	conv := &openaiToGeminiResponse{}
	state := NewTransformState()

	chunk := FormatSSE("", []byte(`{"id":"resp-1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"reasoning_content":"think"}}]}`))
	out, err := conv.TransformChunk(chunk, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), `"thought":true`) {
		t.Fatalf("expected thought=true in gemini output, got: %s", string(out))
	}
}

func TestGeminiToOpenAIResponse_StreamThought(t *testing.T) {
	conv := &geminiToOpenAIResponse{}
	state := NewTransformState()

	chunk := FormatSSE("", []byte(`{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"think","thought":true}]}}]}`))
	out, err := conv.TransformChunk(chunk, state)
	if err != nil {
		t.Fatalf("TransformChunk: %v", err)
	}
	if !strings.Contains(string(out), `"reasoning_content"`) {
		t.Fatalf("expected reasoning_content in openai output, got: %s", string(out))
	}
}
