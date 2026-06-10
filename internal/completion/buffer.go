package completion

import "strings"

// Parse splits the buffer up to cursor into complete tokens and the partial word
// currently being completed.
func Parse(line string, cursor int) (tokens []string, partial string) {
	cursor = clampCursor(line, cursor)
	prefix := line[:cursor]
	tokens = strings.Fields(prefix)
	if len(tokens) == 0 {
		return nil, ""
	}

	if endsWithSpace(prefix) {
		return tokens, ""
	}

	partial = tokens[len(tokens)-1]
	return tokens[:len(tokens)-1], partial
}

// ApplyCompletion returns a new shell buffer with the word at cursor replaced by
// the selected completion.
func ApplyCompletion(line string, cursor int, selected string) string {
	cursor = clampCursor(line, cursor)

	start := cursor
	for start > 0 && !isSpace(line[start-1]) {
		start--
	}

	end := cursor
	for end < len(line) && !isSpace(line[end]) {
		end++
	}

	return line[:start] + selected + line[end:]
}

func DeleteBeforeCursor(line string, cursor int) (string, int) {
	cursor = clampCursor(line, cursor)
	if cursor == 0 {
		return line, cursor
	}
	return line[:cursor-1] + line[cursor:], cursor - 1
}

func DeleteAtCursor(line string, cursor int) (string, int) {
	cursor = clampCursor(line, cursor)
	if cursor >= len(line) {
		return line, cursor
	}
	return line[:cursor] + line[cursor+1:], cursor
}

func InsertAtCursor(line string, cursor int, text string) (string, int) {
	cursor = clampCursor(line, cursor)
	return line[:cursor] + text + line[cursor:], cursor + len(text)
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

func endsWithSpace(s string) bool {
	if s == "" {
		return false
	}
	return isSpace(s[len(s)-1])
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}
