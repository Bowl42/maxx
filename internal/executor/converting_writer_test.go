package executor

import (
	"testing"

	"github.com/awsl-project/maxx/internal/domain"
)

func TestConvertRequestURI(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		from        domain.ClientType
		to          domain.ClientType
		mappedModel string
		isStream    bool
		want        string
	}{
		{
			name:     "same type passthrough",
			original: "/v1/chat/completions",
			from:     domain.ClientTypeOpenAI,
			to:       domain.ClientTypeOpenAI,
			want:     "/v1/chat/completions",
		},
		{
			name:     "openai to claude with query",
			original: "/v1/chat/completions?foo=1",
			from:     domain.ClientTypeOpenAI,
			to:       domain.ClientTypeClaude,
			want:     "/v1/messages?foo=1",
		},
		{
			name:     "claude to codex",
			original: "/v1/messages",
			from:     domain.ClientTypeClaude,
			to:       domain.ClientTypeCodex,
			want:     "/responses",
		},
		{
			name:     "claude count tokens to openai",
			original: "/v1/messages/count_tokens",
			from:     domain.ClientTypeClaude,
			to:       domain.ClientTypeOpenAI,
			want:     "/v1/chat/completions",
		},
		{
			name:        "openai to gemini stream",
			original:    "/v1/chat/completions",
			from:        domain.ClientTypeOpenAI,
			to:          domain.ClientTypeGemini,
			mappedModel: "gemini-2.5-pro",
			isStream:    true,
			want:        "/v1beta/models/gemini-2.5-pro:streamGenerateContent",
		},
		{
			name:        "claude count tokens to gemini",
			original:    "/v1/messages/count_tokens",
			from:        domain.ClientTypeClaude,
			to:          domain.ClientTypeGemini,
			mappedModel: "gemini-2.5-pro",
			want:        "/v1beta/models/gemini-2.5-pro:countTokens",
		},
		{
			name:        "gemini internal preserves version and action",
			original:    "/v1internal/models/gemini-2.0:generateContent?alt=sse",
			from:        domain.ClientTypeOpenAI,
			to:          domain.ClientTypeGemini,
			mappedModel: "gemini-2.5-pro",
			isStream:    true,
			want:        "/v1internal/models/gemini-2.5-pro:generateContent?alt=sse",
		},
		{
			name:     "gemini target without model keeps original",
			original: "/v1/chat/completions",
			from:     domain.ClientTypeOpenAI,
			to:       domain.ClientTypeGemini,
			want:     "/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertRequestURI(tt.original, tt.from, tt.to, tt.mappedModel, tt.isStream)
			if got != tt.want {
				t.Fatalf("ConvertRequestURI(%q) = %q, want %q", tt.original, got, tt.want)
			}
		})
	}
}
