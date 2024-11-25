package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/data"
)
// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: listen for keypresses (ctrlq, q, space, enter, j/k, up/down
// TODO: display the keybinds at the bottom
// TODO: figure out if peermap should be an interface
type model struct {
	KnownPeers   []*data.PeerInfo
	SelectedPeer int
	Mode         string
}

func (m *model) addPeer(peer *data.PeerInfo) {
	m.KnownPeers = append(m.KnownPeers, peer)
}

func (m *model) delPeer(fingerprint string) {
	elem := -1
	for idx, peer := range m.KnownPeers {
		if peer.Fingerprint == fingerprint {
			elem = idx
			break
		}
	}
	if elem != -1 {
		m.KnownPeers[elem] = nil // set to nil so the reference can be garbage collected
		m.KnownPeers = append(m.KnownPeers[:elem], m.KnownPeers[elem+1:]...)
	}
}

func (m* model) cursorUp() {
	
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m model) View() string {
	return "hello"
}

func (m model) Init() tea.Cmd {
	return nil
}
