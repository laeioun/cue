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

var helpLineRE = regexp.MustCompile(`^\s{2,}([a-z][\w-]+)\s{2,}(.+)$`)

// ParseHelp builds a simple completion tree from a command's help output and
// caches it until the executable mtime changes.
func ParseHelp(cmdName string) (*Node, error) {
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

	output, err := runHelp(cmdName)
	if err != nil {
		return nil, err
	}

	root := parseHelpOutput(cmdName, output)
	if len(root.Subcommands) == 0 {
		return nil, errors.New("no help completions found")
	}

	if cachePath != "" {
		_ = writeCache(cachePath, root)
	}
	return root, nil
}

func runHelp(cmdName string) (string, error) {
	tries := [][]string{
		{cmdName, "--help"},
		{cmdName, "-h"},
		{cmdName, "help"},
	}

	for _, args := range tries {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		output, err := cmd.CombinedOutput()
		cancel()

		if len(output) > 0 {
			return string(output), nil
		}
		if err == nil {
			return "", nil
		}
	}
	return "", errors.New("help command produced no output")
}

func parseHelpOutput(cmdName, output string) *Node {
	root := &Node{Name: cmdName}
	seen := map[string]bool{}

	for _, line := range strings.Split(output, "\n") {
		match := helpLineRE.FindStringSubmatch(line)
		if len(match) != 3 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		root.Subcommands = append(root.Subcommands, &Node{
			Name:        match[1],
			Description: strings.TrimSpace(match[2]),
		})
	}
	return root
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
