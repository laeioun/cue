package completion

import (
	"io/fs"
	"strings"
)

func Complete(specs fs.FS, line string, cursor int) ([]Completion, error) {
	tokens, partial := Parse(line, cursor)
	if len(tokens) == 0 {
		return nil, nil
	}

	root, err := Load(specs, tokens[0])
	if err != nil {
		root, err = ParseHelp(tokens[0])
		if err != nil {
			return nil, err
		}
	}

	return filter(Query(root, tokens), partial), nil
}

func filter(completions []Completion, partial string) []Completion {
	if partial == "" {
		return completions
	}

	filtered := make([]Completion, 0, len(completions))
	for _, completion := range completions {
		if strings.HasPrefix(completion.Name, partial) {
			filtered = append(filtered, completion)
		}
	}
	return filtered
}
