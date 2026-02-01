package converter

import (
	"testing"
)

func TestValidation_InvalidJSONRequests(t *testing.T) {
	reqs := []struct {
		name string
		err  error
	}{
		{"claude_to_gemini", func() error {
			_, err := (&claudeToGeminiRequest{}).Transform([]byte("{"), "gemini", false)
			return err
		}()},
		{"openai_to_codex", func() error {
			_, err := (&openaiToCodexRequest{}).Transform([]byte("{"), "codex", false)
			return err
		}()},
		{"codex_to_openai", func() error {
			_, err := (&codexToOpenAIRequest{}).Transform([]byte("{"), "gpt", false)
			return err
		}()},
		{"codex_to_claude", func() error {
			_, err := (&codexToClaudeRequest{}).Transform([]byte("{"), "claude", false)
			return err
		}()},
		{"openai_to_claude", func() error {
			_, err := (&openaiToClaudeRequest{}).Transform([]byte("{"), "claude", false)
			return err
		}()},
		{"gemini_to_claude", func() error {
			_, err := (&geminiToClaudeRequest{}).Transform([]byte("{"), "claude", false)
			return err
		}()},
		{"claude_to_codex", func() error {
			_, err := (&claudeToCodexRequest{}).Transform([]byte("{"), "codex", false)
			return err
		}()},
		{"codex_to_gemini", func() error {
			_, err := (&codexToGeminiRequest{}).Transform([]byte("{"), "gemini", false)
			return err
		}()},
		{"gemini_to_codex", func() error {
			_, err := (&geminiToCodexRequest{}).Transform([]byte("{"), "codex", false)
			return err
		}()},
		{"gemini_to_openai", func() error {
			_, err := (&geminiToOpenAIRequest{}).Transform([]byte("{"), "gpt", false)
			return err
		}()},
		{"openai_to_gemini", func() error {
			_, err := (&openaiToGeminiRequest{}).Transform([]byte("{"), "gemini", false)
			return err
		}()},
		{"claude_to_openai", func() error {
			_, err := (&claudeToOpenAIRequest{}).Transform([]byte("{"), "gpt", false)
			return err
		}()},
	}
	for _, item := range reqs {
		if item.err == nil {
			t.Fatalf("expected error for %s", item.name)
		}
	}
}

func TestValidation_InvalidJSONResponses(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"openai_to_codex", func() error {
			_, err := (&openaiToCodexResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"openai_to_claude", func() error {
			_, err := (&openaiToClaudeResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"claude_to_gemini", func() error {
			_, err := (&claudeToGeminiResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"gemini_to_claude", func() error {
			_, err := (&geminiToClaudeResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"claude_to_openai", func() error {
			_, err := (&claudeToOpenAIResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"claude_to_codex", func() error {
			_, err := (&claudeToCodexResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"codex_to_gemini", func() error {
			_, err := (&codexToGeminiResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"gemini_to_codex", func() error {
			_, err := (&geminiToCodexResponse{}).Transform([]byte("{"))
			return err
		}()},
		{"gemini_to_openai", func() error {
			_, err := (&geminiToOpenAIResponse{}).Transform([]byte("{"))
			return err
		}()},
	}
	for _, item := range cases {
		if item.err == nil {
			t.Fatalf("expected error for %s", item.name)
		}
	}
}
