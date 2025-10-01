package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	sessionmanager "github.com/atomic-7/gocalsend/internal/sessions"
	"github.com/atomic-7/gocalsend/internal/tui/filepicker"
	"github.com/atomic-7/gocalsend/internal/tui/hooks"
	"github.com/atomic-7/gocalsend/internal/tui/peers"
	screens "github.com/atomic-7/gocalsend/internal/tui/screens"
	"github.com/atomic-7/gocalsend/internal/tui/sessions"
	"github.com/atomic-7/gocalsend/internal/tui/transfers"
	"github.com/atomic-7/gocalsend/internal/uploader"
)

type Model struct {
	screen       screens.Screen
	prevScreen   screens.Screen
	peerModel    peers.Model
	sessionModel sessions.Model
	filepicker   filepicker.Model
	transfers    transfers.Model
	config       *config.Config
	node         *data.PeerInfo
	Uploader     *uploader.Uploader
	Context      context.Context
}

type AddSessionManager *sessionmanager.SessionManager

func NewModel(ctx context.Context, node *data.PeerInfo, appconfig *config.Config) Model {
	return Model{
		screen:     screens.FileSelectScreen,
		prevScreen: screens.PeerScreen,
		peerModel:  peers.NewPSModel(),
		filepicker: filepicker.New(),
		config:     appconfig,
		node:       node,
		Context:    ctx,
	}
}

func (m *Model) SetupSessionManagers(sman *sessionmanager.SessionManager) {
	m.sessionModel = sessions.NewSessionHandler(sman)
	m.transfers = transfers.New(sman)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *hooks.SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.prevScreen = m.screen
		m.screen = screens.AcceptScreen
	case *hooks.SessionCancelled:
		slog.Debug("session cancelled", slog.String("src", "main update"))
		m.screen = screens.FileSelectScreen
	case peers.AddPeerMsg:
		m.peerModel.AddPeer(msg)
		slog.Debug("received peermessage", slog.String("peer", msg.Alias))
	case peers.DelPeerMsg:
		m.peerModel.DelPeer(msg)
	case screens.Screen:
		m.prevScreen = m.screen
		m.screen = msg
		if m.screen == screens.FileSelectScreen {
			m.filepicker.Reset()
			m.peerModel.Reset()
		}
		slog.Debug("switching screen", slog.Any("screen", m.screen))
		return m, nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screens.AcceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		if m.sessionModel.ShouldClose() {
			slog.Debug("session handler screen should close")
			m.screen = m.prevScreen
		}
	case screens.PeerScreen:
		m.peerModel, cmd = m.peerModel.Update(msg)
		if m.peerModel.ShouldGoBack {
			m.filepicker.Done = false
			m.peerModel.ShouldGoBack = false
			m.screen = screens.FileSelectScreen
			return m, nil
		}
		if m.peerModel.Done {
			slog.Debug("peer selected", slog.String("peer", m.peerModel.GetPeer().Alias))
			slog.Debug("uploading files", slog.String("file", m.filepicker.Selected[0]))
			// send file, display ongoing transfers
			m.screen = screens.TransfersScreen
			cmd = tea.Batch(cmd, func() tea.Msg {
				err := m.Uploader.UploadFiles(m.peerModel.GetPeer(), m.filepicker.Selected)
				if err != nil {
					if err.Error() == "Rejected" {
						slog.Debug("upload cancelled by peer")
						return hooks.SessionCancelled(true)
					} else {
						slog.Error("upload failed", slog.Any("error", err))
					}
					return nil
				}
				slog.Debug("uploader finished")
				return nil
			}, func() tea.Msg {
				// TODO: see if this is still needed
				return hooks.SessionCreated(true)
			})
		}
	case screens.FileSelectScreen:
		m.filepicker, cmd = m.filepicker.Update(msg)
		if m.filepicker.Done {
			m.peerModel.Files = &m.filepicker.Selected
			m.screen = screens.PeerScreen
		}
	case screens.TransfersScreen:
		m.transfers, cmd = m.transfers.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	switch m.screen {
	case screens.PeerScreen:
		return m.peerModel.View()
	case screens.AcceptScreen:
		return m.sessionModel.View()
	case screens.FileSelectScreen:
		return m.filepicker.View()
	case screens.TransfersScreen:
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
