package custom

import (
	"net/http"
	"testing"
)

func TestApplyCodexHeadersUserAgentPassthroughOnlyForCLI(t *testing.T) {
	upstreamReq, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", nil)
	clientReq, _ := http.NewRequest("POST", "http://localhost/responses", nil)
	clientReq.Header.Set("User-Agent", "codex-cli/1.2.3")

	applyCodexHeaders(upstreamReq, clientReq, "token-1")
	if got := upstreamReq.Header.Get("User-Agent"); got != "codex-cli/1.2.3" {
		t.Fatalf("expected CLI User-Agent passthrough, got %q", got)
	}

	upstreamReq2, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", nil)
	clientReq2, _ := http.NewRequest("POST", "http://localhost/responses", nil)
	clientReq2.Header.Set("User-Agent", "Mozilla/5.0")

	applyCodexHeaders(upstreamReq2, clientReq2, "token-1")
	if got := upstreamReq2.Header.Get("User-Agent"); got != codexUserAgent {
		t.Fatalf("expected default User-Agent for non-CLI client, got %q", got)
	}
}

func TestApplyCodexHeadersDoesNotPassthroughLookalikeCLIUA(t *testing.T) {
	upstreamReq, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", nil)
	clientReq, _ := http.NewRequest("POST", "http://localhost/responses", nil)
	clientReq.Header.Set("User-Agent", "codex-climax/1.2.3")

	applyCodexHeaders(upstreamReq, clientReq, "token-1")
	if got := upstreamReq.Header.Get("User-Agent"); got != codexUserAgent {
		t.Fatalf("expected default User-Agent for lookalike non-CLI UA, got %q", got)
	}
}
