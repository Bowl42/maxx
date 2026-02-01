package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteCountTokensResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeCountTokensResponse(rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var payload map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload["input_tokens"] != 0 || payload["output_tokens"] != 0 {
		t.Fatalf("payload = %v, want zeros", payload)
	}
}
