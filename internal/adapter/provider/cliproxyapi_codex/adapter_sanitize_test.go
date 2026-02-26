package cliproxyapi_codex

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestSanitizeCodexPayload_MapsAndRemovesUnsupportedFields(t *testing.T) {
	in := []byte(`{
		"model":"gpt-5",
		"max_output_tokens":77,
		"max_completion_tokens":88,
		"temperature":0.2,
		"top_p":0.9,
		"service_tier":"default",
		"input":"hi"
	}`)

	out := sanitizeCodexPayload(in)

	if got := gjson.GetBytes(out, "max_output_tokens").Int(); got != 77 {
		t.Fatalf("max_output_tokens = %d, want 77", got)
	}
	if got := gjson.GetBytes(out, "max_completion_tokens").Int(); got != 88 {
		t.Fatalf("max_completion_tokens = %d, want 88", got)
	}
	if got := gjson.GetBytes(out, "temperature").Float(); got != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", got)
	}
	if got := gjson.GetBytes(out, "top_p").Float(); got != 0.9 {
		t.Fatalf("top_p = %v, want 0.9", got)
	}
	if got := gjson.GetBytes(out, "service_tier").String(); got != "default" {
		t.Fatalf("service_tier = %q, want %q", got, "default")
	}
	if gjson.GetBytes(out, "max_tokens").Exists() {
		t.Fatalf("max_tokens should not be synthesized in local sanitize")
	}
}

func TestSanitizeCodexPayload_KeepExistingMaxTokens(t *testing.T) {
	in := []byte(`{
		"model":"gpt-5",
		"max_tokens":12,
		"max_output_tokens":77,
		"input":"hi"
	}`)

	out := sanitizeCodexPayload(in)

	if got := gjson.GetBytes(out, "max_tokens").Int(); got != 12 {
		t.Fatalf("max_tokens = %d, want 12", got)
	}
	if got := gjson.GetBytes(out, "max_output_tokens").Int(); got != 77 {
		t.Fatalf("max_output_tokens = %d, want 77", got)
	}
}
