package transfers

import (
	"fmt"
	"strings"
	"log/slog"

	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/tui/hooks"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)


type Model struct {
	sman   *server.SessionManager
	help   help.Model
	KeyMap KeyMap
}

func New(sman *server.SessionManager) Model {
	return Model{
		sman: sman,
		help: help.New(),
		KeyMap: DefaultKeyMap(),
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
		// TODO: cancel all ongoing sessions
			return m, tea.Quit
		}
	case hooks.FileFinished:
		slog.Debug("received file finished msg", slog.String("src", "transfers"))
	case hooks.SessionFinished:
		slog.Debug("received session finished msg", slog.String("src", "transfers"))
	case hooks.SessionCancelled:
		slog.Debug("received session cancelled msg", slog.String("src", "transfers"))

	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("Transfers\n\n")
	for k, s := range m.sman.Sessions {
		fmt.Fprintf(&b, " %s | %s (%d / %d)\n", k, s.SessionID)
	}
	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.KeyMap))
	return b.String()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+q"), key.WithHelp("q", "quit")),
	}
}

type KeyMap struct {
	Quit key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit},
	}
}
