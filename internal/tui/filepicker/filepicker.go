package filepicker

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

func New() Model {
	fp := filepicker.New()
	home, err := os.UserHomeDir()
	fp.AllowedTypes = []string{".txt"}
	fp.DirAllowed = true
	fp.ShowHidden = true
	if err != nil {
		// TODO: display error or just use the cwd the program was started from
		slog.Debug("failed to get path to home dir", slog.Any("err", err))
	}
	fp.CurrentDirectory = home
	dirEntries, err := os.ReadDir(home)
	if err != nil {
		slog.Debug("failed to read directory", slog.Any("err", err))
	}
	sort.Slice(dirEntries, func(i, j int) bool {
		if dirEntries[i].IsDir() == dirEntries[j].IsDir() {
			return dirEntries[i].Name() < dirEntries[j].Name()
		}
		return dirEntries[i].IsDir()
	})
	var sanitizedDirEntries []os.DirEntry
	for _, dirEntry := range dirEntries {
		isHidden, _ := filepicker.IsHidden(dirEntry.Name())
		if isHidden {
			continue
		}
		sanitizedDirEntries = append(sanitizedDirEntries, dirEntry)
	}
	slog.Debug("sorted", slog.Any("entries", dirEntries))
	slog.Debug("sanitized", slog.Any("entries", sanitizedDirEntries))
	return Model{
		dbgfiles: dirEntries,
		sanitized: sanitizedDirEntries,
		fp: fp,
	}
}

type Model struct {
	Selected []string
	dbgfiles []fs.DirEntry
	sanitized []fs.DirEntry
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
	// b.WriteString("Sorted listing")
	// for _, entry := range m.dbgfiles {
	// 	fmt.Fprintf(&b, " + %s\n", entry.Name())
	// }
	// b.WriteString("sanitized listing\n")
	// for _, entry := range m.sanitized {
	// 	fmt.Fprintf(&b, " * %s\n", entry.Name())
	// }

	b.WriteString("\n---\n")
	b.WriteString(m.fp.View())
	b.WriteString("\n---\n")

	if len(m.Selected) != 0 {
		b.WriteString("\n\n")
		for _, path := range m.Selected {
			fmt.Fprintf(&b, "-> %s\n", path)
		}
	}
	b.WriteString(strings.Repeat("-", 10))

	return b.String()
}
