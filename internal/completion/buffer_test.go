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

func TestDeleteBeforeCursor(t *testing.T) {
	gotLine, gotCursor := DeleteBeforeCursor("git comm", 8)
	if gotLine != "git com" || gotCursor != 7 {
		t.Fatalf("DeleteBeforeCursor() = %q, %d; want git com, 7", gotLine, gotCursor)
	}
}

func TestDeleteAtCursor(t *testing.T) {
	gotLine, gotCursor := DeleteAtCursor("git commit", 4)
	if gotLine != "git ommit" || gotCursor != 4 {
		t.Fatalf("DeleteAtCursor() = %q, %d; want git ommit, 4", gotLine, gotCursor)
	}
}

func TestInsertAtCursor(t *testing.T) {
	gotLine, gotCursor := InsertAtCursor("git ", 4, "commit ")
	if gotLine != "git commit " || gotCursor != 11 {
		t.Fatalf("InsertAtCursor() = %q, %d; want git commit, 11", gotLine, gotCursor)
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
