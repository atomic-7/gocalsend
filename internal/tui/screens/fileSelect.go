package screens

import (
	tea "github.com/charmbracelet/bubbletea"
)

type FSModel struct {
	path string
}

func initFileSelect(path string) FSModel {
	return FSModel{
		path: path,
	}
}

func (fsm FSModel) Init() tea.Cmd {
	return nil
}

func (fsm FSModel) Update(msg tea.Msg) (FSModel, tea.Cmd) {
	return fsm, nil
}

func (fsm FSModel) View() string {
	return "selecting files!"
}
