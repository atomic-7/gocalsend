package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/tui/filepicker"
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
	config       *config.Config
	node         *data.PeerInfo
	upl          *uploader.Uploader
	Context      context.Context
}

type AddSessionManager *server.SessionManager

const (
	peerScreen = iota
	acceptScreen
	fileSelectScreen
	settingsScreen
)

func NewModel(node *data.PeerInfo, appconfig *config.Config) Model {
	return Model{
		screen:     fileSelectScreen,
		prevScreen: peerScreen,
		peerModel:  peers.NewPSModel(),
		filepicker: filepicker.New(),
		config:     appconfig,
		upl:        uploader.CreateUploader(node),
		node:       node,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *sessions.SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.prevScreen = m.screen
		m.screen = acceptScreen
	case AddSessionManager:
		// The session manager needs the reference to the tea program for the hooks
		// This means the session manager cannot be passed at initial creation of the model, because the model is needed to create the program
		m.sessionModel = sessions.NewSessionHandler(msg)
		m.transfers = transfers.New(msg)
	}
	var cmd tea.Cmd
	switch m.screen {
	case peerScreen:
		m.peerModel, cmd = m.peerModel.Update(msg)
		if m.peerModel.Done {
			slog.Debug("peer selected", slog.String("peer", m.peerModel.GetPeer().Alias))
			slog.Debug("uploading files", slog.String("file", m.filepicker.Selected[0]))
			// send file, display ongoing transfers
			m.upl.UploadFiles(m.peerModel.GetPeer(), m.filepicker.Selected)
		}
	case acceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		if m.sessionModel.ShouldClose() {
			slog.Debug("session handler screen should close")
			m.screen = m.prevScreen
		}
	case fileSelectScreen:
		m.filepicker, cmd = m.filepicker.Update(msg)
		if m.filepicker.Done {
			m.screen = peerScreen
		}
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
