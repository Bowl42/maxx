package converter

import (
	"encoding/json"
	"testing"
)

func TestClaudeToGeminiRequest_ToolConfigAndGoogleSearch(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Tools: []ClaudeTool{
			{Name: "do_work", Description: "x", InputSchema: map[string]interface{}{"type": "object"}},
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
	if len(got.Tools) == 0 || got.ToolConfig == nil {
		t.Fatalf("expected tools + toolConfig")
	}
}

func TestClaudeToGeminiRequest_GoogleSearchOnly(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Tools: []ClaudeTool{
			{Type: "web_search_20250305"},
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
	if len(got.Tools) == 0 || got.Tools[0].GoogleSearch == nil {
		t.Fatalf("expected google search tool")
	}
}

func TestClaudeToGeminiRequest_ImageAndDocument(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Messages: []ClaudeMessage{{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "image",
					"source": map[string]interface{}{
						"type":       "base64",
						"media_type": "image/png",
						"data":       "aGVsbG8=",
					},
				},
				map[string]interface{}{
					"type": "document",
					"source": map[string]interface{}{
						"type":       "base64",
						"media_type": "application/pdf",
						"data":       "aGVsbG8=",
					},
				},
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
	if len(got.Contents) == 0 || len(got.Contents[0].Parts) < 2 {
		t.Fatalf("expected inline parts")
	}
}

func TestClaudeToGeminiRequest_RedactedThinkingAndServerTool(t *testing.T) {
	req := ClaudeRequest{
		Model: "claude-opus-4-5-thinking",
		Messages: []ClaudeMessage{{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "redacted_thinking", "data": "secret"},
				map[string]interface{}{"type": "server_tool_use"},
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
	foundRedacted := false
	for _, c := range got.Contents {
		for _, p := range c.Parts {
			if p.Text != "" && p.Text != "..." {
				foundRedacted = true
			}
		}
	}
	if !foundRedacted {
		t.Fatalf("expected redacted thinking text")
	}
}
