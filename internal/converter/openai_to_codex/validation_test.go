package openai_to_codex

import "testing"

func TestInvalidJSONRequest(t *testing.T) {
	if _, err := (&Request{}).Transform([]byte("{"), "codex", false); err == nil {
		t.Fatalf("expected error for invalid request JSON")
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	if _, err := (&Response{}).Transform([]byte("{")); err == nil {
		t.Fatalf("expected error for invalid response JSON")
	}
}
