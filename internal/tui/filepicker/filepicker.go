package filepicker

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func New() Model {
	fp := filepicker.New()
	home, err := os.UserHomeDir()
	fp.DirAllowed = true
	fp.AutoHeight = true
	fp.KeyMap.Open = key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "open"))
	fp.KeyMap.Select = key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select"))
	if err != nil {
		// TODO: display error or just use the cwd the program was started from
		slog.Debug("failed to get path to home dir", slog.Any("err", err))
	}
	fp.CurrentDirectory = home
	return Model{
		Done: false,
		fp:   fp,
		KeyMap: DefaultKeyMap(),
		help: help.New(),
	}
}

type Model struct {
	Done     bool
	Selected []string
	fp       filepicker.Model
	KeyMap KeyMap
	help help.Model
}

func (m Model) Init() tea.Cmd {
	return m.fp.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Confirm):
			m.Done = true
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		}
	}
	newfp, cmd := m.fp.Update(msg)
	m.fp = newfp

	// check if a file was selected
	if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
		slog.Debug("file seleected", slog.String("path", path))
		m.Selected = append(m.Selected, path)
	}
	// check if a file was selected that the fileters disable
	// if didSelect, path := m.fp.DidSelectDisabledFile(msg); didSelect {
	// 	//TODO: implement display of error message
	// }
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "File Selection > %s\n", m.fp.CurrentDirectory)
	b.WriteString(m.fp.View())

	if len(m.Selected) != 0 {
		b.WriteString("\n\n")
		for _, path := range m.Selected {
			fmt.Fprintf(&b, "-> %s\n", path)
		}
	}

	b.WriteString("\n")
	b.WriteString(m.help.View(m.KeyMap))
	b.WriteString("\n")

	return b.String()
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		// ctrl+enter is not yet supported by bubbletea and is somewhat problematic in terminals
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm selection")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+q"), key.WithHelp("q", "quit")),
	}
}

type KeyMap struct {
	Confirm key.Binding
	Quit    key.Binding
}

// keybindinds to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Quit}
}

// keybinds to be shown in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Quit},
	}
}
