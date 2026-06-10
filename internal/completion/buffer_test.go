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

func TestMoveCursor(t *testing.T) {
	if got := MoveCursorLeft("git commit", 4); got != 3 {
		t.Fatalf("MoveCursorLeft() = %d, want 3", got)
	}
	if got := MoveCursorRight("git commit", 4); got != 5 {
		t.Fatalf("MoveCursorRight() = %d, want 5", got)
	}
}

func TestMoveCursorWord(t *testing.T) {
	if got := MoveCursorWordLeft("git commit --amend", 11); got != 4 {
		t.Fatalf("MoveCursorWordLeft() = %d, want 4", got)
	}
	if got := MoveCursorWordRight("git commit --amend", 4); got != 11 {
		t.Fatalf("MoveCursorWordRight() = %d, want 11", got)
	}
}

func TestDeleteWordBeforeCursor(t *testing.T) {
	gotLine, gotCursor := DeleteWordBeforeCursor("git commit --amend", 11)
	if gotLine != "git --amend" || gotCursor != 4 {
		t.Fatalf("DeleteWordBeforeCursor() = %q, %d; want git --amend, 4", gotLine, gotCursor)
	}
}

func TestDeleteWordAtCursor(t *testing.T) {
	gotLine, gotCursor := DeleteWordAtCursor("git commit --amend", 4)
	if gotLine != "git --amend" || gotCursor != 4 {
		t.Fatalf("DeleteWordAtCursor() = %q, %d; want git --amend, 4", gotLine, gotCursor)
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
