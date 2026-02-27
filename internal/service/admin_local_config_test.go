package service

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestMergeCodexAuthJSONPreservesOtherProviders(t *testing.T) {
	existing := []byte(`{
  "OPENAI_API_KEY": "old-root",
  "other": {
    "OPENAI_API_KEY": "old-other"
  },
  "maxx": {
    "OPENAI_API_KEY": "old-maxx",
    "extra": "keep-me"
  }
}`)

	out, recovered, backupNeeded, err := mergeCodexAuthJSON(existing, "maxx", "new-token")
	if err != nil {
		t.Fatalf("mergeCodexAuthJSON returned error: %v", err)
	}
	if recovered {
		t.Fatal("did not expect recovered=true for valid JSON")
	}
	if backupNeeded {
		t.Fatal("did not expect backupNeeded=true for valid JSON")
	}

	doc := map[string]any{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged auth json: %v", err)
	}

	other := mustMap(t, doc["other"])
	if got := mustString(t, other["OPENAI_API_KEY"]); got != "old-other" {
		t.Fatalf("other provider token changed, got %q", got)
	}

	if got := mustString(t, doc["OPENAI_API_KEY"]); got != "new-token" {
		t.Fatalf("root OPENAI_API_KEY not updated, got %q", got)
	}

	maxx := mustMap(t, doc["maxx"])
	if _, exists := maxx["OPENAI_API_KEY"]; exists {
		t.Fatalf("legacy nested token should be removed from maxx object")
	}
	if got := mustString(t, maxx["extra"]); got != "keep-me" {
		t.Fatalf("maxx extra field lost, got %q", got)
	}
}

func TestMergeCodexAuthJSONMigratesLegacyProviderKey(t *testing.T) {
	existing := []byte(`{
  "legacy-provider": {
    "OPENAI_API_KEY": "old-token",
    "note": "keep"
  }
}`)

	out, recovered, backupNeeded, err := mergeCodexAuthJSON(existing, "legacy-provider", "new-token")
	if err != nil {
		t.Fatalf("mergeCodexAuthJSON returned error: %v", err)
	}
	if recovered {
		t.Fatal("did not expect recovered=true for valid JSON")
	}
	if backupNeeded {
		t.Fatal("did not expect backupNeeded=true for valid JSON")
	}

	doc := map[string]any{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged auth json: %v", err)
	}

	if got := mustString(t, doc["OPENAI_API_KEY"]); got != "new-token" {
		t.Fatalf("root OPENAI_API_KEY not updated, got %q", got)
	}

	legacy := mustMap(t, doc["legacy-provider"])
	if _, exists := legacy["OPENAI_API_KEY"]; exists {
		t.Fatalf("legacy nested token should be removed")
	}
	if got := mustString(t, legacy["note"]); got != "keep" {
		t.Fatalf("legacy provider fields should be preserved, got %q", got)
	}
}

func TestMergeCodexAuthJSONPreservesRootFields(t *testing.T) {
	existing := []byte(`{
  "OPENAI_API_KEY": "old",
  "featureFlag": true
}`)

	out, recovered, backupNeeded, err := mergeCodexAuthJSON(existing, "maxx", "new")
	if err != nil {
		t.Fatalf("mergeCodexAuthJSON returned error: %v", err)
	}
	if recovered {
		t.Fatal("did not expect recovered=true for valid JSON")
	}
	if backupNeeded {
		t.Fatal("did not expect backupNeeded=true for valid JSON")
	}

	doc := map[string]any{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged auth json: %v", err)
	}

	if got := mustString(t, doc["OPENAI_API_KEY"]); got != "new" {
		t.Fatalf("root OPENAI_API_KEY not updated, got %q", got)
	}
	if got := mustBool(t, doc["featureFlag"]); !got {
		t.Fatal("featureFlag should remain true")
	}
}

func TestMergeCodexAuthJSONRecoversTrailingComma(t *testing.T) {
	existing := []byte(`{
  "OPENAI_API_KEY": "old",
}`)

	out, recovered, backupNeeded, err := mergeCodexAuthJSON(existing, "maxx", "token")
	if err != nil {
		t.Fatalf("mergeCodexAuthJSON returned error: %v", err)
	}
	if !recovered {
		t.Fatal("expected recovered=true for trailing comma JSON")
	}
	if backupNeeded {
		t.Fatal("did not expect backup when trailing comma is recoverable")
	}

	doc := map[string]any{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged auth json: %v", err)
	}
	if got := mustString(t, doc["OPENAI_API_KEY"]); got != "token" {
		t.Fatalf("root OPENAI_API_KEY not updated, got %q", got)
	}
}

func TestMergeCodexAuthJSONFallsBackForInvalidJSON(t *testing.T) {
	out, recovered, backupNeeded, err := mergeCodexAuthJSON([]byte("{invalid"), "maxx", "token")
	if err != nil {
		t.Fatalf("expected fallback for invalid json, got error: %v", err)
	}
	if !recovered {
		t.Fatal("expected recovered=true for invalid JSON")
	}
	if !backupNeeded {
		t.Fatal("expected backupNeeded=true for unrecoverable JSON")
	}

	doc := map[string]any{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged auth json: %v", err)
	}
	if got := mustString(t, doc["OPENAI_API_KEY"]); got != "token" {
		t.Fatalf("root OPENAI_API_KEY not updated, got %q", got)
	}
}

func TestMergeCodexConfigTOMLPreservesExistingSections(t *testing.T) {
	existing := []byte(`
title = "demo"

[feature]
enabled = true

[model_providers.other]
name = "other"
base_url = "http://other.local"
`)

	out, err := mergeCodexConfigTOML(existing, "maxx", "http://localhost:9880", "")
	if err != nil {
		t.Fatalf("mergeCodexConfigTOML returned error: %v", err)
	}

	doc := map[string]any{}
	if err := toml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged toml: %v", err)
	}

	if got := mustString(t, doc["title"]); got != "demo" {
		t.Fatalf("title changed unexpectedly, got %q", got)
	}

	feature := mustMap(t, doc["feature"])
	if got := mustBool(t, feature["enabled"]); !got {
		t.Fatal("feature.enabled should remain true")
	}

	if got := mustString(t, doc["model_provider"]); got != "maxx" {
		t.Fatalf("model_provider not set, got %q", got)
	}

	modelProviders := mustMap(t, doc["model_providers"])
	other := mustMap(t, modelProviders["other"])
	if got := mustString(t, other["name"]); got != "other" {
		t.Fatalf("other provider name changed, got %q", got)
	}

	maxx := mustMap(t, modelProviders["maxx"])
	if got := mustString(t, maxx["base_url"]); got != "http://localhost:9880" {
		t.Fatalf("maxx base_url mismatch, got %q", got)
	}
	if got := mustString(t, maxx["wire_api"]); got != "responses" {
		t.Fatalf("maxx wire_api mismatch, got %q", got)
	}
}

func TestMergeCodexConfigTOMLRejectsInvalidTOML(t *testing.T) {
	_, err := mergeCodexConfigTOML([]byte("[broken"), "maxx", "http://localhost:9880", "")
	if err == nil {
		t.Fatal("expected error for invalid toml, got nil")
	}
}

func TestMergeCodexConfigTOMLWithModel(t *testing.T) {
	out, err := mergeCodexConfigTOML([]byte(""), "maxx", "http://localhost:9880", "gpt-5.3-codex")
	if err != nil {
		t.Fatalf("mergeCodexConfigTOML returned error: %v", err)
	}

	doc := map[string]any{}
	if err := toml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("failed to unmarshal merged toml: %v", err)
	}

	modelProviders := mustMap(t, doc["model_providers"])
	maxx := mustMap(t, modelProviders["maxx"])
	if got := mustString(t, maxx["model"]); got != "gpt-5.3-codex" {
		t.Fatalf("maxx model mismatch, got %q", got)
	}
}

func TestDeriveRequestBaseURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1/api/admin/local-config/codex/sync", nil)
	req.Host = "127.0.0.1:9880"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "proxy.example.com")

	got := deriveRequestBaseURL(req, ":9880")
	if got != "https://proxy.example.com" {
		t.Fatalf("unexpected base url: %q", got)
	}
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", value)
	}
	return m
}

func mustString(t *testing.T, value any) string {
	t.Helper()
	s, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	return s
}

func mustBool(t *testing.T, value any) bool {
	t.Helper()
	b, ok := value.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", value)
	}
	return b
}
