package tui

import (
	"log/slog"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/server"
)


// this could be rewritten to where the model is implementing the PeerMap interface
type UIHooks struct {
	program *tea.Program
}

func NewHooks(p *tea.Program) *UIHooks {
	return &UIHooks{
		program: p,
	}
}

func (h *UIHooks) OfferSession(sess *server.Session, res responseChannel) {
	h.program.Send(&SessionOffer{sess: sess, res: res})
}

func (h *UIHooks) SessionFinished() {
	h.program.Send(SessionFinished(true))
}

func NewPeerMap(prog *tea.Program) *PeerMap {
	return &PeerMap{
		peers:   make(map[string]*data.PeerInfo),
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
