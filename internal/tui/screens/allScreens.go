package screens

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Screen (uint)

const (
	PeerScreen       = Screen(0)
	AcceptScreen     = Screen(1)
	FileSelectScreen = Screen(2)
	SettingsScreen   = Screen(3)
	TransfersScreen  = Screen(4)
)

func SwitchScreen(screen Screen) tea.Cmd {
	return func() tea.Msg {
		return screen
	}
}

/*
 * The drawback of using commands to switch between screens is that the screens now need to know the control flow
 * Constantly checking for done in the update loop seems awkward, but maybe is the optimal way to constrain the business logic to one file?
 */
