package transfers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/atomic-7/gocalsend/internal/sessions"
	"github.com/atomic-7/gocalsend/internal/tui/hooks"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	sman   *sessions.SessionManager
	help   help.Model
	KeyMap KeyMap
}

func New(sman *sessions.SessionManager) Model {
	return Model{
		sman:   sman,
		help:   help.New(),
		KeyMap: DefaultKeyMap(),
	}
}

func (m *Model) cancelAllSessions() tea.Msg {
	for id, _ := range m.sman.Downloads {
		m.sman.CancelSession(id)
	}
	for id, _ := range m.sman.Uploads {
		m.sman.CancelSession(id)
	}
	slog.Debug("cancelled all sessions", slog.String("src", "transfers"))
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Sequence(m.cancelAllSessions, tea.Quit)
		case key.Matches(msg, m.KeyMap.Cancel):
			return m, m.cancelAllSessions
		}
	case hooks.FileFinished:
		slog.Debug("received file finished msg", slog.String("src", "transfers"))
	case hooks.SessionCreated:
		slog.Debug("received session start msg", slog.String("src", "transfers"))
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
	if len(m.sman.Downloads) != 0 {
		b.WriteString("Downloads\n")
		for _, s := range m.sman.Downloads {
			fmt.Fprintf(&b, " %s | %s (%d / %d)\n", s.Peer.Alias, s.SessionID, s.Remaining, len(s.Files))
		}
		b.WriteString("\n\n")
	}
	if len(m.sman.Uploads) != 0 {
		b.WriteString("Uploads\n")
		for _, s := range m.sman.Uploads {
			fmt.Fprintf(&b, " %s | %s (%d / %d)\n", s.Peer.Alias, s.SessionID, s.Remaining, len(s.Files))
		}
		b.WriteString("\n\n")
	}
	b.WriteString(m.help.View(m.KeyMap))
	return b.String()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+q"), key.WithHelp("q", "quit")),
		Cancel: key.NewBinding(key.WithKeys("c", "esc"), key.WithHelp("esc", "cancel")),
	}
}

type KeyMap struct {
	Quit   key.Binding
	Cancel key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Cancel}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Cancel},
	}
}
