package completion

import (
	"errors"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

type Completion struct {
	Name        string
	Description string
}

type Node struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Flags       []string `json:"flags,omitempty" yaml:"flags"`
	Subcommands []*Node  `json:"subcommands,omitempty" yaml:"subcommands"`
}

// Load reads the embedded YAML spec for a command name.
func Load(specs fs.FS, cmdName string) (*Node, error) {
	if cmdName == "" || strings.ContainsAny(cmdName, `/\`) {
		return nil, errors.New("invalid command name")
	}

	data, err := fs.ReadFile(specs, cmdName+".yaml")
	if err != nil {
		return nil, err
	}

	var root Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root.Name == "" {
		root.Name = cmdName
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
	for _, token := range tokens[1:] {
		if token == "" || strings.HasPrefix(token, "-") {
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
