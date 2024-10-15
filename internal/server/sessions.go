package server

import (
	"errors"
	"fmt"
	"sync"

	"github.com/atomic-7/gocalsend/internal/data"
)

type SessionManager struct {
	Serial   int
	Sessions map[string]*Session
	lock     sync.Mutex
}

type Session struct {
	SessionID string
	Files     map[string]*data.File //map between file ids and file structs
	Finished  int
	lock      sync.Mutex
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
