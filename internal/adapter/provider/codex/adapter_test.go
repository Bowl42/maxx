package codex

import (
	"net/http"
	"testing"

	"github.com/awsl-project/maxx/internal/domain"
	"github.com/awsl-project/maxx/internal/flow"
	"github.com/tidwall/gjson"
)

func TestApplyCodexRequestTuning(t *testing.T) {
	c := flow.NewCtx(nil, nil)
	c.Set(flow.KeyOriginalClientType, domain.ClientTypeClaude)
	c.Set(flow.KeyOriginalRequestBody, []byte(`{"metadata":{"user_id":"user-123"}}`))

	body := []byte(`{"model":"gpt-5","stream":false,"instructions":"x","previous_response_id":"r1","prompt_cache_retention":123,"safety_identifier":"s1"}`)
	cacheID, tuned := applyCodexRequestTuning(c, body)

	if cacheID == "" {
		t.Fatalf("expected cacheID to be set")
	}
	if gjson.GetBytes(tuned, "prompt_cache_key").String() == "" {
		t.Fatalf("expected prompt_cache_key to be set")
	}
	if !gjson.GetBytes(tuned, "stream").Bool() {
		t.Fatalf("expected stream=true")
	}
	if gjson.GetBytes(tuned, "previous_response_id").Exists() {
		t.Fatalf("expected previous_response_id to be removed")
	}
	if gjson.GetBytes(tuned, "prompt_cache_retention").Exists() {
		t.Fatalf("expected prompt_cache_retention to be removed")
	}
	if gjson.GetBytes(tuned, "safety_identifier").Exists() {
		t.Fatalf("expected safety_identifier to be removed")
	}
}

func TestApplyCodexHeadersFiltersSensitiveAndPreservesUA(t *testing.T) {
	a := &CodexAdapter{}
	upstreamReq, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/codex/responses", nil)
	clientReq, _ := http.NewRequest("POST", "http://localhost/responses", nil)
	clientReq.Header.Set("User-Agent", "codex-cli-custom/1.2.3")
	clientReq.Header.Set("X-Forwarded-For", "1.2.3.4")
	clientReq.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00")
	clientReq.Header.Set("X-Request-Id", "rid-1")
	clientReq.Header.Set("X-Custom", "ok")

	a.applyCodexHeaders(upstreamReq, clientReq, "token-1", "acct-1", true, "")

	if got := upstreamReq.Header.Get("X-Forwarded-For"); got != "" {
		t.Fatalf("expected X-Forwarded-For filtered, got %q", got)
	}
	if got := upstreamReq.Header.Get("Traceparent"); got != "" {
		t.Fatalf("expected Traceparent filtered, got %q", got)
	}
	if got := upstreamReq.Header.Get("X-Request-Id"); got != "" {
		t.Fatalf("expected X-Request-Id filtered, got %q", got)
	}
	if got := upstreamReq.Header.Get("User-Agent"); got != "codex-cli-custom/1.2.3" {
		t.Fatalf("expected User-Agent passthrough, got %q", got)
	}
	if got := upstreamReq.Header.Get("X-Custom"); got != "ok" {
		t.Fatalf("expected X-Custom passthrough, got %q", got)
	}
}

func TestIsCodexResponseCompletedLine(t *testing.T) {
	if !isCodexResponseCompletedLine("data: {\"type\":\"response.completed\",\"response\":{}}\n") {
		t.Fatal("expected response.completed line to be detected")
	}
	if isCodexResponseCompletedLine("data: {\"type\":\"response.delta\"}\n") {
		t.Fatal("expected non-completed line to be false")
	}
	if isCodexResponseCompletedLine("data: not-json\n") {
		t.Fatal("expected invalid json line to be false")
	}
}
