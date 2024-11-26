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
)

// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: listen for keypresses (ctrlq, q, space, enter, j/k, up/down
// TODO: display the keybinds at the bottom
// TODO: figure out if peermap should be an interface
type Model struct {
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

type AddPeerMsg *data.PeerInfo
type DelPeerMsg = string
type SessionOffer struct {
	sess *data.SessionInfo
	res  responseChannel
}
type SessionFinished bool

func (m *Model) addPeer(peer *data.PeerInfo) {
	m.peers = append(m.peers, peer)
}

func (m *Model) delPeer(fingerprint string) {
	elem := -1
	for idx, peer := range m.peers {
		if peer.Fingerprint == fingerprint {
			elem = idx
			break
		}
	}
	if elem != -1 {
		if m.cursor >= elem {
			m.cursor -= 1
		}
		m.peers[elem] = nil // set to nil so the reference can be garbage collected
		m.peers = append(m.peers[:elem], m.peers[elem+1:]...)
	}
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

func NewModel(appconfig *config.Config, sessionManager *server.SessionManager) Model {
	return Model{
		screen:         peerScreen,
		peers:          make([]*data.PeerInfo, 0, 10),
		cursor:         0,
		config:         appconfig,
		SessionManager: sessionManager,
		sessionOffers:  make([]*SessionOffer, 0, 10),
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AddPeerMsg:
		m.addPeer(msg)
		slog.Debug("received peermessage", slog.String("peer", msg.Alias))
	case DelPeerMsg:
		m.delPeer(msg)
	case *SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.screen = acceptScreen
		m.cursor = 0
		m.sessionOffers = append(m.sessionOffers, msg)
	}
	switch m.screen {
	case peerScreen:
		return peerScreenUpdate(msg, m)
	case acceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		return acceptScreenUpdate(msg, m)
	}
	return m, nil
}

func peerScreenUpdate(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyUp:
			m.cursorUp()
		case tea.KeyDown:
			m.cursorDown()
		case tea.KeyEnter, tea.KeySpace:
			slog.Info("entry selected", slog.String("screen", "peerScreen"), slog.String("peer", m.peers[m.cursor].Alias))
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return m, tea.Quit
			case "j":
				m.cursorDown()
			case "k":
				m.cursorUp()
			}
		}
	}
	return m, nil
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
			m.cursor = 0
			m.screen = peerScreen
			return m, nil
		case tea.KeyUp:
			m.cursorUp()
		case tea.KeyDown:
			m.cursorDown()
		case tea.KeyEnter, tea.KeySpace:
			slog.Info("entry selected", slog.String("screen", "acceptScreen"))
			m.acceptSession()
			m.cursor = 0
			m.screen = peerScreen
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
		for fi, tok := range offer.sess.Files {
			fmt.Fprintf(&b, "  # %s - %s\n", fi, tok)
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
