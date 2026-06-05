package aliases

import "testing"

func TestExpandAliasOnlyToken(t *testing.T) {
	line, cursor, ok := Expand("gcm", 3, map[string]string{
		"gcm": "git commit -m",
	})
	if !ok {
		t.Fatal("Expand() did not expand alias")
	}
	if line != "git commit -m " {
		t.Fatalf("line = %q, want %q", line, "git commit -m ")
	}
	if cursor != len(line) {
		t.Fatalf("cursor = %d, want %d", cursor, len(line))
	}
}

func TestExpandAliasWithArguments(t *testing.T) {
	line, cursor, ok := Expand("gcm --a", 7, map[string]string{
		"gcm": "git commit -m",
	})
	if !ok {
		t.Fatal("Expand() did not expand alias")
	}
	if line != "git commit -m --a" {
		t.Fatalf("line = %q, want %q", line, "git commit -m --a")
	}
	if cursor != len(line) {
		t.Fatalf("cursor = %d, want %d", cursor, len(line))
	}
}

func TestExpandIgnoresPartialFirstToken(t *testing.T) {
	line, cursor, ok := Expand("gcm --a", 2, map[string]string{
		"gcm": "git commit -m",
	})
	if ok {
		t.Fatal("Expand() expanded while cursor was inside first token")
	}
	if line != "gcm --a" || cursor != 2 {
		t.Fatalf("Expand() = %q, %d; want original line and cursor", line, cursor)
	}
}
