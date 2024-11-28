package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/tui/screens"
)

// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: listen for keypresses (ctrlq, q, space, enter, j/k, up/down
// TODO: display the keybinds at the bottom
// TODO: figure out if peermap should be an interface
type Model struct {
	peerModel	screens.PSModel
	screen         uint
	peers          []*data.PeerInfo
	cursor         int
	config         *config.Config
	Context        context.Context
	SessionManager *server.SessionManager

	// TODO: remove session offers after they have been handled
	sessionOffers []*SessionOffer
}

const (
	peerScreen = iota
	acceptScreen
	fileSelectScreen
	settingsScreen
)

type responseChannel = chan bool

type SessionOffer struct {
	sess *server.Session
	res  responseChannel
}
type SessionFinished bool

func NewModel(appconfig *config.Config, sessionManager *server.SessionManager) Model {
	return Model{
		peerModel: screens.NewPSModel(sessionManager),
		screen:         peerScreen,
		cursor:         0,
		config:         appconfig,
		SessionManager: sessionManager,
		sessionOffers:  make([]*SessionOffer, 0, 10),
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.screen = acceptScreen
		m.cursor = 0
		m.sessionOffers = append(m.sessionOffers, msg)
	}
	switch m.screen {
	case peerScreen:
		return m.peerModel.Update(msg)
	case acceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		return acceptScreenUpdate(msg, m)
	}
	return m, nil
}

func (m *Model) cursorUp() {
	if m.cursor > 0 {
		m.cursor -= 1
	}
}

func (m *Model) cursorDown() {
	if m.cursor < len(m.peers)-1 {
		m.cursor += 1
	}
}

func (m *Model) acceptSession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].res <- true
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
}
func (m *Model) denySession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].res <- false
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
}
func acceptScreenUpdate(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.denySession()
			if len(m.sessionOffers) == 0 {
				m.cursor = 0
				m.screen = peerScreen
			} else {
				m.cursor -= 1
			}
			return m, nil
		case tea.KeyUp:
			m.cursorUp()
		case tea.KeyDown:
			m.cursorDown()
		case tea.KeyEnter, tea.KeySpace:
			slog.Info("entry selected", slog.String("screen", "acceptScreen"))
			m.acceptSession()
			if len(m.sessionOffers) == 0 {
				m.cursor = 0
				m.screen = peerScreen
			} else {
				m.cursor -= 1
			}
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				m.denySession()
				m.cursor = 0
				m.screen = peerScreen
				return m, nil
			case "j":
				m.cursorDown()
			case "k":
				m.cursorUp()
			}
		}
	}
	return m, nil
}

func renderPeerScreen(m *Model) string {
	var b strings.Builder
	b.WriteString("Peers\n\n")
	slog.Debug("render", slog.Any("peers", m.peers))
	for i, peer := range m.peers {
		indicator := " "
		if m.cursor == i {
			indicator = ">"
		}
		fmt.Fprintf(&b, "%s | %s\n", indicator, peer.Alias)
	}

	b.WriteString("\nPress q or Ctrl+C to quit.\n")

	return b.String()
}

func renderAcceptScreen(m *Model) string {
	var b strings.Builder
	b.WriteString("Incoming transfers\n")
	slog.Debug("render", slog.Any("sessions", m.sessionOffers))
	for i, offer := range m.sessionOffers {
		indicator := " "
		if m.cursor == i {
			indicator = ">"
		}
		fmt.Fprintf(&b, "%s | %s\n", indicator, offer.sess.SessionID)
		for _, file := range offer.sess.Files {
			fmt.Fprintf(&b, "  # %s - %d \n", file.FileName, file.Size)
		}
	}

	b.WriteString("\nPress Enter/Space to accept.\nPress q or Ctrl+C to deny.\n")

	return b.String()
}
func (m Model) View() string {
	switch m.screen {
	case peerScreen:
		return renderPeerScreen(&m)
	case acceptScreen:
		return renderAcceptScreen(&m)
	}
	return "wth no scren?"
}

func (m Model) Init() tea.Cmd {
	return tea.SetWindowTitle("gocalsend-tui")
}
