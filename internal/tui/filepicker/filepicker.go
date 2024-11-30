package filepicker

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

func New() Model {
	fp := filepicker.New()
	home, err := os.UserHomeDir()
	fp.AllowedTypes = []string{".txt"}
	fp.DirAllowed = true
	if err != nil {
		// TODO: display error or just use the cwd the program was started from
		slog.Debug("failed to get path to home dir", slog.Any("err", err))
	}
	fp.CurrentDirectory = home
	return Model{
		fp: fp,
	}
}

type Model struct {
	Selected []string
	fp       filepicker.Model
}

func (m Model) Init() tea.Cmd {
	return m.fp.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			slog.Debug("key press", slog.String("key", msg.String()))
		}

	}
	newfp, cmd := m.fp.Update(msg)
	m.fp = newfp
	slog.Debug("fp", slog.String("cwd", m.fp.CurrentDirectory), slog.String("path", m.fp.Path))

	// check if a file was selected
	if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
		m.Selected = append(m.Selected, path)
	}
	// check if a file was selected that the fileters disable
	// if didSelect, path := m.fp.DidSelectDisabledFile(msg); didSelect {
	// 	//TODO: implement display of error message
	// }
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "File Selection > %s\n", m.fp.CurrentDirectory)
	b.WriteString(m.fp.View())

	if len(m.Selected) != 0 {
		b.WriteString("\n\n")
		for _, path := range m.Selected {
			fmt.Fprintf(&b, "-> %s\n", path)
		}
	}

	return b.String()
}
