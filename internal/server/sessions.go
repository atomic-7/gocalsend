package server

import (
	"errors"
	"fmt"
	"sync"

	"github.com/atomic-7/gocalsend/internal/data"
)

type SessionManager struct {
	BasePath string
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

func NewSessionManager(basePath string) *SessionManager {
	return &SessionManager{
		BasePath: basePath,
		Serial:   1,
		Sessions: make(map[string]*Session),
	}
}

func (sm *SessionManager) tokenize(sess *data.SessionInfo, file *data.File) string {
	return fmt.Sprintf("%s#%s", sess.SessionID, file.ID)
}

func (sm *SessionManager) CreateSession(files map[string]*data.File) *data.SessionInfo {
	fileToToken := make(map[string]string, len(files))
	idToFile := make(map[string]*data.File, len(files))
	sm.Serial += 1
	sessID := fmt.Sprintf("gclsnd-%d", sm.Serial)
	sessInfo := &data.SessionInfo{
		SessionID: sessID,
		Files:     fileToToken,
	}
	// TODO: Check if there are already existing files with the same name in the BasePath
	for fileID, file := range files {
		files[fileID].ID = fileID
		fileToToken[fileID] = sm.tokenize(sessInfo, file)
		idToFile[fileID] = file
	}
	sm.lock.Lock()
	sm.Sessions[sessInfo.SessionID] = &Session{
		SessionID: sessID,
		Files:     idToFile,
		Finished:  0,
	}
	sm.lock.Unlock()
	return sessInfo
}

func (sm *SessionManager) CancelSession(sessionID string) {
	// Delete associated files if a session is cancelled before it is completed?
	if _, ok := sm.Sessions[sessionID]; ok {
		sm.lock.Lock()
		delete(sm.Sessions, sessionID)
		sm.lock.Unlock()
	}
}

func (sm *SessionManager) FinishFile(sessID string, fileID string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if _, ok := sm.Sessions[sessID]; !ok {
		return errors.New("Invalid session id")
	}
	sess := sm.Sessions[sessID]
	sess.lock.Lock()
	defer sess.lock.Unlock()
	if _, ok := sess.Files[fileID]; !ok {
		return errors.New("Invalid file id")
	}
	sess.Files[fileID].Done = true
	sess.Finished += 1
	return nil
}

func (sm *SessionManager) FinishSession(sessionId string) {
	sm.lock.Lock()
	delete(sm.Sessions, sessionId)
	sm.lock.Unlock()
}
