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
        if [[ "$result" == __cue_cursor__:* ]]; then
            local payload="${result#__cue_cursor__:}"
            READLINE_POINT="${payload%%:*}"
            READLINE_LINE="${payload#*:}"
        else
            READLINE_LINE="$result"
            READLINE_POINT="${#result}"
        fi
    else
        local word="${READLINE_LINE:0:$READLINE_POINT}"
        word="${word##* }"
        local matches
        mapfile -t matches < <(compgen -f -- "$word" 2>/dev/null)
        if [ "${#matches[@]}" -eq 1 ]; then
            local prefix="${READLINE_LINE:0:$((READLINE_POINT - ${#word}))}"
            READLINE_LINE="${prefix}${matches[0]}"
            READLINE_POINT="${#READLINE_LINE}"
        fi
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
        if string match -q "__cue_cursor__:*" -- "$result"
            set -l payload (string replace -r "^__cue_cursor__:" "" -- "$result")
            set -l parts (string split -m 1 ":" -- "$payload")
            commandline -r "$parts[2]"
            commandline -C "$parts[1]"
        else
            commandline -r "$result"
            commandline -C (string length "$result")
        end
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
        $prefix = "__cue_cursor__:"
        if ($result.StartsWith($prefix)) {
            $payload = $result.Substring($prefix.Length)
            $separator = $payload.IndexOf(":")
            if ($separator -ge 0) {
                $nextCursor = [int]$payload.Substring(0, $separator)
                $nextLine = $payload.Substring($separator + 1)
                [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, $nextLine)
                [Microsoft.PowerShell.PSConsoleReadLine]::SetCursorPosition($nextCursor)
            }
        } else {
            [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, $result)
            [Microsoft.PowerShell.PSConsoleReadLine]::SetCursorPosition($result.Length)
        }
    } else {
        [Microsoft.PowerShell.PSConsoleReadLine]::TabCompleteNext()
    }
}
`,
	"zsh": `_cue_complete() {
    local result
    result=$(cue complete "$BUFFER" "$CURSOR" 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$result" ]; then
        if [[ "$result" == __cue_cursor__:* ]]; then
            local payload="${result#__cue_cursor__:}"
            CURSOR="${payload%%:*}"
            BUFFER="${payload#*:}"
        else
            BUFFER="$result"
            CURSOR="${#BUFFER}"
        fi
        zle reset-prompt
    else
        zle expand-or-complete
    fi
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
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "cue integration already present in %s - reload your shell or run: %s\n", rcFile, reloadCommand(shell, rcFile))
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
