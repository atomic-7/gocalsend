package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/tui/filepicker"
	"github.com/atomic-7/gocalsend/internal/tui/hooks"
	"github.com/atomic-7/gocalsend/internal/tui/peers"
	"github.com/atomic-7/gocalsend/internal/tui/sessions"
	"github.com/atomic-7/gocalsend/internal/tui/transfers"
	"github.com/atomic-7/gocalsend/internal/uploader"
)

// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: figure out if peermap should be an interface
type Model struct {
	screen       uint
	prevScreen   uint
	peerModel    peers.Model
	sessionModel sessions.Model
	filepicker   filepicker.Model
	transfers    transfers.Model
	config       *config.Config
	node         *data.PeerInfo
	Uploader     *uploader.Uploader
	Context      context.Context
}

type AddSessionManager *server.SessionManager

const (
	peerScreen = iota
	acceptScreen
	fileSelectScreen
	settingsScreen
	transfersScreen
)

func NewModel(node *data.PeerInfo, appconfig *config.Config) Model {
	return Model{
		screen:     fileSelectScreen,
		prevScreen: peerScreen,
		peerModel:  peers.NewPSModel(),
		filepicker: filepicker.New(),
		config:     appconfig,
		node:       node,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *hooks.SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.prevScreen = m.screen
		m.screen = acceptScreen
	case AddSessionManager:
		// The session manager needs the reference to the tea program for the hooks
		// This means the session manager cannot be passed at initial creation of the model, because the model is needed to create the program
		m.sessionModel = sessions.NewSessionHandler(msg)
		m.transfers = transfers.New(msg)
		// Did not call init!!!!
	case peers.AddPeerMsg:
		m.peerModel.AddPeer(msg)
		slog.Debug("received peermessage", slog.String("peer", msg.Alias))
	case peers.DelPeerMsg:
		m.peerModel.DelPeer(msg)
	}
	var cmd tea.Cmd
	switch m.screen {
	case acceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		if m.sessionModel.ShouldClose() {
			slog.Debug("session handler screen should close")
			m.screen = m.prevScreen
		}
	case peerScreen:
		m.peerModel, cmd = m.peerModel.Update(msg)
		if m.peerModel.Done {
			slog.Debug("peer selected", slog.String("peer", m.peerModel.GetPeer().Alias))
			slog.Debug("uploading files", slog.String("file", m.filepicker.Selected[0]))
			// send file, display ongoing transfers
			// maybe do this in a cmd?
			go m.Uploader.UploadFiles(m.peerModel.GetPeer(), m.filepicker.Selected)
			m.screen = transfersScreen
		}
	case fileSelectScreen:
		m.filepicker, cmd = m.filepicker.Update(msg)
		if m.filepicker.Done {
			m.screen = peerScreen
		}
	case transfersScreen:
		m.transfers, cmd = m.transfers.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	switch m.screen {
	case peerScreen:
		return m.peerModel.View()
	case acceptScreen:
		return m.sessionModel.View()
	case fileSelectScreen:
		return m.filepicker.View()
	case transfersScreen:
		return m.transfers.View()
	}
	return "wth no scren?"
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.filepicker.Init(),
		m.sessionModel.Init(),
		m.peerModel.Init(),
		tea.SetWindowTitle("gocalsend-tui"),
	)
}
