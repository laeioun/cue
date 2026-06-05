package picker

import (
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/laeioun/cue/internal/completion"
)

const (
	maxVisible = 5
	nameWidth  = 18
)

func Run(completions []completion.Completion) (string, error) {
	if len(completions) == 0 {
		return "", nil
	}

	tty, err := openTTY()
	if err != nil {
		return "", err
	}
	defer tty.Close()

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(tty.Fd()), oldState)

	p := inlinePicker{
		tty:         tty,
		completions: completions,
	}
	if err := p.render(); err != nil {
		return "", err
	}
	defer p.clear()

	for {
		key, err := readKey(tty)
		if err != nil {
			return "", err
		}
		switch key {
		case keyAccept:
			return completions[p.selected].Name, nil
		case keyCancel:
			return "", nil
		case keyUp:
			if p.selected == 0 {
				p.selected = len(completions) - 1
			} else {
				p.selected--
			}
			if err := p.render(); err != nil {
				return "", err
			}
		case keyDown:
			p.selected = (p.selected + 1) % len(completions)
			if err := p.render(); err != nil {
				return "", err
			}
		}
	}
}

type inlinePicker struct {
	tty         *os.File
	completions []completion.Completion
	selected    int
	offset      int
}

func (p *inlinePicker) render() error {
	p.ensureVisible()
	width, _, err := term.GetSize(int(p.tty.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}

	var b strings.Builder
	b.WriteString("\x1b[s\x1b[?25l\x1b[1B\r")
	for i := 0; i < maxVisible; i++ {
		b.WriteString("\x1b[2K")
		if idx := p.offset + i; idx < len(p.completions) {
			selected := idx == p.selected
			if selected {
				b.WriteString("\x1b[7m")
			}
			b.WriteString(fitLine(formatCompletion(p.completions[idx]), width))
			if selected {
				b.WriteString("\x1b[0m")
			}
		}
		if i < maxVisible-1 {
			b.WriteString("\x1b[1B\r")
		}
	}
	b.WriteString("\x1b[u\x1b[?25h")

	_, err = p.tty.WriteString(b.String())
	return err
}

func (p *inlinePicker) clear() {
	var b strings.Builder
	b.WriteString("\x1b[s\x1b[?25l\x1b[1B\r")
	for i := 0; i < maxVisible; i++ {
		b.WriteString("\x1b[2K")
		if i < maxVisible-1 {
			b.WriteString("\x1b[1B\r")
		}
	}
	b.WriteString("\x1b[u\x1b[?25h")
	_, _ = p.tty.WriteString(b.String())
}

func (p *inlinePicker) ensureVisible() {
	if p.selected < p.offset {
		p.offset = p.selected
	}
	if p.selected >= p.offset+maxVisible {
		p.offset = p.selected - maxVisible + 1
	}
}

func formatCompletion(c completion.Completion) string {
	if c.Description == "" {
		return "  " + c.Name
	}
	return "  " + padRight(c.Name, nameWidth) + " " + c.Description
}

func fitLine(line string, width int) string {
	if len(line) <= width {
		return line
	}
	if width <= 1 {
		return line[:width]
	}
	return line[:width-1] + "~"
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func openTTY() (*os.File, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		return tty, nil
	}
	if tty, err = os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		return tty, nil
	}
	return os.OpenFile("CONOUT$", os.O_RDWR, 0)
}

type key int

const (
	keyUnknown key = iota
	keyAccept
	keyCancel
	keyUp
	keyDown
)

func readKey(tty *os.File) (key, error) {
	var b [1]byte
	if _, err := tty.Read(b[:]); err != nil {
		return keyUnknown, err
	}

	switch b[0] {
	case '\r', '\n', '\t':
		return keyAccept, nil
	case 0x03:
		return keyCancel, nil
	case 0x10, 'k':
		return keyUp, nil
	case 0x0e, 'j':
		return keyDown, nil
	case 0x1b:
		seq := readEscapeSequence(tty)
		if len(seq) >= 2 && seq[0] == '[' {
			switch seq[1] {
			case 'A':
				return keyUp, nil
			case 'B':
				return keyDown, nil
			}
		}
		return keyCancel, nil
	default:
		return keyUnknown, nil
	}
}

func readEscapeSequence(tty *os.File) []byte {
	seq := make([]byte, 0, 8)
	for len(seq) < cap(seq) {
		b, ok := readByteWithTimeout(tty, 15*time.Millisecond)
		if !ok {
			break
		}
		seq = append(seq, b)
		if len(seq) >= 2 && seq[0] == '[' {
			break
		}
	}
	return seq
}

func readByteWithTimeout(tty *os.File, timeout time.Duration) (byte, bool) {
	type result struct {
		b   byte
		err error
	}

	ch := make(chan result, 1)
	go func() {
		var b [1]byte
		_, err := tty.Read(b[:])
		ch <- result{b: b[0], err: err}
	}()

	select {
	case result := <-ch:
		return result.b, result.err == nil
	case <-time.After(timeout):
		return 0, false
	}
}
