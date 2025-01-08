package sessions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/tui/hooks"
	"github.com/atomic-7/gocalsend/internal/sessions"
)

type Model struct {
		cursor int
		KeyMap KeyMap
		help help.Model
		SessionManager *sessions.SessionManager
		sessionOffers  []*hooks.SessionOffer
}

func NewSessionHandler(sessionManager *sessions.SessionManager) Model {
	return Model{
		cursor: 0,
		KeyMap: DefaultKeyMap(),
		help: help.New(),
		SessionManager: sessionManager,
		sessionOffers:  make([]*hooks.SessionOffer, 0, 10),
	}	
}

func (m *Model) cursorUp() {
	if m.cursor > 0 {
		m.cursor -= 1
	}
}

func (m *Model) cursorDown() {
	if m.cursor < len(m.sessionOffers)-1 {
		m.cursor += 1
	}
}


func (m *Model) acceptSession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].Res <- true
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
	if len(m.sessionOffers) == 0 {
		m.cursor = 0
	} else {
		m.cursor -= 1
	}
}
func (m *Model) denySession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].Res <- false
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
	if len(m.sessionOffers) == 0 {
		m.cursor = 0
	} else {
		m.cursor -= 1
	}
}
func (m *Model) denyAll() {
	for _, offer := range m.sessionOffers {
		offer.Res <- false
	}
	m.sessionOffers = make([]*hooks.SessionOffer, 0, 10)
}
func (m *Model) ShouldClose() bool {
	return len(m.sessionOffers) == 0
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *hooks.SessionOffer:
		m.sessionOffers = append(m.sessionOffers, msg)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Up):
			m.cursorUp()
		case key.Matches(msg, m.KeyMap.Down):
			m.cursorDown()
		case key.Matches(msg, m.KeyMap.Accept):
			m.acceptSession()
		case key.Matches(msg, m.KeyMap.Deny):
			m.denySession()
		case key.Matches(msg, m.KeyMap.DenyAll):
			m.denyAll()
		case key.Matches(msg, m.KeyMap.Quit):
			m.denyAll()
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("Incoming transfers\n")
	for i, offer := range m.sessionOffers {
		indicator := " "
		if m.cursor == i {
			indicator = ">"
		}
		fmt.Fprintf(&b, "%s | %s\n", indicator, offer.Sess.SessionID)
		for _, file := range offer.Sess.Files {
			fmt.Fprintf(&b, "  # %s - %d \n", file.FileName, file.Size)
		}
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.KeyMap))
	b.WriteString("\n")

	return b.String()
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		Down:    key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Accept: key.NewBinding(key.WithKeys("space", "enter","y"), key.WithHelp("enter", "accept")),
		Deny: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "deny")),
		DenyAll: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctr+c", "deny all")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+q"), key.WithHelp("q", "quit")),
	}
}

type KeyMap struct {
	Up key.Binding
	Down key.Binding
	Accept key.Binding
	Deny key.Binding
	DenyAll key.Binding
	Quit key.Binding
}
// keybindinds to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Accept, k.Deny, k.Quit}
}

// keybinds to be shown in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Accept, k.Deny, k.DenyAll, k.Quit},
	}
}
