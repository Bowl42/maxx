package codex

import (
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
