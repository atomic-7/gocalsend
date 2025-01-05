package screens

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type FSModel struct {
	cursor   int
	path     string
	listings map[string][]fs.DirEntry
	selected map[fs.DirEntry]struct{}
	err      error
}
type DirEntriesMsg struct {
	entries []fs.DirEntry
	err     error
}

func createFileSelect(path string) FSModel {
	return FSModel{
		cursor:   0,
		path:     path,
		listings: make(map[string][]fs.DirEntry),
		err:      nil,
	}
}

func (fsm *FSModel) cursorUp() {
	if fsm.cursor > 0 {
		fsm.cursor -= 1
	}
}

func (fsm *FSModel) cursorDown() {
	if fsm.cursor < len(fsm.listings[fsm.path])-1 {
		fsm.cursor += 1
	}
}

func (fsm *FSModel) cursorSelect() {
	key := fsm.listings[fsm.path][fsm.cursor]
	_, ok := fsm.selected[key]
	if ok {
		delete(fsm.selected, key)
	} else {
		fsm.selected[key] = struct{}{}
	}
}

func openDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		var result DirEntriesMsg
		if err != nil {
			result.err = err
			slog.Error("failed to open dir", slog.String("src", "file select"), slog.String("dir", path))
		}
		result.entries = entries
		return result
	}
}

func (fsm FSModel) Init() tea.Cmd {
	return openDir(fsm.path)
}

func (fsm FSModel) Update(msg tea.Msg) (FSModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return fsm, tea.Quit
		case tea.KeyUp:
			fsm.cursorUp()
		case tea.KeyDown:
			fsm.cursorDown()
		case tea.KeySpace:
			slog.Info("entry selected", slog.String("screen", "file select"))
			fsm.cursorSelect()
		case tea.KeyEnter:
			// TODO: figure out how to pass the control back to the main program
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return fsm, tea.Quit
			case "j":
				fsm.cursorDown()
			case "k":
				fsm.cursorUp()
			}
		}
	case DirEntriesMsg:
		if msg.err != nil {
			fsm.listings[fsm.path] = msg.entries
		} else {
			fsm.err  = msg.err
		}
	}
	return fsm, nil
}

func (fsm FSModel) View() string {
	if fsm.err != nil {
		return fmt.Sprintf("Error opening %s: %v", fsm.path, fsm.err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# File select - %s", fsm.path)
	for _, entry := range fsm.listings[fsm.path] {
		indicator := " "
		fmt.Fprintf(&b, "%s | %s", indicator, entry.Name())
	}
	return "selecting files!"
}
