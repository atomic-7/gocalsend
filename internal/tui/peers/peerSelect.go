package peers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/tui/screens"
)

type Model struct {
	cursor int
	peers  []*data.PeerInfo
	config *config.Config
	help   help.Model
	KeyMap KeyMap
}
type AddPeerMsg *data.PeerInfo
type DelPeerMsg = string

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

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		Down:    key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Confirm: key.NewBinding(key.WithKeys("space", "enter"), key.WithHelp("space", "confirm")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+q"), key.WithHelp("q", "quit")),
	}
}

func NewPSModel() Model {
	return Model{
		peers:  make([]*data.PeerInfo, 0, 10),
		cursor: 0,
		help: help.New(),
		KeyMap: DefaultKeyMap(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *screens.SessionOffer:
		slog.Debug("session offer got trapped in peer select")
	case AddPeerMsg:
		m.addPeer(msg)
		slog.Debug("received peermessage", slog.String("peer", msg.Alias))
	case DelPeerMsg:
		m.delPeer(msg)
	case tea.WindowSizeMsg:
		// help truncates if the width is not enough
		m.help.Width = msg.Width
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

func (m Model) View() string {
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

	helpView := m.help.View(m.KeyMap)

	b.WriteString("\n\n")
	b.WriteString(helpView)
	b.WriteString("\n")

	return b.String()
}

// Implements key.Map interface
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Confirm key.Binding
	Quit    key.Binding
}

// keybindinds to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Confirm, k.Quit}
}

// keybinds to be shown in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Confirm, k.Quit},
	}
}
