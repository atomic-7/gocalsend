package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
)

// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: listen for keypresses (ctrlq, q, space, enter, j/k, up/down
// TODO: display the keybinds at the bottom
// TODO: figure out if peermap should be an interface
type Model struct {
	peers   []*data.PeerInfo
	cursor  int
	config  *config.Config
	Context context.Context
	CancelF context.CancelFunc
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

func NewModel(appconfig *config.Config) Model {
	return Model{
		peers:  make([]*data.PeerInfo, 0, 10),
		cursor: 0,
		config: appconfig,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			// if _, ok := m.selected[m.cursor]; ok {
			// 	delete(m.selected, m.cursor)
			// } else {
			// 	m.selected[m.cursor] = struct{}{}
			// }
			// nop
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
	case AddPeerMsg:
		m.addPeer(msg)
		slog.Debug("received peermessage", slog.String("peer", msg.Alias))
	case DelPeerMsg:
		m.delPeer(msg)
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

	b.WriteString("\nPress q or Ctrl+C to quit.\n")

	return b.String()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func NewPeerMap(prog *tea.Program) *PeerMap {
	return &PeerMap{
		peers: make(map[string]*data.PeerInfo),
		program: prog,
	}
}

type PeerMap struct {
	peers   map[string]*data.PeerInfo
	lock    sync.Mutex
	program *tea.Program
}

func (pm *PeerMap) Add(peer *data.PeerInfo) bool {
	slog.Debug("adding to peertracker", slog.String("peer", peer.Alias))
	pm.lock.Lock()
	_, present := pm.peers[peer.Fingerprint]
	pm.peers[peer.Fingerprint] = peer
	pm.lock.Unlock()
	if !present {
		pm.program.Send(AddPeerMsg(peer))
	}
	return !present
}
func (pm *PeerMap) Del(peer *data.PeerInfo) {

	_, present := pm.peers[peer.Fingerprint]
	if !present {
		pm.program.Send(AddPeerMsg(peer))
	}
	pm.lock.Lock()
	delete(pm.peers, peer.Fingerprint)
	pm.lock.Unlock()
}
func (pm *PeerMap) Has(fingerprint string) bool {
	pm.lock.Lock()
	_, ok := pm.peers[fingerprint]
	pm.lock.Unlock()
	return ok
}
