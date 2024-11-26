package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atomic-7/gocalsend/internal/data"
)

type SessionManager struct {
	BasePath string
	Serial   int
	Sessions map[string]*Session
	lock     sync.Mutex
	ui        UIHooks
}

// maybe track which peer this session belongs to. Needed to check for 403
type Session struct {
	SessionID string
	Files     map[string]*data.File //map between file ids and file structs
	Finished  int
	lock      sync.Mutex
}

type UIHooks interface {
	OfferSession(*data.SessionInfo, chan bool)
	SessionFinished()
}

func NewSessionManager(basePath string, uihooks UIHooks) *SessionManager {
	return &SessionManager{
		BasePath: basePath,
		Serial:   1,
		Sessions: make(map[string]*Session),
		ui: uihooks,
	}
}

func (sm *SessionManager) tokenize(sess *data.SessionInfo, file *data.File) string {
	token := fmt.Sprintf("%s.%s", sess.SessionID, file.ID)
	return hex.EncodeToString(sha256.New().Sum([]byte(token)))
}

// asks the ui to accept the session and creates if it if the user accepts. returns nil if the session offer is rejected
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
		token := sm.tokenize(sessInfo, file)
		files[fileID].Token = token
		fileToToken[fileID] = token
		idToFile[fileID] = file
	}

	res := make(chan bool)
	sm.ui.OfferSession(sessInfo, res)
	timer := time.NewTimer(1 * time.Minute)
	answer := false
	select {
		case <- timer.C:
			slog.Debug("Session offer timed out", slog.Any("sess", sessInfo))
		case answer = <- res:
			slog.Debug("User accepted session")
	}
	if answer {
		sm.lock.Lock()
		sm.Sessions[sessInfo.SessionID] = &Session{
			SessionID: sessID,
			Files:     idToFile,
			Finished:  len(idToFile),
		}
		sm.lock.Unlock()
		return sessInfo
	} else {
		return nil
	}
}

func (sm *SessionManager) RegisterSession(sess *data.SessionInfo, files map[string]*data.File) string {
	sm.Serial += 1
	for fileID, file := range files {
		file.Token = sess.Files[fileID]
	}
	sm.lock.Lock()
	sessID := fmt.Sprintf("gclsnd-client-%d", sm.Serial)
	sm.Sessions[sessID] = &Session{
		SessionID: sess.SessionID,
		Files:     files,
		Finished:  len(files),
	}
	sm.lock.Unlock()
	return sessID
}

func (sm *SessionManager) CancelSession(sessionID string) {
	// Delete associated files if a session is cancelled before it is completed?
	if _, ok := sm.Sessions[sessionID]; ok {
		sm.lock.Lock()
		delete(sm.Sessions, sessionID)
		sm.lock.Unlock()
	}
}

// Finish processing a file. References to sessions can become invalid after calling this if the entire session is finished as well
func (sm *SessionManager) FinishFile(sessID string, fileID string) error {
	sm.lock.Lock()
	if _, ok := sm.Sessions[sessID]; !ok {
		return errors.New("Invalid session id")
	}
	sm.lock.Unlock()
	sess := sm.Sessions[sessID]
	sess.lock.Lock()
	defer sess.lock.Unlock()
	if _, ok := sess.Files[fileID]; !ok {
		return errors.New("Invalid file id")
	}
	if !sess.Files[fileID].Done {
		sess.Files[fileID].Done = true
		sess.Finished -= 1
	}
	if sess.Finished <= 0 {
		sm.FinishSession(sess.SessionID)
	}
	return nil
}

func (sm *SessionManager) FinishSession(sessionId string) {
	sm.lock.Lock()
	delete(sm.Sessions, sessionId)
	sm.lock.Unlock()
	sm.ui.SessionFinished()
	slog.Info("Finished session", slog.String("sessionId", sessionId))
}

// headless implementation of the ui hook interface
type HeadlessUI struct {}

func (hui *HeadlessUI) OfferSession(sess *data.SessionInfo, res chan bool) {
	res <- true
}

func (hui *HeadlessUI) SessionFinished() {
	slog.Debug("headless session finished")
}
