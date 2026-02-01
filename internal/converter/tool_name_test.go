package converter

import "testing"

func TestShortenNameIfNeeded(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	short := shortenNameIfNeeded(long)
	if len(short) > maxToolNameLen {
		t.Fatalf("expected shortened length <= %d, got %d", maxToolNameLen, len(short))
	}
	if short == long {
		t.Fatalf("expected shortened name to differ")
	}
}

func TestBuildShortNameMapUniqueness(t *testing.T) {
	names := []string{
		"tool_" + "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmno1",
		"tool_" + "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmno2",
	}
	m := buildShortNameMap(names)
	if len(m) != 2 {
		t.Fatalf("expected 2 entries")
	}
	if m[names[0]] == m[names[1]] {
		t.Fatalf("expected unique shortened names")
	}
}
