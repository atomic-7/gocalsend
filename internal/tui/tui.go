package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/tui/filepicker"
	"github.com/atomic-7/gocalsend/internal/tui/sessions"
	"github.com/atomic-7/gocalsend/internal/tui/peers"
)

// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: listen for keypresses (ctrlq, q, space, enter, j/k, up/down
// TODO: display the keybinds at the bottom
// TODO: figure out if peermap should be an interface
type Model struct {
	screen       uint
	peerModel    peers.Model
	sessionModel sessions.Model
	filepicker   filepicker.Model
	config       *config.Config
	Context      context.Context
}

type AddSessionManager *server.SessionManager

const (
	peerScreen = iota
	acceptScreen
	fileSelectScreen
	settingsScreen
)

func NewModel(appconfig *config.Config) Model {
	return Model{
		screen:    fileSelectScreen,
		peerModel: peers.NewPSModel(),
		filepicker: filepicker.New(),
		config:    appconfig,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *sessions.SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.screen = acceptScreen
	case AddSessionManager:
		// The session manager needs the reference to the tea program for the hooks
		// This means the session manager cannot be passed at initial creation of the model, because the model is needed to create the program
		m.sessionModel = sessions.NewSessionHandler(msg)
	}
	slog.Debug("main update", slog.Any("msg", msg))
	var cmd tea.Cmd
	switch m.screen {
	case peerScreen:
		m.peerModel, cmd = m.peerModel.Update(msg)
	case acceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		m.sessionModel, cmd = m.sessionModel.Update(msg)
		if m.sessionModel.ShouldClose() {
			slog.Debug("session handler screen should close")
			m.screen = peerScreen
		}
	case fileSelectScreen:
		m.filepicker, cmd = m.filepicker.Update(msg)
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
