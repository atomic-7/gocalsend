package screens

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
)

type KeyMap struct {
	Up key.Binding
	Down key.Binding
	Confirm key.Binding
	Quit key.Binding
}

type PSModel struct {
	cursor int
	peers  []*data.PeerInfo
	config *config.Config
	KeyMap KeyMap
}
type AddPeerMsg *data.PeerInfo
type DelPeerMsg = string

func (m *PSModel) addPeer(peer *data.PeerInfo) {
	m.peers = append(m.peers, peer)
}

func (m *PSModel) delPeer(fingerprint string) {
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

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		Down: key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Confirm: key.NewBinding(key.WithKeys("space", "enter"), key.WithHelp("space", "confirm")),
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+q"), key.WithHelp("q", "quit")),
	}
}

func NewPSModel() PSModel {
	return PSModel{
		peers:  make([]*data.PeerInfo, 0, 10),
		cursor: 0,
		KeyMap: DefaultKeyMap(),
	}
}

func (m PSModel) Init() tea.Cmd {
	return nil
}

func (m PSModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *SessionOffer:
		slog.Debug("session offer got trapped in peer select")
	case AddPeerMsg:
		m.addPeer(msg)
		slog.Debug("received peermessage", slog.String("peer", msg.Alias))
	case DelPeerMsg:
		m.delPeer(msg)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Up):
			if m.cursor > 0 {
				m.cursor -= 1
			}
		case key.Matches(msg, m.KeyMap.Down):
			if m.cursor < len(m.peers)-1 {
					m.cursor += 1
			}
			
		case key.Matches(msg, m.KeyMap.Confirm):
			slog.Info("entry selected", slog.String("screen", "peerScreen"), slog.String("peer", m.peers[m.cursor].Alias))
			return m, nil
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m PSModel) View() string {
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
