package aliases

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type config struct {
	Aliases map[string]string `yaml:"aliases"`
}

// Load reads ~/.config/cue/aliases.yaml, returning an empty map when the file
// does not exist.
func Load() (map[string]string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(dir, "cue", "aliases.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	aliases := make(map[string]string, len(cfg.Aliases))
	for key, value := range cfg.Aliases {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		aliases[key] = value
	}
	return aliases, nil
}

// Expand replaces the first token with its configured alias and adjusts the
// cursor to keep it at the same logical point in the command line.
func Expand(line string, cursor int, aliases map[string]string) (string, int, bool) {
	cursor = clampCursor(line, cursor)
	start, end, ok := firstToken(line)
	if !ok || cursor < end {
		return line, cursor, false
	}

	expansion, ok := aliases[line[start:end]]
	if !ok {
		return line, cursor, false
	}
	expansion = strings.TrimSpace(expansion)
	if expansion == "" {
		return line, cursor, false
	}

	expanded := line[:start] + expansion + line[end:]
	newCursor := cursor + len(expansion) - (end - start)
	if cursor == end && end == len(line) {
		expanded += " "
		newCursor++
	}
	return expanded, newCursor, true
}

func firstToken(line string) (int, int, bool) {
	start := 0
	for start < len(line) && isSpace(line[start]) {
		start++
	}
	if start == len(line) {
		return 0, 0, false
	}

	end := start
	for end < len(line) && !isSpace(line[end]) {
		end++
	}
	return start, end, true
}

func clampCursor(line string, cursor int) int {
	if cursor < 0 {
		return 0
	}
	if cursor > len(line) {
		return len(line)
	}
	return cursor
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}
