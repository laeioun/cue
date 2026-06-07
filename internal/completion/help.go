package completion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	helpLineRE          = regexp.MustCompile(`^\s{2,}([a-z][\w-]+)\s{2,}(.+)$`)
	helpSectionHeaderRE = regexp.MustCompile(`(?i)^\s*(available commands|subcommands|commands):\s*$`)
	sectionHeaderRE     = regexp.MustCompile(`^\s*[A-Za-z][A-Za-z ]+:\s*$`)
)

// ParseHelp builds a simple completion tree from a command's help output and
// caches it until the executable mtime changes.
func ParseHelp(cmdName string) (*Node, error) {
	if err := ValidateCommandName(cmdName); err != nil {
		return nil, err
	}

	path, err := exec.LookPath(cmdName)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	cachePath, err := cachePath(cmdName, path, info.ModTime())
	if err == nil {
		if root, readErr := readCache(cachePath); readErr == nil {
			return root, nil
		}
	}

	root, err := GenerateSpec(cmdName)
	if err != nil {
		return nil, err
	}

	if cachePath != "" {
		_ = writeCache(cachePath, root)
	}
	return root, nil
}

// GenerateSpec runs a command's help fallback chain and returns a draft spec.
func GenerateSpec(cmdName string) (*Node, error) {
	if err := ValidateCommandName(cmdName); err != nil {
		return nil, err
	}
	if _, err := exec.LookPath(cmdName); err != nil {
		return nil, err
	}

	var sawOutput bool
	for _, args := range helpArgs {
		output, err := runHelp(cmdName, args...)
		if err != nil {
			continue
		}
		if strings.TrimSpace(output) != "" {
			sawOutput = true
		}

		root := parseHelpOutput(cmdName, output)
		if len(root.Subcommands) > 0 {
			return root, nil
		}
	}
	if sawOutput {
		return nil, errors.New("no help completions found")
	}
	return nil, errors.New("help command produced no output")
}

var helpArgs = [][]string{
	{"--help"},
	{"-h"},
	{"help"},
}

func runHelp(cmdName string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, args...)
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		return string(output), nil
	}
	if err == nil {
		return "", nil
	}
	return "", err
}

func parseHelpOutput(cmdName, output string) *Node {
	root := &Node{Name: cmdName}
	seen := map[string]bool{}
	lines := strings.Split(output, "\n")
	if hasHelpSection(lines) {
		for _, line := range commandSectionLines(lines) {
			addHelpLine(root, seen, line)
		}
		return root
	}

	for _, line := range lines {
		addHelpLine(root, seen, line)
	}
	return root
}

func hasHelpSection(lines []string) bool {
	for _, line := range lines {
		if helpSectionHeaderRE.MatchString(line) {
			return true
		}
	}
	return false
}

func commandSectionLines(lines []string) []string {
	var sectionLines []string
	inSection := false
	sectionHasCommands := false

	for _, line := range lines {
		switch {
		case helpSectionHeaderRE.MatchString(line):
			inSection = true
			sectionHasCommands = false
			continue
		case !inSection:
			continue
		case strings.TrimSpace(line) == "":
			if sectionHasCommands {
				inSection = false
			}
			continue
		case sectionHeaderRE.MatchString(line) && len(line) == len(strings.TrimLeft(line, " \t")):
			inSection = false
			continue
		}

		sectionLines = append(sectionLines, line)
		sectionHasCommands = true
	}

	return sectionLines
}

func addHelpLine(root *Node, seen map[string]bool, line string) {
	match := helpLineRE.FindStringSubmatch(line)
	if len(match) != 3 || seen[match[1]] {
		return
	}
	seen[match[1]] = true
	root.Subcommands = append(root.Subcommands, &Node{
		Name:        match[1],
		Description: strings.TrimSpace(match[2]),
	})
}

func cachePath(cmdName, executable string, mtime time.Time) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(executable + ":" + mtime.UTC().Format(time.RFC3339Nano)))
	name := cmdName + "-" + hex.EncodeToString(sum[:8]) + "-" + mtime.UTC().Format("20060102150405") + ".json"
	return filepath.Join(base, "cue", name), nil
}

func readCache(path string) (*Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root Node
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return &root, nil
}

func writeCache(path string, root *Node) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
