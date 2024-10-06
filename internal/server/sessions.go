package server

import (
	"fmt"
	"github.com/atomic-7/gocalsend/internal/data"
)

type SessionManager struct {
	Serial int
	Sessions []*data.Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Serial: 1,
		Sessions: make([]*data.Session, 10),
	}
}

func (sm *SessionManager) createSession(files map[string]*data.File) *data.Session {
	sm.Serial += 1
	return &data.Session{
		SessionId: fmt.Sprintf("gclsnd-%d", sm.Serial),
		Files: files,
	}
}
