package filepicker

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func New() Model {
	fp := filepicker.New()
	home, err := os.UserHomeDir()
	fp.DirAllowed = false // not implemented yet
	fp.AutoHeight = true
	fp.KeyMap.Open = key.NewBinding(key.WithKeys("l", "right", " "), key.WithHelp("l", "open"))
	fp.KeyMap.Select = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))
	if err != nil {
		// TODO: display error or just use the cwd the program was started from
		slog.Debug("failed to get path to home dir", slog.Any("err", err))
	}
	fp.CurrentDirectory = home

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	fl := list.New(make([]list.Item, 0, 10), delegate, 5, 14)
	fl.InfiniteScrolling = true
	fl.SetShowHelp(false)
	fl.SetShowStatusBar(false)
	fl.Title = "Selected Files"
	return Model{
		Done:     false,
		focus:    screen(FILEPICKER),
		fp:       fp,
		fileList: fl,
		KeyMap:   DefaultKeyMap(),
		help:     help.New(),
	}
}

const (
	FILEPICKER = iota
	SELECTEDFILES
)

type screen = int

type Model struct {
	Done     bool
	Selected []string
	width    int
	height   int
	focus    screen
	fp       filepicker.Model
	fileList list.Model
	KeyMap   KeyMap
	help     help.Model
}

type selectedItem struct {
	path string
}

func (i selectedItem) Title() string       { return i.path }
func (i selectedItem) Description() string { return i.path }
func (i selectedItem) FilterValue() string { return i.path }

func (m Model) Init() tea.Cmd {
	// fileList does not need to be initialized
	return m.fp.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		msg.Height -= m.fileList.Height()
		// ignore the cmd because the filepicker responds with nil cmd for resize msgs
		m.fileList.SetWidth(msg.Width)
		m.fp, _ = m.fp.Update(msg)
		return m, nil
	case tea.KeyMsg:
		switch m.focus {
		case FILEPICKER:
			switch {
			case key.Matches(msg, m.KeyMap.Confirm):
				if len(m.Selected) != 0 {
					m.Done = true
					slog.Debug("confirm key caught", slog.Int("files", len(m.Selected)), slog.String("src", "filepicker"))
				}
			}
		case SELECTEDFILES:
			switch {
			case key.Matches(msg, m.KeyMap.Confirm):
				slog.Debug("deselecting", slog.String("file", m.fileList.SelectedItem().FilterValue()))
				m.fileList.RemoveItem(m.fileList.Index())
			}
		}
		switch {
		case key.Matches(msg, m.KeyMap.FocusFilepicker):
			m.focus = screen(FILEPICKER)
			m.fileList.SetShowHelp(false)
			return m, nil
		case key.Matches(msg, m.KeyMap.FocusSelectedFiles):
			m.focus = screen(SELECTEDFILES)
			m.fileList.SetShowHelp(true)
			return m, nil
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	switch m.focus {
	case FILEPICKER:
		m.fp, cmd = m.fp.Update(msg)
	case SELECTEDFILES:
		m.fileList, cmd = m.fileList.Update(msg)
	}

	// check if a file was selected
	if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
		slog.Debug("file selected", slog.String("path", path))
		m.Selected = append(m.Selected, path)
		m.fileList.InsertItem(
			len(m.fileList.Items()),
			selectedItem{path},
		)
		resizeMsg := tea.WindowSizeMsg{
			Width:  m.width,
			Height: m.height - m.fileList.Height(),
		}
		m.fp, _ = m.fp.Update(resizeMsg)
	}
	// check if a file was selected that the filters disable
	// if didSelect, path := m.fp.DidSelectDisabledFile(msg); didSelect {
	// 	//TODO: implement display of error message
	// }
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "File Selection > %s\n", m.fp.CurrentDirectory)
	b.WriteString(m.fp.View())
	b.WriteString(m.fileList.View())

	// TODO: figure out how to handle the help texts better
	b.WriteString("\n")
	b.WriteString(m.help.View(m.KeyMap))
	b.WriteString("\n")

	return b.String()
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		// ctrl+enter is not yet supported by bubbletea and is somewhat problematic in terminals
		Confirm:            key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm selection")),
		FocusFilepicker:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pick files")),
		FocusSelectedFiles: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "remove selected files")),
		Quit:               key.NewBinding(key.WithKeys("q", "ctrl+c", "ctrl+q"), key.WithHelp("q", "quit")),
	}
}

type KeyMap struct {
	Confirm            key.Binding
	FocusFilepicker    key.Binding
	FocusSelectedFiles key.Binding
	Quit               key.Binding
}

// keybindinds to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.FocusFilepicker, k.FocusSelectedFiles, k.Quit}
}

// keybinds to be shown in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.FocusFilepicker, k.FocusSelectedFiles, k.Quit},
	}
}
