package client

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/awsl-project/maxx/internal/domain"
)

func TestDetectClientTypePrefersClaudeUserAgent(t *testing.T) {
	adapter := NewAdapter()
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)

	req := httptest.NewRequest("POST", "/unknown", strings.NewReader(string(body)))
	req.Header.Set("User-Agent", "claude-cli/2.0")
	if got := adapter.DetectClientType(req, body); got != domain.ClientTypeClaude {
		t.Fatalf("client type = %s, want %s", got, domain.ClientTypeClaude)
	}

	req = httptest.NewRequest("POST", "/unknown", strings.NewReader(string(body)))
	req.Header.Set("User-Agent", "curl/7.0")
	if got := adapter.DetectClientType(req, body); got != domain.ClientTypeOpenAI {
		t.Fatalf("client type = %s, want %s", got, domain.ClientTypeOpenAI)
	}

	req = httptest.NewRequest("POST", "/unknown", strings.NewReader(string(body)))
	req.Header.Set("User-Agent", " Claude-cli/2.0")
	if got := adapter.DetectClientType(req, body); got != domain.ClientTypeOpenAI {
		t.Fatalf("client type = %s, want %s", got, domain.ClientTypeOpenAI)
	}
}
