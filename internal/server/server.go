package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/sessions"
)

func reqLogger(w http.ResponseWriter, r *http.Request) {
	slog.Info("request", slog.Any("request", r))
}

func createPrepareUploadHandler(sman *sessions.SessionManager, peers data.PeerTracker) http.Handler {
	logga := slog.Default().With(slog.String("handler", "prepare upload"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 204 Finished, no file transfer needed
		// 400 Invalid body
		// 401 Pin required / invalid pin
		// 403 Rejected
		// 409 Blocked by another session
		// 429 Too many requests
		// 500 Server error
		// TODO: Use ParseForm to get pin, it only reads the body when content-type is urlencoded
		payload := &data.PreparePayload{
			Files: make(map[string]*data.File),
		}

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			logga.Error("could not read request body", slog.Any("error", err))
			return
		}
		err = json.Unmarshal(buf, payload)
		if err != nil {
			w.WriteHeader(500)
			logga.Error("could not unmarshal payload", string(buf[0:100]), err)
			return
		}

		logga.Debug("incoming session", slog.Any("peer", payload.Info))
		logga.Debug("session files", slog.Any("files", payload.Files))
		logga.Debug("Files to tokens")
		// maybe track the client to which this session belongs?
		// TODO: use session manager to check if the user wants to accept the incoming request
		pred := func(p *data.PeerInfo) bool {
			return p.IP.Equal(net.IP(r.RemoteAddr))
		}
		peer := peers.Find(pred)
		sess := sman.CreateSession(peer, payload.Files)
		if sess == nil {
			w.WriteHeader(403)
			logga.Debug("user declined session")
			return
		}
		for fid, tok := range sess.Files {
			logga.Info("[File]", slog.String("fileID", fid), slog.String("token", tok))
		}

		resp, err := json.Marshal(sess)
		if err != nil {
			w.WriteHeader(500)
			logga.Error("failed to marshal session", slog.Any("session", sess), slog.Any("error", err))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(resp)
		if err != nil {
			logga.Error("failed to send the payload", slog.Any("error", err))
			return
		}
	})
}

func SessionReader(w http.ResponseWriter, r *http.Request) {

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read request body", slog.String("handler", "session reader"), slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("session raw string", slog.String("bytes", string(buf)))
	//var sess *data.Session
	sess := &data.SessionInfo{}
	sess.Files = make(map[string]string)
	err = json.Unmarshal(buf, sess)
	if err != nil {
		slog.Error("failed to unmarshal into session", slog.Any("error", err))
		w.WriteHeader(400)
		return
	}
	slog.Info("received session", slog.String("id", sess.SessionID))
	for fk, fv := range sess.Files {
		slog.Info("[File]", slog.String("fileID", fk), slog.String("token", fv))
	}
	// Localsend Phone Client: type 'String' is not a subtype of type 'Map<String, dynamic>'
}

func createUploadHandler(sman *sessions.SessionManager) http.Handler {
	logga := slog.Default().With(slog.String("handler", "upload"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 400 missing parameters
		// 403 invalid token or ip addr
		// 409 blocked by another session
		// 500 Server error
		r.ParseForm()
		// TODO: Check for malicious url parameters
		// TODO: Use http.Error instead of write header for better feedback for requests
		if !r.Form.Has("sessionId") || !r.Form.Has("fileId") || !r.Form.Has("token") {
			logga.Error("request with invalid url parameters", slog.String("url", r.URL.String()))
			slog.Debug("expected parameters",
				slog.String("sessionId", r.Form.Get("sessionId")),
				slog.String("fileId", r.Form.Get("fileId")),
				slog.String("tokekn", r.Form.Get("token")),
			)
			w.WriteHeader(400)
			return
		}
		sessID := r.Form.Get("sessionId")
		fileID := r.Form.Get("fileId")
		token := r.Form.Get("token")
		if _, ok := sman.Downloads[sessID]; !ok {
			logga.Error("invalid session", slog.String("sessionId", sessID))
			w.WriteHeader(403)
			return
		}
		// an upload from a peer is a download to the local node
		sess := sman.Downloads[sessID]
		// TODO: Check if the sending peer is associated with this session in the session manager
		if _, ok := sess.Files[fileID]; !ok {
			logga.Error("invalid fileid", slog.String("fileId", fileID))
			w.WriteHeader(403)
			return
		}
		file := sess.Files[fileID]
		if file.Token != token {
			logga.Error("valid session and id with invalid token", slog.String("file token", file.Token), slog.String("url token", token))
			w.WriteHeader(500)
			return
		}
		path := sman.BasePath
		if file.Destination != "" {
			path = file.Destination
			// TODO: Create potentially missing folders
		}
		logga.Debug("dl path", slog.String("path", path), slog.String("name", file.FileName), slog.String("dest", file.Destination))

		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			logga.Error("failed to create output directory", slog.String("out", path))
		}

		osFile, err := os.Create(filepath.Join(path, file.FileName))
		defer osFile.Close()
		if err != nil {
			logga.Error("failed to create file ", slog.String("file", path+"/"+file.FileName), slog.Any("error", err))
			w.WriteHeader(500)
			return
		}

		_, err = osFile.ReadFrom(r.Body) // could probably also use io.Copy
		if err != nil {
			logga.Error("failed to write to file", slog.String("file", file.FileName), slog.Any("error", err))
			w.WriteHeader(500)
			return
		}

		// Not a deferred close to be able to catch errors that might happen when closing a file after writing
		err = osFile.Close()
		if err != nil {
			logga.Error("failed to close file", slog.String("file", file.FileName), slog.Any("error", err))
			w.WriteHeader(500)
			return
		}

		logga.Info("file downloaded", slog.String("sessionId", sess.SessionID), slog.String("file", file.FileName), slog.String("path", path))
		sman.FinishFile(sess.SessionID, fileID)
	})
}

func createCancelHandler(sman *sessions.SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			slog.Error("failed to parse query parameters for cancel request", slog.Any("error", err), slog.String("handler", "cancel"))
			w.WriteHeader(500)
			return
		}
		if !r.Form.Has("sessionId") { // It seems that the reference implementation does not send a sessionId to cancel?
			slog.Error("cancel request without session id")
			slog.Debug("cancel query", slog.String("url", r.URL.String()), slog.String("handler", "cancel"))
			w.WriteHeader(400)
			return
		}
		sessID := r.Form.Get("sessionId")
		sman.CancelSession(sessID)
		slog.Debug("cancelled session", slog.String("id", sessID))
	})
}

func createInfoHandler(nodeJson []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// apparently the localsend implementation expects some response here? https://github.com/localsend/localsend/blob/main/common/lib/src/discovery/http_target_discovery.dart
		// content-type: application/json; charset=utf-8
		// x-frame-options: SAMEORIGIN
		// x-xss-protection: 1; mode=block
		// transfer-encoding: chunked
		// x-content-type-options: nosniff
		// {"alias":"Strategic Carrot","version":"2.1","deviceModel":"Linux","deviceType":"desktop","fingerprint":"1E6045836FC02E3A88B683FA47DDBD4E4CBDFD3F5C8C65136F118DFB9B0F2ACE","download":false}

		r.ParseForm()
		slog.Info("incoming request", slog.String("url", r.URL.String()), slog.Any("form", r.Form))
		w.Header().Add("Content-Type", "application/json")
		w.Write(nodeJson)
	})

}

// Registry seems to work when encryption is turned of for the peer, but not when active
func createRegisterHandler(localNode *data.PeerInfo, peers data.PeerTracker) http.Handler {
	logga := slog.Default().With(slog.String("handler", "register"))
	regResp, err := json.Marshal(localNode.ToRegisterResponse())
	if err != nil {
		logga.Error("Could not marshal local node for response to register handler: ", err)
		os.Exit(1)
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		logga.Debug("incoming registry via api", slog.String("url", r.URL.String()))
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			logga.Error("failed to read request body", slog.Any("error", err))
			os.Exit(1)
		}
		var peer data.PeerInfo
		json.Unmarshal(buf, &peer)
		parts := strings.Split(r.RemoteAddr, ":")
		peer.IP = net.ParseIP(parts[0])
		if peer.IP == nil {
			logga.Error("failed to parse peer ip", slog.Any("host", r.Host))
			os.Exit(1)
		}
		// TODO: maybe reuse the registratinator here?
		if peers.Add(&peer) {
			logga.Info("registering peer", slog.String("peer", peer.Alias))
		} else {
			logga.Debug("peer was already known", slog.String("peer", peer.Alias))
		}
		writer.Write(regResp)
	})
}

func StartServer(ctx context.Context, localNode *data.PeerInfo, peers data.PeerTracker, sessionManager *sessions.SessionManager, tlsInfo *data.TLSPaths, downloadBase string) {

	if peers == nil {
		slog.Error("failed to setup server", slog.String("reason", "peertracker is nil"))
		os.Exit(1)
	}
	jsonBuf, err := json.Marshal(localNode.ToPeerBody())
	if err != nil {
		slog.Error("failed to unmarshal local node to json", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Debug("NodeJson", slog.String("json", string(jsonBuf)))

	infoHandler := createInfoHandler(jsonBuf)
	prepUploadHandler := createPrepareUploadHandler(sessionManager, peers)
	uploadHandler := createUploadHandler(sessionManager)
	cancelHandler := createCancelHandler(sessionManager)
	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", createRegisterHandler(localNode, peers))
	mux.Handle("/api/localsend/v1/info", infoHandler)
	mux.Handle("/api/localsend/v2/info", infoHandler)
	mux.Handle("/api/localsend/v2/prepare-upload", prepUploadHandler)
	mux.Handle("/api/localsend/v1/upload", uploadHandler)
	mux.Handle("/api/localsend/v2/upload", uploadHandler)
	mux.Handle("/api/localsend/v2/cancel", cancelHandler)
	mux.HandleFunc("/testing/sessions", SessionReader)
	mux.HandleFunc("/", reqLogger)

	var srv http.Server
	port := fmt.Sprintf(":%d", localNode.Port)
	slog.Info("server started", slog.Int("port", localNode.Port), slog.String("protocol", localNode.Protocol))

	// Might have to use InsecureSkipVerify here with a VerifyConnection function to check against the known fingerprints?
	// TODO: Look into VerifyConnection
	// TODO: ErrorLog
	if tlsInfo != nil {
		slog.Debug("setting up https api")
		srv = http.Server{
			Addr:    port,
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		}
		slog.Error("server error", slog.Any("error", srv.ListenAndServeTLS(tlsInfo.Cert, tlsInfo.Key)))
		os.Exit(1)
	} else {
		srv = http.Server{
			Addr:    port,
			Handler: mux,
		}
		slog.Error("server error", slog.Any("error", srv.ListenAndServe()))
	}
}
