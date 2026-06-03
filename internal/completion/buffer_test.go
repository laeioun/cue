package completion

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		cursor      int
		wantTokens  []string
		wantPartial string
	}{
		{
			name:        "partial word",
			line:        "git comm",
			cursor:      8,
			wantTokens:  []string{"git"},
			wantPartial: "comm",
		},
		{
			name:        "trailing space",
			line:        "git commit ",
			cursor:      11,
			wantTokens:  []string{"git", "commit"},
			wantPartial: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTokens, gotPartial := Parse(tt.line, tt.cursor)
			if !equalStrings(gotTokens, tt.wantTokens) {
				t.Fatalf("tokens = %v, want %v", gotTokens, tt.wantTokens)
			}
			if gotPartial != tt.wantPartial {
				t.Fatalf("partial = %q, want %q", gotPartial, tt.wantPartial)
			}
		})
	}
}

func TestApplyCompletion(t *testing.T) {
	got := ApplyCompletion("git comm", 8, "commit")
	if got != "git commit" {
		t.Fatalf("ApplyCompletion() = %q, want %q", got, "git commit")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
