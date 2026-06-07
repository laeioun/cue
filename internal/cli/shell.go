package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var hookScripts = map[string]string{
	"bash": `_cue_complete() {
    local result
    result=$(cue complete "$READLINE_LINE" "$READLINE_POINT" 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$result" ]; then
        READLINE_LINE="$result"
        READLINE_POINT="${#result}"
    fi
}

bind -x '"\t": _cue_complete'
`,
	"fish": `function _cue_complete
    set -l buffer (commandline -b)
    set -l cursor (commandline -C)
    set -l result (cue complete "$buffer" "$cursor" 2>/dev/null)
    set -l exit_status $status
    if test $exit_status -eq 0; and test -n "$result"
        commandline -r "$result"
        commandline -C (string length "$result")
    end
end

bind \t _cue_complete
`,
	"powershell": `Set-PSReadLineKeyHandler -Key Tab -ScriptBlock {
    $line = $null
    $cursor = $null
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    $result = & cue complete $line $cursor 2>$null
    if ($LASTEXITCODE -eq 0 -and $result) {
        [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, $result)
        [Microsoft.PowerShell.PSConsoleReadLine]::SetCursorPosition($result.Length)
    }
}
`,
	"zsh": `_cue_complete() {
    local result
    result=$(cue complete "$BUFFER" "$CURSOR" 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$result" ]; then
        BUFFER="$result"
        CURSOR="${#BUFFER}"
    fi
    zle reset-prompt
}

zle -N _cue_complete
bindkey '\t' _cue_complete
`,
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <shell>",
		Short: "Print shell integration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := normalizeShell(args[0])
			script, ok := hookScripts[shell]
			if !ok {
				return fmt.Errorf("unsupported shell %q (supported: %s)", args[0], strings.Join(supportedShells(), ", "))
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), script)
			return err
		},
	}
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [shell]",
		Short: "Install shell integration automatically",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installShellIntegration(cmd, args)
		},
	}
}

func installShellIntegration(cmd *cobra.Command, args []string) error {
	shell := ""
	if len(args) > 0 {
		shell = normalizeShell(args[0])
	} else {
		shell = detectShell()
	}
	if strings.TrimSpace(shell) == "" {
		return errors.New("could not detect shell; run cue install <shell> or cue init <shell> manually")
	}
	if _, ok := hookScripts[shell]; !ok {
		return fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(supportedShells(), ", "))
	}

	rcFile, err := rcFileForShell(shell)
	if err != nil {
		return err
	}
	line := installLine(shell)

	content, err := os.ReadFile(rcFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if strings.Contains(string(content), "cue init") {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "cue integration already present in %s\n", rcFile)
		return err
	}

	if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added to %s - restart your shell or run: %s\n", rcFile, reloadCommand(shell, rcFile))
	return err
}

func detectShell() string {
	shellEnv := os.Getenv("SHELL")
	if strings.TrimSpace(shellEnv) != "" {
		return normalizeShell(shellEnv)
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return ""
}

func normalizeShell(shell string) string {
	if strings.TrimSpace(shell) == "" {
		return ""
	}
	shell = strings.ToLower(strings.TrimSuffix(filepath.Base(shell), ".exe"))
	switch shell {
	case "pwsh":
		return "powershell"
	default:
		return shell
	}
}

func supportedShells() []string {
	shells := make([]string, 0, len(hookScripts))
	for shell := range hookScripts {
		shells = append(shells, shell)
	}
	sort.Strings(shells)
	return shells
}

func rcFileForShell(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	case "powershell":
		return powershellProfile(home), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shell, strings.Join(supportedShells(), ", "))
	}
}

func powershellProfile(home string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
}

func installLine(shell string) string {
	switch shell {
	case "fish":
		return "\ncue init fish | source\n"
	case "powershell":
		return "\nInvoke-Expression (& { (cue init powershell | Out-String) })\n"
	default:
		return fmt.Sprintf("\neval \"$(cue init %s)\"\n", shell)
	}
}

func reloadCommand(shell, rcFile string) string {
	if shell == "powershell" {
		return ". " + rcFile
	}
	return "source " + rcFile
}
