package sessions

import (
	"context"
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
	// Downloads and uploads can probably be combined, but keeping them seperate for now
	// this makes rendering uploads and downloads seperately easier in the ui
	Downloads map[string]*Session
	Uploads   map[string]*Session
	ui        UIHooks
	dlLock    sync.Mutex
	upLock    sync.Mutex
	ctxGlobal context.Context
}

// maybe track which peer this session belongs to. Needed to check for 403
type Session struct {
	SessionID string
	Files     map[string]*data.File //map between file ids and file structs
	Remaining int
	Peer      *data.PeerInfo
	lock      sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
}
func (s *Session) GetCtx() context.Context {
	return s.ctx	
}

type UIHooks interface {
	// can block until user accpets or times out
	OfferSession(*Session, chan bool)
	FileFinished()
	SessionCreated()
	SessionFinished()
	SessionCancelled()
}

func NewSessionManager(ctx context.Context, basePath string, uihooks UIHooks) *SessionManager {
	return &SessionManager{
		BasePath:  basePath,
		Serial:    1,
		Downloads: make(map[string]*Session),
		Uploads:   make(map[string]*Session),
		ui:        uihooks,
		ctxGlobal: ctx,
	}
}

func (sm *SessionManager) tokenize(sess *data.SessionInfo, file *data.File) string {
	token := fmt.Sprintf("%s.%s", sess.SessionID, file.ID)
	return hex.EncodeToString(sha256.New().Sum([]byte(token)))
}

// asks the ui to accept the session and creates if it if the user accepts. returns nil if the session offer is rejected
func (sm *SessionManager) CreateSession(peer *data.PeerInfo, files map[string]*data.File) *data.SessionInfo {
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
	ctxChild, cancel := context.WithCancel(sm.ctxGlobal)
	sessionCandidate := &Session{
		SessionID: sessID,
		Files:     idToFile,
		Remaining: len(idToFile),
		Peer:      peer,
		ctx:       ctxChild,
		cancel:    cancel,
	}

	res := make(chan bool)
	// TODO: make offer sessions take a timeout context
	sm.ui.OfferSession(sessionCandidate, res)
	timer := time.NewTimer(1 * time.Minute)
	answer := false
	select {
	case <-timer.C:
		slog.Debug("Session offer timed out", slog.Any("sess", sessInfo))
	case answer = <-res:
		slog.Debug("User accepted session")
	}
	if answer {
		sm.dlLock.Lock()
		sm.Downloads[sessInfo.SessionID] = sessionCandidate
		sm.dlLock.Unlock()
		sm.ui.SessionCreated()
		return sessInfo
	} else {
		return nil
	}
}

func (sm *SessionManager) CreateUpload(peer *data.PeerInfo, sess *data.SessionInfo, files map[string]*data.File) string {
	// sm.Serial += 1
	for fileID, file := range files {
		file.Token = sess.Files[fileID]
	}
	sm.upLock.Lock()
	ctxChild, cancel := context.WithCancel(sm.ctxGlobal)
	// sessID := fmt.Sprintf("gclsnd-client-%d", sm.Serial)
	// using the session id from the peer could lead to collisions or security risks
	// this might get acceptable when combined with a check which peer a session belongs to
	// this is not implemented yet, also makes serial a bit useless
	sm.Uploads[sess.SessionID] = &Session{
		SessionID: sess.SessionID,
		Files:     files,
		Remaining: len(files),
		Peer:      peer,
		ctx:       ctxChild,
		cancel:    cancel,
	}
	sm.upLock.Unlock()
	sm.ui.SessionCreated()
	return sess.SessionID
}

func (sm *SessionManager) CancelSession(sessionID string) {
	// TODO: test cancel route with invalid sessionID
	// TODO: Delete associated files if a session is cancelled before it is completed?
	if sess, ok := sm.Downloads[sessionID]; ok {
		sess.cancel()
		sess.cancel = nil
		sess.ctx = nil
		// TODO: Provide info to display about cancelled session
		sm.dlLock.Lock()
		delete(sm.Downloads, sessionID)
		sm.dlLock.Unlock()
		sm.ui.SessionCancelled()
		slog.Debug("removed download", slog.String("id", sessionID))
		return
	}
	if sess, ok := sm.Uploads[sessionID]; ok {
		sess.cancel()
		sess.cancel = nil
		sess.ctx = nil
		sm.upLock.Lock()
		delete(sm.Uploads, sessionID)
		sm.upLock.Unlock()
		sm.ui.SessionCancelled()
		slog.Debug("removed upload", slog.String("id", sessionID))
	}
}

// Finish processing a file. References to sessions can become invalid after calling this if the entire session is finished as well
func (sm *SessionManager) FinishFile(sessID string, fileID string) error {
	set := &sm.Downloads

	sm.dlLock.Lock()
	sm.upLock.Lock()
	if _, isDL := sm.Downloads[sessID]; !isDL {
		if _, isUP := sm.Uploads[sessID]; !isUP {
			sm.upLock.Unlock()
			sm.dlLock.Unlock()
			return errors.New("Invalid session id")
		}
		set = &sm.Uploads
	}
	sm.upLock.Unlock()
	sm.dlLock.Unlock()

	sess := (*set)[sessID]
	sess.lock.Lock()
	defer sess.lock.Unlock()
	if _, ok := sess.Files[fileID]; !ok {
		return errors.New("Invalid file id")
	}
	if !sess.Files[fileID].Done {
		sess.Files[fileID].Done = true
		sess.Remaining -= 1
	}
	if sess.Remaining <= 0 {
		sm.FinishSession(sess.SessionID)
	}
	// TODO: Provide info about the finished file
	sm.ui.FileFinished()
	return nil
}

func (sm *SessionManager) FinishSession(sessionID string) {
	set := &sm.Downloads

	sm.dlLock.Lock()
	sm.upLock.Lock()
	if _, isDL := sm.Downloads[sessionID]; !isDL {
		if _, isUP := sm.Uploads[sessionID]; !isUP {
			for k, v := range sm.Uploads {
				slog.Debug("upload", slog.String("id", k), slog.String("sess", v.SessionID))
			}
			for k, v := range sm.Downloads {
				slog.Debug("download", slog.String("id", k), slog.String("sess", v.SessionID))
			}
			sm.upLock.Unlock()
			sm.dlLock.Unlock()
			slog.Error("called finished session with invalid session", slog.String("id", sessionID))
		}
		set = &sm.Uploads
	}

	sess := (*set)[sessionID]
	sess.cancel = nil
	sess.ctx = nil
	delete(*set, sessionID)
	sm.upLock.Unlock()
	sm.dlLock.Unlock()
	sm.ui.SessionFinished()
	slog.Info("Finished session", slog.String("sessionId", sessionID))
}

// headless implementation of the ui hook interface
type HeadlessUI struct{}

func (hui *HeadlessUI) OfferSession(sess *Session, res chan bool) {
	go func() {
		res <- true
	}()
}

func (hui *HeadlessUI) FileFinished() {
	slog.Debug("file finished", slog.String("src", "headless ui"))
}

func (hui *HeadlessUI) SessionCreated() {
	slog.Debug("session created", slog.String("src", "headless ui"))
}

func (hui *HeadlessUI) SessionFinished() {
	slog.Debug("session finished", slog.String("src", "headless ui"))
}

func (hui *HeadlessUI) SessionCancelled() {
	slog.Debug("headless session cancelled")
}
