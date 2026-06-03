package completion

import (
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
