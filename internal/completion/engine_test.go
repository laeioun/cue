package completion

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestCompleteFromSpec(t *testing.T) {
	specs := fstest.MapFS{
		"git.yaml": {
			Data: []byte(`name: git
subcommands:
  - name: commit
    description: Record changes
    flags: [--message, -m]
  - name: checkout
    description: Switch branches
`),
		},
	}

	got, err := Complete(specs, "git com", 7)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "commit" {
		t.Fatalf("Complete() = %v, want commit", got)
	}
}

func TestCompleteFallsBackWhenEmbeddedSpecIsEmpty(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "emptycmd"), `#!/bin/sh
cat <<'EOF'
Commands:
  build    Build the project
EOF
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	specs := fstest.MapFS{
		"emptycmd.yaml": {Data: []byte("")},
	}

	got, err := Complete(specs, "emptycmd bu", 11)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "build" {
		t.Fatalf("Complete() = %v, want build from help fallback", got)
	}
}

func TestQueryNestedFlags(t *testing.T) {
	root := &Node{
		Name: "git",
		Subcommands: []*Node{
			{
				Name:  "commit",
				Flags: []string{"--message", "-m"},
			},
		},
	}

	got := Query(root, []string{"git", "commit"})
	if len(got) != 2 || got[0].Name != "--message" || got[1].Name != "-m" {
		t.Fatalf("Query() = %v, want commit flags", got)
	}
}

func TestQuerySkipsAlreadyUsedFlags(t *testing.T) {
	root := &Node{
		Name: "git",
		Subcommands: []*Node{
			{
				Name:  "commit",
				Flags: []string{"--message", "-m", "--amend"},
			},
		},
	}

	got := Query(root, []string{"git", "commit", "-m"})
	if len(got) != 2 || got[0].Name != "--message" || got[1].Name != "--amend" {
		t.Fatalf("Query() = %v, want unused commit flags", got)
	}
}

func TestFastSelectionSingleCompletion(t *testing.T) {
	got, ok := FastSelection([]Completion{{Name: "commit"}}, "com")
	if !ok || got != "commit" {
		t.Fatalf("FastSelection() = %q, %v; want commit, true", got, ok)
	}
}

func TestFastSelectionRejectsAmbiguousShortPrefix(t *testing.T) {
	got, ok := FastSelection([]Completion{
		{Name: "commit"},
		{Name: "config"},
	}, "co")
	if ok || got != "" {
		t.Fatalf("FastSelection() = %q, %v; want empty, false", got, ok)
	}
}
