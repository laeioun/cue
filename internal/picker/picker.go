package picker

import (
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/laeioun/cue/internal/completion"
)

type item struct {
	name        string
	description string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.name }

type model struct {
	list     list.Model
	selected string
	quitting bool
}

func Run(completions []completion.Completion) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		tty, err = os.OpenFile("CONOUT$", os.O_RDWR, 0)
		if err != nil {
			return "", err
		}
	}
	defer tty.Close()

	program := tea.NewProgram(
		newModel(completions),
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	result, err := program.Run()
	if m, ok := result.(model); ok && m.selected != "" {
		return m.selected, nil
	}
	return "", err
}

func newModel(completions []completion.Completion) model {
	items := make([]list.Item, 0, len(completions))
	for _, completion := range completions {
		items = append(items, item{
			name:        completion.Name,
			description: completion.Description,
		})
	}

	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "cue completions"
	l.SetShowHelp(true)
	l.SetShowStatusBar(false)

	return model{list: l}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selected = i.name
			}
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}
