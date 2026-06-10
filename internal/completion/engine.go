package completion

import (
	"io/fs"
	"sort"
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

	return Filter(Query(root, tokens), partial), nil
}

// FastSelection returns a completion that can be accepted without opening a
// picker.
func FastSelection(completions []Completion, partial string) (string, bool) {
	if len(completions) == 1 {
		return completions[0].Name, true
	}
	return "", false
}

// Filter returns completions that fuzzily match query, ordered by match quality.
func Filter(completions []Completion, query string) []Completion {
	if query == "" {
		return completions
	}

	matches := make([]scoredCompletion, 0, len(completions))
	for _, completion := range completions {
		score, ok := fuzzyScore(completion.Name, query)
		if ok {
			matches = append(matches, scoredCompletion{
				Completion: completion,
				Score:      score,
				Index:      len(matches),
			})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Index < matches[j].Index
		}
		return matches[i].Score > matches[j].Score
	})

	filtered := make([]Completion, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, match.Completion)
	}
	return filtered
}

type scoredCompletion struct {
	Completion Completion
	Score      int
	Index      int
}

func fuzzyScore(value, query string) (int, bool) {
	valueLower := strings.ToLower(value)
	queryLower := strings.ToLower(query)
	if strings.HasPrefix(valueLower, queryLower) {
		return 1000 + len(queryLower)*10 - len(valueLower), true
	}

	score := 0
	last := -1
	contiguous := 0
	for i := 0; i < len(queryLower); i++ {
		idx := strings.IndexByte(valueLower[last+1:], queryLower[i])
		if idx < 0 {
			return 0, false
		}

		pos := last + 1 + idx
		score += 100 - pos
		if pos == last+1 {
			contiguous++
			score += 50 + contiguous*5
		} else {
			contiguous = 0
		}
		if pos == 0 || isWordBoundary(valueLower[pos-1]) {
			score += 20
		}
		last = pos
	}

	return score - len(valueLower), true
}

func isWordBoundary(b byte) bool {
	switch b {
	case '-', '_', '.', '/', ':':
		return true
	default:
		return false
	}
}
