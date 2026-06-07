package completion

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestLoadPrefersUserSpec(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	specDir := filepath.Join(configDir, "cue", "specs")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "tool.yaml"), []byte(`name: tool
subcommands:
  - name: user
    description: User spec command
`), 0o644); err != nil {
		t.Fatalf("write user spec: %v", err)
	}

	specs := fstest.MapFS{
		"tool.yaml": {
			Data: []byte(`name: tool
subcommands:
  - name: embedded
    description: Embedded spec command
`),
		},
	}

	got, err := Load(specs, "tool")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	assertSubcommands(t, got, []string{"user"}, []string{"User spec command"})
}

func TestBundledSpecsDecode(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	specs := os.DirFS("../../specs")

	for _, name := range []string{"cargo", "docker"} {
		t.Run(name, func(t *testing.T) {
			got, err := Load(specs, name)
			if err != nil {
				t.Fatalf("Load(%q) error = %v", name, err)
			}
			if len(got.Subcommands) == 0 {
				t.Fatalf("Load(%q) returned no subcommands", name)
			}
		})
	}
}
