package screens

import(
	"fmt"
	"log/slog"
	"strings"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atomic-7/gocalsend/internal/server"
)

type SOModel struct {
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

func NewSessionHandler(sessionManager *server.SessionManager) SOModel {
	return SOModel{
		cursor: 0,
		SessionManager: sessionManager,
		sessionOffers:  make([]*SessionOffer, 0, 10),
	}	
}

func (m *SOModel) cursorUp() {
	if m.cursor > 0 {
		m.cursor -= 1
	}
}

func (m *SOModel) cursorDown() {
	if m.cursor < len(m.sessionOffers)-1 {
		m.cursor += 1
	}
}


func (m *SOModel) acceptSession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].Res <- true
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
}
func (m *SOModel) denySession() {
	if len(m.sessionOffers) != 0 {
		m.sessionOffers[m.cursor].Res <- false
		m.sessionOffers = append(m.sessionOffers[:m.cursor], m.sessionOffers[m.cursor+1:]...)
	}
}
func (m *SOModel) ShouldClose() bool {
	return len(m.sessionOffers) == 0
}

func (m SOModel) Init() tea.Cmd {
	return nil
}

func (m SOModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *SessionOffer:
		m.sessionOffers = append(m.sessionOffers, msg)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.denySession()
			if len(m.sessionOffers) == 0 {
				m.cursor = 0
			} else {
				m.cursor -= 1
			}
			return m, nil
		case tea.KeyUp:
			m.cursorUp()
		case tea.KeyDown:
			m.cursorDown()
		case tea.KeyEnter, tea.KeySpace:
			slog.Info("entry selected", slog.String("screen", "acceptScreen"))
			m.acceptSession()
			if len(m.sessionOffers) == 0 {
				m.cursor = 0
			} else {
				m.cursor -= 1
			}
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				m.denySession()
				m.cursor = 0
				// deny all sessions here?
				if len(m.sessionOffers) == 0 {
					m.cursor = 0
				} else {
					m.cursor -= 1
				}
			case "j":
				m.cursorDown()
			case "k":
				m.cursorUp()
			}
		}
	}
	return m, nil
}

func (m SOModel) View() string {
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
