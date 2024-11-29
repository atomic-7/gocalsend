package tui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/tui/screens"
)

// TODO: when a new client registers, send a message to the update function
// TODO: allow to select clients from a list
// TODO: listen for keypresses (ctrlq, q, space, enter, j/k, up/down
// TODO: display the keybinds at the bottom
// TODO: figure out if peermap should be an interface
type Model struct {
	peerModel    screens.PSModel
	sessionModel screens.SOModel
	screen       uint
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
		peerModel:    screens.NewPSModel(),
		screen:       peerScreen,
		config:       appconfig,
	}
}


func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *screens.SessionOffer:
		slog.Debug("incoming session offer", slog.String("src", "main update"))
		m.screen = acceptScreen
	case AddSessionManager:
		// The session manager needs the reference to the tea program for the hooks
		// This means the session manager cannot be passed at initial creation of the model, because the model is needed to create the program
		m.sessionModel = screens.NewSessionHandler(msg)
	}
	slog.Debug("main update", slog.Any("msg", msg))
	switch m.screen {
	case peerScreen:
		res, cmd := m.peerModel.Update(msg)
		m.peerModel = res.(screens.PSModel)
		return m,cmd
	case acceptScreen:
		// TODO: use batch to create a timer that sends false on the response channel
		res,cmd := m.sessionModel.Update(msg)
		m.sessionModel = res.(screens.SOModel)
		
		if m.sessionModel.ShouldClose() {
			slog.Debug("session handler screen should close")
			m.screen = peerScreen
			// return m.Update(nil)
		}
		return m,cmd
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case peerScreen:
		return m.peerModel.View()
	case acceptScreen:
		return m.sessionModel.View()
	}
	return "wth no scren?"
}

func (m Model) Init() tea.Cmd {
	return tea.SetWindowTitle("gocalsend-tui")
}
