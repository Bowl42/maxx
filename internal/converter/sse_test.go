package converter

import "testing"

func TestParseSSEAndDone(t *testing.T) {
	input := "event: message\n" +
		"data: {\"x\":1}\n\n" +
		"data: [DONE]\n\n"
	events, remaining := ParseSSE(input)
	if remaining != "" {
		t.Fatalf("expected no remaining, got %q", remaining)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Event != "message" {
		t.Fatalf("expected event message, got %q", events[0].Event)
	}
	if events[1].Event != "done" {
		t.Fatalf("expected done event, got %q", events[1].Event)
	}
}

func TestIsSSE(t *testing.T) {
	if !IsSSE("data: {\"x\":1}\n\n") {
		t.Fatalf("expected SSE true")
	}
	if IsSSE("{\"x\":1}\n") {
		t.Fatalf("expected SSE false")
	}
}

func TestFormatDone(t *testing.T) {
	if string(FormatDone()) != "data: [DONE]\n\n" {
		t.Fatalf("unexpected done format")
	}
}
