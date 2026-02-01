package converter

import (
	"encoding/json"
	"testing"
)

func TestOpenAIToGeminiRequest_ToolChoiceStrings(t *testing.T) {
	cases := []struct {
		name  string
		value string
		mode  string
	}{
		{name: "none", value: "none", mode: "NONE"},
		{name: "auto", value: "auto", mode: "AUTO"},
		{name: "required", value: "required", mode: "ANY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := OpenAIRequest{
				Model:      "gpt-test",
				ToolChoice: tc.value,
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
			if got.ToolConfig == nil || got.ToolConfig.FunctionCallingConfig == nil {
				t.Fatalf("expected toolConfig")
			}
			if got.ToolConfig.FunctionCallingConfig.Mode != tc.mode {
				t.Fatalf("expected mode %s, got %q", tc.mode, got.ToolConfig.FunctionCallingConfig.Mode)
			}
		})
	}
}

func TestGeminiToOpenAIRequest_InlineDataContentParts(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{{
			Role: "user",
			Parts: []GeminiPart{
				{Text: "hello"},
				{InlineData: &GeminiInlineData{MimeType: "image/png", Data: "aGVsbG8="}},
			},
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
	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	if _, ok := got.Messages[0].Content.([]interface{}); !ok {
		t.Fatalf("expected content parts array, got %#v", got.Messages[0].Content)
	}
}

func TestGeminiToOpenAIRequest_FunctionResponseNameSplit(t *testing.T) {
	req := GeminiRequest{
		Contents: []GeminiContent{{
			Role: "user",
			Parts: []GeminiPart{{
				FunctionResponse: &GeminiFunctionResponse{
					Name:     "search_call_123",
					Response: map[string]interface{}{"result": "ok"},
				},
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
	if len(got.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got.Messages))
	}
	msg := got.Messages[0]
	if msg.Role != "tool" {
		t.Fatalf("expected tool message, got %q", msg.Role)
	}
	if msg.ToolCallID != "call_123" {
		t.Fatalf("expected tool_call_id call_123, got %q", msg.ToolCallID)
	}
	if msg.Name != "search" {
		t.Fatalf("expected name search, got %q", msg.Name)
	}
}

func TestOpenAIToGeminiRequest_ImageURLInlineData(t *testing.T) {
	req := OpenAIRequest{
		Model: "gpt-test",
		Messages: []OpenAIMessage{{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "data:image/png;base64,aGVsbG8=",
					},
				},
			},
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
	if len(got.Contents) == 0 || len(got.Contents[0].Parts) == 0 || got.Contents[0].Parts[0].InlineData == nil {
		t.Fatalf("expected inlineData from image_url")
	}
}
