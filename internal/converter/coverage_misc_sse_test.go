package converter

import (
	"strings"
	"testing"
)

func TestIsSSEAdditional(t *testing.T) {
	if !IsSSE("data: {}\n\n") {
		t.Fatalf("expected SSE")
	}
	if IsSSE("hello") {
		t.Fatalf("expected non-SSE")
	}
}

func TestIsSSEEmptyLines(t *testing.T) {
	if IsSSE("\n\n") {
		t.Fatalf("expected false for empty lines")
	}
}

func TestIsSSEEventLine(t *testing.T) {
	if !IsSSE("event: message\n\n") {
		t.Fatalf("expected SSE event")
	}
}

func TestSSE_ParseIncompleteLine(t *testing.T) {
	events, remaining := ParseSSE("data: {\"a\":1}")
	if len(events) != 0 {
		t.Fatalf("expected no events")
	}
	if remaining == "" {
		t.Fatalf("expected remaining buffer")
	}
}

func TestSSE_FormatStringData(t *testing.T) {
	out := FormatSSE("", "hello")
	if !strings.Contains(string(out), "data: hello") {
		t.Fatalf("expected string data")
	}
}
