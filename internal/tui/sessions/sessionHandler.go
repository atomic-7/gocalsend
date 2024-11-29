package sessions

import(
	"fmt"
	"log/slog"
	"strings"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/server"
)

type Model struct {
		cursor int
		SessionManager *server.SessionManager
		sessionOffers  []*SessionOffer
}
type SessionOffer struct {
	Sess *server.Session
	Res  ResponseChannel
}
type ResponseChannel = chan bool
type SessionFinished bool

func NewSessionHandler(sessionManager *server.SessionManager) Model {
	return Model{
		cursor: 0,
		SessionManager: sessionManager,
		sessionOffers:  make([]*SessionOffer, 0, 10),
	}	
}

func (m *Model) cursorUp() {
	if m.cursor > 0 {
		m.cursor -= 1
	}
}

func (m *Model) cursorDown() {
	if m.cursor < len(m.sessionOffers)-1 {
		m.cursor += 1
	}
}


func (m *Model) acceptSession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].Res <- true
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
	if len(m.sessionOffers) == 0 {
		m.cursor = 0
	} else {
		m.cursor -= 1
	}
}
func (m *Model) denySession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].Res <- false
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
	if len(m.sessionOffers) == 0 {
		m.cursor = 0
	} else {
		m.cursor -= 1
	}
}
func (m *Model) denyAll() {
	for _, offer := range m.sessionOffers {
		offer.Res <- false
	}
	m.sessionOffers = make([]*SessionOffer, 0, 10)
}
func (m *Model) ShouldClose() bool {
	return len(m.sessionOffers) == 0
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *SessionOffer:
		m.sessionOffers = append(m.sessionOffers, msg)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.denyAll()
			return m, tea.Quit
		case tea.KeyUp:
			m.cursorUp()
		case tea.KeyDown:
			m.cursorDown()
		case tea.KeyEnter, tea.KeySpace:
			slog.Info("entry selected", slog.String("screen", "acceptScreen"))
			m.acceptSession()
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				m.denyAll()
				m.cursor = 0
			case "j":
				m.cursorDown()
			case "k":
				m.cursorUp()
			case "y":
				m.acceptSession()
			case "n":
				m.denySession()
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("Incoming transfers\n")
	slog.Debug("render", slog.Any("sessions", m.sessionOffers))
	for i, offer := range m.sessionOffers {
		indicator := " "
		if m.cursor == i {
			indicator = ">"
		}
		fmt.Fprintf(&b, "%s | %s\n", indicator, offer.Sess.SessionID)
		for _, file := range offer.Sess.Files {
			fmt.Fprintf(&b, "  # %s - %d \n", file.FileName, file.Size)
		}
	}

	b.WriteString("\nPress Enter/Space to accept.\nPress q or Ctrl+C to deny.\n")

	return b.String()
}
