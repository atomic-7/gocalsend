package hooks

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/sessions"
	"github.com/atomic-7/gocalsend/internal/tui/peers"
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
type FileFinished bool
type SessionCreated bool
type SessionFinished bool
type SessionCancelled bool
type ResponseChannel = chan bool
type SessionOffer struct {
	Sess *sessions.Session
	Res  ResponseChannel
}

func (h *UIHooks) OfferSession(sess *sessions.Session, res ResponseChannel) {
	h.program.Send(&SessionOffer{Sess: sess, Res: res})
}

func (h *UIHooks) FileFinished() {
	h.program.Send(FileFinished(true))
}

func (h *UIHooks) SessionCreated() {
	h.program.Send(SessionCreated(true))
}

func (h *UIHooks) SessionFinished() {
	slog.Debug("sending session finished msg to tui")
	h.program.Send(SessionFinished(true))
}

func (h *UIHooks) SessionCancelled() {
	h.program.Send(SessionCancelled(true))
}

func NewPeerMap(prog *tea.Program) *PeerMap {
	return &PeerMap{
		peers:   *data.NewPeerMap(),
		program: prog,
	}
}

type PeerMap struct {
	peers   data.PeerMap
	program *tea.Program
}

func (pm *PeerMap) Add(peer *data.PeerInfo) bool {
	add := pm.Add(peer)
	if add {
		pm.program.Send(peers.AddPeerMsg(peer))
	}
	return add

}
func (pm *PeerMap) Del(peer *data.PeerInfo) {
	if pm.Has(peer.Fingerprint) {
		pm.program.Send(peers.DelPeerMsg(peer.Fingerprint))
	}
	pm.peers.Del(peer)
}
func (pm *PeerMap) Has(fingerprint string) bool {
	return pm.peers.Has(fingerprint)
}
func (pm *PeerMap) Get(fingerprint string) (*data.PeerInfo, bool) {
	return pm.peers.Get(fingerprint)
}
