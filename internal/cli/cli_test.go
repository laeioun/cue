package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/spf13/cobra"
)

func TestRootDisablesCobraCompletionCommand(t *testing.T) {
	rootCmd := New(fstest.MapFS{})
	if !rootCmd.CompletionOptions.DisableDefaultCmd {
		t.Fatal("root command should disable Cobra's default completion command")
	}
}

func TestInitPrintsShellHook(t *testing.T) {
	output, err := executeCommand(New(fstest.MapFS{}), "init", "zsh")
	if err != nil {
		t.Fatalf("init zsh failed: %v", err)
	}
	if !strings.Contains(output, `bindkey '\t' _cue_complete`) {
		t.Fatalf("init zsh output = %q, want zsh hook", output)
	}
}

func TestInstallAddsShellHookOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	output, err := executeCommand(New(fstest.MapFS{}), "install")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !strings.Contains(output, "Added to") {
		t.Fatalf("install output = %q, want added message", output)
	}

	rcFile := filepath.Join(home, ".zshrc")
	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}
	if got := strings.Count(string(content), "cue init zsh"); got != 1 {
		t.Fatalf("cue init zsh count = %d, want 1", got)
	}

	output, err = executeCommand(New(fstest.MapFS{}), "install")
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}
	if !strings.Contains(output, "already present") {
		t.Fatalf("second install output = %q, want already present message", output)
	}

	content, err = os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("read rc file after second install: %v", err)
	}
	if got := strings.Count(string(content), "cue init zsh"); got != 1 {
		t.Fatalf("cue init zsh count after second install = %d, want 1", got)
	}
}

func TestSpecGenerateWritesUserSpec(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "draftcmd"), `#!/bin/sh
cat <<'EOF'
Commands:
  deploy    Deploy the app
  logs      Show logs
EOF
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	output, err := executeCommand(New(fstest.MapFS{}), "spec", "generate", "draftcmd")
	if err != nil {
		t.Fatalf("spec generate failed: %v", err)
	}
	if !strings.Contains(output, "Wrote") {
		t.Fatalf("spec generate output = %q, want written path", output)
	}

	specPath := filepath.Join(configDir, "cue", "specs", "draftcmd.yaml")
	content, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read generated spec: %v", err)
	}
	if !strings.Contains(string(content), "name: deploy") || !strings.Contains(string(content), "description: Deploy the app") {
		t.Fatalf("generated spec = %q, want parsed deploy command", content)
	}
}

func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	var output bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err := cmd.Execute()
	return output.String(), err
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}
