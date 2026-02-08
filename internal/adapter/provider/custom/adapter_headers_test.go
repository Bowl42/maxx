package custom

import (
	"net/http"
	"testing"
)

func TestCopyHeadersFilteredDropsSensitiveHeaders(t *testing.T) {
	src := make(http.Header)
	src.Set("Host", "example.com")
	src.Set("X-Forwarded-For", "1.2.3.4")
	src.Set("Content-Length", "123")
	src.Set("X-Custom", "ok")

	dst := make(http.Header)
	copyHeadersFiltered(dst, src)

	if dst.Get("Host") != "" {
		t.Fatalf("expected Host to be filtered")
	}
	if dst.Get("X-Forwarded-For") != "" {
		t.Fatalf("expected X-Forwarded-For to be filtered")
	}
	if dst.Get("Content-Length") != "" {
		t.Fatalf("expected Content-Length to be filtered")
	}
	if dst.Get("X-Custom") != "ok" {
		t.Fatalf("expected X-Custom to be preserved")
	}
}

