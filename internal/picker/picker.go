package picker

import (
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/laeioun/cue/internal/completion"
)

const (
	maxVisible = 5
	nameWidth  = 18
)

type Options struct {
	Query   string
	VimMode bool
}

type Action int

const (
	ActionCancel Action = iota
	ActionSelect
	ActionBackspace
	ActionDelete
	ActionInsert
)

type Result struct {
	Action   Action
	Selected string
	Text     string
}

func Run(completions []completion.Completion, opts Options) (Result, error) {
	if len(completions) == 0 {
		return Result{}, nil
	}

	var in, out *os.File
	if runtime.GOOS == "windows" {
		var err error
		in, err = os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			return Result{}, nil
		}
		out, err = os.OpenFile("CONOUT$", os.O_RDWR, 0)
		if err != nil {
			in.Close()
			return Result{}, nil
		}
		defer in.Close()
		defer out.Close()
	} else {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return Result{}, nil
		}
		defer tty.Close()
		in = tty
		out = tty
	}

	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return Result{}, err
	}
	defer term.Restore(int(in.Fd()), oldState)

	p := inlinePicker{
		out:          out,
		all:          completions,
		initialQuery: opts.Query,
		query:        opts.Query,
		vimMode:      opts.VimMode,
		mode:         insertMode,
	}
	if opts.VimMode {
		p.mode = normalMode
	}
	p.refilter()
	if err := p.render(); err != nil {
		return Result{}, err
	}
	defer p.clear()

	for {
		event, err := readKey(in)
		if err != nil {
			return Result{}, err
		}
		switch event.key {
		case keyAccept:
			if len(p.completions) == 0 {
				continue
			}
			return Result{Action: ActionSelect, Selected: p.completions[p.selected].Name}, nil
		case keyCancel:
			if p.vimMode && p.mode == insertMode {
				p.mode = normalMode
				if err := p.render(); err != nil {
					return Result{}, err
				}
				continue
			}
			return Result{Action: ActionCancel}, nil
		case keyBackspace:
			if p.canEditQuery() {
				p.query = p.query[:len(p.query)-1]
				p.refilter()
				if err := p.render(); err != nil {
					return Result{}, err
				}
				continue
			}
			return Result{Action: ActionBackspace}, nil
		case keyDelete:
			return Result{Action: ActionDelete}, nil
		case keyUp:
			p.moveUp()
			if err := p.render(); err != nil {
				return Result{}, err
			}
		case keyDown:
			p.moveDown()
			if err := p.render(); err != nil {
				return Result{}, err
			}
		case keyText:
			if p.vimMode && p.mode == normalMode {
				if event.value == 'i' {
					p.mode = insertMode
				} else if event.value == 'j' || event.value == 'l' {
					p.moveDown()
				} else if event.value == 'k' || event.value == 'h' {
					p.moveUp()
				}
				if err := p.render(); err != nil {
					return Result{}, err
				}
				continue
			}
			nextQuery := p.query + string(event.value)
			nextCompletions := completion.Filter(p.all, nextQuery)
			if len(nextCompletions) == 0 {
				return Result{
					Action: ActionInsert,
					Text:   strings.TrimPrefix(nextQuery, p.initialQuery),
				}, nil
			}
			p.query = nextQuery
			p.completions = nextCompletions
			if p.selected >= len(p.completions) {
				p.selected = len(p.completions) - 1
			}
			p.ensureVisible()
			if err := p.render(); err != nil {
				return Result{}, err
			}
		}
	}
}

type inlinePicker struct {
	out          *os.File
	all          []completion.Completion
	completions  []completion.Completion
	initialQuery string
	query        string
	vimMode      bool
	mode         pickerMode
	selected     int
	offset       int
}

func (p *inlinePicker) render() error {
	p.ensureVisible()
	width, _, err := term.GetSize(int(p.out.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}

	var b strings.Builder
	b.WriteString("\x1b[s\x1b[?25l\x1b[1B\r")
	b.WriteString("\x1b[2K")
	b.WriteString(fitLine(p.prompt(), width))
	b.WriteString("\x1b[1B\r")
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
		} else if i == 0 && len(p.completions) == 0 {
			b.WriteString("  no matches")
		}
		if i < maxVisible-1 {
			b.WriteString("\x1b[1B\r")
		}
	}
	b.WriteString("\x1b[u\x1b[?25h")

	_, err = p.out.WriteString(b.String())
	return err
}

func (p *inlinePicker) clear() {
	var b strings.Builder
	b.WriteString("\x1b[s\x1b[?25l\x1b[1B\r")
	for i := 0; i < maxVisible+1; i++ {
		b.WriteString("\x1b[2K")
		if i < maxVisible {
			b.WriteString("\x1b[1B\r")
		}
	}
	b.WriteString("\x1b[u\x1b[?25h")
	_, _ = p.out.WriteString(b.String())
}

func (p *inlinePicker) ensureVisible() {
	if len(p.completions) == 0 {
		p.selected = 0
		p.offset = 0
		return
	}
	if p.selected < p.offset {
		p.offset = p.selected
	}
	if p.selected >= p.offset+maxVisible {
		p.offset = p.selected - maxVisible + 1
	}
}

func (p *inlinePicker) refilter() {
	p.completions = completion.Filter(p.all, p.query)
	if len(p.completions) == 0 {
		p.selected = 0
		p.offset = 0
		return
	}
	if p.selected >= len(p.completions) {
		p.selected = len(p.completions) - 1
	}
	p.ensureVisible()
}

func (p *inlinePicker) canEditQuery() bool {
	return p.mode == insertMode && len(p.query) > len(p.initialQuery)
}

func (p *inlinePicker) moveUp() {
	if len(p.completions) == 0 {
		return
	}
	if p.selected == 0 {
		p.selected = len(p.completions) - 1
	} else {
		p.selected--
	}
}

func (p *inlinePicker) moveDown() {
	if len(p.completions) == 0 {
		return
	}
	p.selected = (p.selected + 1) % len(p.completions)
}

func (p *inlinePicker) prompt() string {
	mode := "insert"
	if p.mode == normalMode {
		mode = "normal"
	}
	if p.vimMode {
		return "  fzf (" + mode + ") > " + p.query
	}
	return "  fzf > " + p.query
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

type key int

const (
	keyUnknown key = iota
	keyAccept
	keyCancel
	keyUp
	keyDown
	keyBackspace
	keyDelete
	keyText
)

type keyEvent struct {
	key   key
	value byte
}

type pickerMode int

const (
	insertMode pickerMode = iota
	normalMode
)

func readKey(tty *os.File) (keyEvent, error) {
	var b [1]byte
	if _, err := tty.Read(b[:]); err != nil {
		return keyEvent{key: keyUnknown}, err
	}

	switch b[0] {
	case '\r', '\n', '\t':
		return keyEvent{key: keyAccept}, nil
	case 0x03:
		return keyEvent{key: keyCancel}, nil
	case 0x08, 0x7f:
		return keyEvent{key: keyBackspace}, nil
	case 0x04:
		return keyEvent{key: keyDelete}, nil
	case 0x10:
		return keyEvent{key: keyUp}, nil
	case 0x0e:
		return keyEvent{key: keyDown}, nil
	case 0x1b:
		seq := readEscapeSequence(tty)
		if len(seq) >= 2 && seq[0] == '[' {
			switch seq[1] {
			case 'A':
				return keyEvent{key: keyUp}, nil
			case 'B':
				return keyEvent{key: keyDown}, nil
			case 'Z':
				return keyEvent{key: keyUp}, nil
			case '3':
				if len(seq) >= 3 && seq[2] == '~' {
					return keyEvent{key: keyDelete}, nil
				}
			}
		}
		return keyEvent{key: keyCancel}, nil
	default:
		if b[0] >= 0x20 && b[0] < 0x7f {
			return keyEvent{key: keyText, value: b[0]}, nil
		}
		return keyEvent{key: keyUnknown}, nil
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
		if len(seq) >= 2 && seq[0] == '[' && isEscapeTerminator(seq[len(seq)-1]) {
			break
		}
	}
	return seq
}

func isEscapeTerminator(b byte) bool {
	return b >= '@' && b <= '~'
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
