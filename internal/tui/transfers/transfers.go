package transfers

import (
	"github.com/atomic-7/gocalsend/internal/server"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	sman *server.SessionManager
}

func New(sman *server.SessionManager) Model {
	return Model{
		sman: sman,
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return ""
}

func (m Model) Init() tea.Cmd {
	return nil
}
