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
	ActionDeleteWordBefore
	ActionDeleteWordAt
	ActionMoveLeft
	ActionMoveRight
	ActionMoveWordLeft
	ActionMoveWordRight
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
		case keyDeleteWordBefore:
			return Result{Action: ActionDeleteWordBefore}, nil
		case keyDeleteWordAt:
			return Result{Action: ActionDeleteWordAt}, nil
		case keyLeft:
			return Result{Action: ActionMoveLeft}, nil
		case keyRight:
			return Result{Action: ActionMoveRight}, nil
		case keyWordLeft:
			return Result{Action: ActionMoveWordLeft}, nil
		case keyWordRight:
			return Result{Action: ActionMoveWordRight}, nil
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
	keyDeleteWordBefore
	keyDeleteWordAt
	keyLeft
	keyRight
	keyWordLeft
	keyWordRight
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
	case 0x17:
		return keyEvent{key: keyDeleteWordBefore}, nil
	case 0x04:
		return keyEvent{key: keyDelete}, nil
	case 0x1b:
		seq := readEscapeSequence(tty)
		if len(seq) == 0 {
			return keyEvent{key: keyCancel}, nil
		}
		if event, ok := parseEscapeSequence(seq); ok {
			return event, nil
		}
		return keyEvent{key: keyUnknown}, nil
	default:
		if b[0] >= 0x20 && b[0] < 0x7f {
			return keyEvent{key: keyText, value: b[0]}, nil
		}
		return keyEvent{key: keyUnknown}, nil
	}
}

func parseEscapeSequence(seq []byte) (keyEvent, bool) {
	if len(seq) == 0 {
		return keyEvent{}, false
	}

	switch seq[0] {
	case '[':
		csi := string(seq)
		switch seq[len(seq)-1] {
		case 'A':
			return keyEvent{key: keyUp}, true
		case 'B':
			return keyEvent{key: keyDown}, true
		case 'C':
			if isModifiedArrow(csi, '5') {
				return keyEvent{key: keyWordRight}, true
			}
			return keyEvent{key: keyRight}, true
		case 'D':
			if isModifiedArrow(csi, '5') {
				return keyEvent{key: keyWordLeft}, true
			}
			return keyEvent{key: keyLeft}, true
		case 'u':
			return parseCSIU(csi)
		case '~':
			if strings.HasPrefix(csi, "[3;5") || strings.HasPrefix(csi, "[27;5;3") {
				return keyEvent{key: keyDeleteWordAt}, true
			}
			if strings.HasPrefix(csi, "[3") {
				return keyEvent{key: keyDelete}, true
			}
		}
	case 'O':
		if len(seq) < 2 {
			return keyEvent{}, false
		}
		switch seq[1] {
		case 'A':
			return keyEvent{key: keyUp}, true
		case 'B':
			return keyEvent{key: keyDown}, true
		case 'C':
			return keyEvent{key: keyRight}, true
		case 'D':
			return keyEvent{key: keyLeft}, true
		}
	}
	return keyEvent{}, false
}

func isModifiedArrow(csi string, modifier byte) bool {
	if csi == "["+string(modifier)+"C" || csi == "["+string(modifier)+"D" {
		return true
	}
	if len(csi) < 5 || csi[len(csi)-1] < 'A' || csi[len(csi)-1] > 'D' {
		return false
	}
	return strings.Contains(csi, ";"+string(modifier))
}

func parseCSIU(csi string) (keyEvent, bool) {
	switch csi {
	case "[8;5u", "[127;5u":
		return keyEvent{key: keyDeleteWordBefore}, true
	case "[3;5u":
		return keyEvent{key: keyDeleteWordAt}, true
	}
	return keyEvent{}, false
}

func readEscapeSequence(tty *os.File) []byte {
	seq := make([]byte, 0, 8)
	for len(seq) < cap(seq) {
		b, ok := readByteWithTimeout(tty, 50*time.Millisecond)
		if !ok {
			break
		}
		seq = append(seq, b)
		if isCompleteEscapeSequence(seq) {
			break
		}
	}
	return seq
}

func isCompleteEscapeSequence(seq []byte) bool {
	if len(seq) == 0 {
		return false
	}
	switch seq[0] {
	case '[':
		return len(seq) >= 2 && isEscapeTerminator(seq[len(seq)-1])
	case 'O':
		return len(seq) >= 2
	default:
		return true
	}
}

func isEscapeTerminator(b byte) bool {
	return b >= '@' && b <= '~'
}

func readByteWithTimeout(tty *os.File, timeout time.Duration) (byte, bool) {
	if err := tty.SetReadDeadline(time.Now().Add(timeout)); err == nil {
		defer tty.SetReadDeadline(time.Time{})

		var b [1]byte
		_, err := tty.Read(b[:])
		return b[0], err == nil
	}

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
