package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHelpOutputCommandSections(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantNames  []string
		wantDesc   []string
		wantAbsent string
	}{
		{
			name: "cobra available commands",
			output: `Usage:
  gh <command>

Available Commands:
  auth        Authenticate with GitHub
  repo        Manage repositories

Flags:
  -h, --help   help for gh
`,
			wantNames: []string{"auth", "repo"},
			wantDesc:  []string{"Authenticate with GitHub", "Manage repositories"},
		},
		{
			name: "clap subcommands",
			output: `Usage: cargo <COMMAND>

SUBCOMMANDS:
    build       Compile the current package
    test        Run the tests

OPTIONS:
    -h, --help  Print help
`,
			wantNames: []string{"build", "test"},
			wantDesc:  []string{"Compile the current package", "Run the tests"},
		},
		{
			name: "click commands",
			output: `Usage: pip [OPTIONS] COMMAND [ARGS]...

Commands:
  install    Install packages
  uninstall  Uninstall packages
`,
			wantNames: []string{"install", "uninstall"},
			wantDesc:  []string{"Install packages", "Uninstall packages"},
		},
		{
			name: "uppercase commands",
			output: `COMMANDS:
  apply    Builds or changes infrastructure
  plan     Show changes required
`,
			wantNames: []string{"apply", "plan"},
			wantDesc:  []string{"Builds or changes infrastructure", "Show changes required"},
		},
		{
			name: "section suppresses unrelated global matches",
			output: `Usage:
  myapp <command>

Examples:
  myapp       This looks command-like but is not a command

Commands:
  start       Start the service
`,
			wantNames:  []string{"start"},
			wantDesc:   []string{"Start the service"},
			wantAbsent: "myapp",
		},
		{
			name: "legacy two column fallback",
			output: `usage: tool <command>
  alpha       First command
  beta        Second command
`,
			wantNames: []string{"alpha", "beta"},
			wantDesc:  []string{"First command", "Second command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHelpOutput("tool", tt.output)
			assertSubcommands(t, got, tt.wantNames, tt.wantDesc)
			if tt.wantAbsent != "" && hasSubcommand(got, tt.wantAbsent) {
				t.Fatalf("parseHelpOutput() unexpectedly included %q", tt.wantAbsent)
			}
		})
	}
}

func assertSubcommands(t *testing.T, got *Node, wantNames, wantDesc []string) {
	t.Helper()
	if len(got.Subcommands) != len(wantNames) {
		t.Fatalf("subcommands = %d, want %d (%v)", len(got.Subcommands), len(wantNames), got.Subcommands)
	}
	for i, want := range wantNames {
		if got.Subcommands[i].Name != want {
			t.Fatalf("subcommand[%d].Name = %q, want %q", i, got.Subcommands[i].Name, want)
		}
		if got.Subcommands[i].Description != wantDesc[i] {
			t.Fatalf("subcommand[%d].Description = %q, want %q", i, got.Subcommands[i].Description, wantDesc[i])
		}
	}
}

func hasSubcommand(root *Node, name string) bool {
	for _, child := range root.Subcommands {
		if child.Name == name {
			return true
		}
	}
	return false
}

func TestGenerateSpecFallsBackWhenHelpDoesNotParse(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "fallbackcmd"), `#!/bin/sh
case "$1" in
  --help)
    echo "usage only"
    ;;
  -h)
    cat <<'EOF'
Commands:
  build    Build the project
EOF
    ;;
  *)
    exit 1
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := GenerateSpec("fallbackcmd")
	if err != nil {
		t.Fatalf("GenerateSpec() error = %v", err)
	}
	assertSubcommands(t, got, []string{"build"}, []string{"Build the project"})
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}
