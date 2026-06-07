package completion

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var errEmptySpec = errors.New("empty spec")

type Completion struct {
	Name        string
	Description string
}

type Node struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description,omitempty"`
	Flags       []string `json:"flags,omitempty" yaml:"flags,omitempty"`
	Subcommands []*Node  `json:"subcommands,omitempty" yaml:"subcommands,omitempty"`
}

// Load reads the embedded YAML spec for a command name.
func Load(specs fs.FS, cmdName string) (*Node, error) {
	if err := ValidateCommandName(cmdName); err != nil {
		return nil, err
	}

	if root, err := loadUserSpec(cmdName); err == nil {
		return root, nil
	} else if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errEmptySpec) {
		return nil, err
	}

	data, err := fs.ReadFile(specs, cmdName+".yaml")
	if err != nil {
		return nil, err
	}
	return decodeSpec(cmdName, data)
}

// ValidateCommandName rejects names that could escape the specs directory.
func ValidateCommandName(cmdName string) error {
	if cmdName == "" || strings.ContainsAny(cmdName, `/\`) {
		return errors.New("invalid command name")
	}
	return nil
}

// UserSpecPath returns where a generated or user-authored spec should live.
func UserSpecPath(cmdName string) (string, error) {
	if err := ValidateCommandName(cmdName); err != nil {
		return "", err
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cue", "specs", cmdName+".yaml"), nil
}

func loadUserSpec(cmdName string) (*Node, error) {
	path, err := UserSpecPath(cmdName)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeSpec(cmdName, data)
}

func decodeSpec(cmdName string, data []byte) (*Node, error) {
	var root Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root.Name == "" {
		root.Name = cmdName
	}
	if len(root.Subcommands) == 0 && len(root.Flags) == 0 {
		return nil, errEmptySpec
	}
	return &root, nil
}

// Query walks the node tree following complete tokens and returns valid next
// completions from the current node.
func Query(root *Node, tokens []string) []Completion {
	if root == nil {
		return nil
	}

	current := root
	usedFlags := map[string]bool{}
	for _, token := range tokens[1:] {
		if token == "" {
			continue
		}
		if strings.HasPrefix(token, "-") {
			usedFlags[token] = true
			continue
		}
		if child := current.child(token); child != nil {
			current = child
		}
	}

	completions := make([]Completion, 0, len(current.Subcommands)+len(current.Flags))
	for _, child := range current.Subcommands {
		completions = append(completions, Completion{
			Name:        child.Name,
			Description: child.Description,
		})
	}
	for _, flag := range current.Flags {
		if usedFlags[flag] {
			continue
		}
		completions = append(completions, Completion{Name: flag})
	}
	return completions
}

func (n *Node) child(name string) *Node {
	for _, child := range n.Subcommands {
		if child.Name == name {
			return child
		}
	}
	return nil
}
