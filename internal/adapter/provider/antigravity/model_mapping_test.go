package antigravity

import "testing"

func TestDefaultModelMappingRulesOpusWildcardOrder(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "opus 4.6 exact via wildcard",
			input: "claude-opus-4-6",
			want:  "claude-opus-4-6-thinking",
		},
		{
			name:  "opus 4.6 thinking via wildcard",
			input: "claude-opus-4-6-thinking",
			want:  "claude-opus-4-6-thinking",
		},
		{
			name:  "opus 4.5 exact keeps 4.5",
			input: "claude-opus-4-5",
			want:  "claude-opus-4-5-thinking",
		},
		{
			name:  "opus 4.5 thinking keeps 4.5",
			input: "claude-opus-4-5-thinking",
			want:  "claude-opus-4-5-thinking",
		},
		{
			name:  "other opus 4 falls back to 4.6",
			input: "claude-opus-4-7-preview",
			want:  "claude-opus-4-6-thinking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchRulesInOrder(tt.input, defaultModelMappingRules)
			if got != tt.want {
				t.Fatalf("MatchRulesInOrder(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
