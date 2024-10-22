package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/atomic-7/gocalsend/internal/data"
)

func reqLogger(w http.ResponseWriter, r *http.Request) {
	log.Printf("RQ: %v", r)
}

func createPrepareUploadHandler(sman *SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 204 Finished, no file transfer needed
		// 400 Invalid body
		// 401 Pin required / invalid pin
		// 403 Rejected
		// 409 Blocked by another session
		// 429 Too many requests
		// 500 Server error
		// TODO: Use ParseForm to get pin, it only reads the body when content-type is urlencoded
		// TODO: adjust error codes
		payload := &data.PreparePayload{
			Files: make(map[string]*data.File),
		}

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			log.Fatal("Could not read request body for prepare upload request: ", err)
		}
		err = json.Unmarshal(buf, payload)
		if err != nil {
			w.WriteHeader(500)
			log.Fatalf("Could not unmarshal payload from %v: %v ", string(buf[0:100]), err)
		}

		log.Printf("Received upload prep info: %v\n", payload.Info)
		log.Printf("Files: %v\n", payload.Files)
		log.Println("Files to tokens")
		// files := make(map[string]string)
		// for fk, fv := range payload.Files {
		// 	fmt.Printf("[File] %s: %v\n", fk, fv)
		// 	files[fk] = fmt.Sprintf("TOK:%s", fv)
		// }
		// maybe track the client to which this session belongs?
		sess := sman.CreateSession(payload.Files)
		for fid, tok := range sess.Files {
			fmt.Printf("[File] %s: TOK(%s)\n", fid, tok)
		}

		// session := &data.SessionInfo{
		// 	SessionID: "not implemented yet",
		// 	Files:     files,
		// }
		resp, err := json.Marshal(sess)
		if err != nil {
			w.WriteHeader(500)
			log.Fatal("Failed to marshal the example response: ", err)
		}
		w.Write(resp)
		// w.WriteHeader(403) // reject all requests for now
	})
}

func SessionReader(w http.ResponseWriter, r *http.Request) {

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Failed to read request body: ", err)
	}

	log.Printf("SessBody: %s\n", string(buf))
	//var sess *data.Session
	sess := &data.SessionInfo{}
	sess.Files = make(map[string]string)
	err = json.Unmarshal(buf, sess)
	if err != nil {
		log.Fatal("Failed to unmarshal into session: ", err)
	}
	log.Printf("Received session %s", sess.SessionID)
	for fk, fv := range sess.Files {
		fmt.Printf("[File] %s : %v\n", fk, fv)
	}
}

func createUploadHandler(sman *SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 400 missing parameters
		// 403 invalid token or ip addr
		// 409 blocked by another session
		// 500 Server error
		r.ParseForm()
		// TODO: Check for malicious url parameters
		if !r.Form.Has("sessionId") || !r.Form.Has("fileId") || !r.Form.Has("token") {
			log.Printf("Req %s had invalid url params\n", r.URL)
			log.Printf("sessionID: %s | fileID: %s | token: %s", r.Form.Get("sessionId"), r.Form.Get("fileId"), r.Form.Get("token"))
			w.WriteHeader(400)
			return
		}
		sessID := r.Form.Get("sessionId")
		fileID := r.Form.Get("fileId")
		token := r.Form.Get("token")
		if _, ok := sman.Sessions[sessID]; !ok {
			log.Printf("Invalid session %s\n", sessID)
			w.WriteHeader(403)
			return
		}
		sess := sman.Sessions[sessID]
		if _, ok := sess.Files[fileID]; !ok {
			log.Printf("Invalid fileid %s\n", fileID)
			w.WriteHeader(403)
			return
		}
		file := sess.Files[fileID]
		if file.Token != token {
			log.Printf("Valid session and id with invalid token: %s != %s\n", file.Token, token)
			w.WriteHeader(500)
			return
		}
		path := sman.BasePath
		if file.Destination != "" {
			path = file.Destination
		}

		osFile, err := os.Create(path + "/" + file.FileName)
		defer osFile.Close()
		if err != nil {
			log.Printf("Failed to create the file %s: %v\n", path+"/"+file.FileName, err)
			w.WriteHeader(500)
			return
		}
		_, err = osFile.ReadFrom(r.Body) // could probably also use io.Copy
		if err != nil {
			log.Printf("Failed to write to file %s: %v\n", file.FileName, err)
			w.WriteHeader(500)
			return
		}
		err = osFile.Close()
		if err != nil {
			log.Printf("Failed to close file %s: %v\n", file.FileName, err)
			w.WriteHeader(500)
			return
		}

		log.Printf("[%s] Downloaded %s to %s", sess.SessionID, file.FileName, path)
		sman.FinishFile(sess.SessionID, fileID)
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
		log.Printf("%s: %v", r.URL, r.Form)
		w.Header().Add("Content-Type", "application/json")
		w.Write(nodeJson)
	})

}

// Registry seems to work when encryption is turned of for the peer, but not when active
func createRegisterHandler(localNode *data.PeerInfo, peers *data.PeerMap) http.Handler {
	regResp, err := json.Marshal(localNode.ToRegisterResponse())
	if err != nil {
		log.Fatal("Could not marshal local node for response to register handler: ", err)
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		log.Printf("Incoming registry via api: %s", r.URL)
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading register body: ", err)
		}
		var peer data.PeerInfo
		json.Unmarshal(buf, &peer)
		log.Printf("Registering %s via api register route", peer.Alias)
		pm := *peers.GetMap()
		defer peers.ReleaseMap()
		if _, ok := pm[peer.Fingerprint]; ok {
			log.Printf("%s was already a known peer", peer.Alias)
		} else {
			pm[peer.Fingerprint] = &peer
		}
		writer.Write(regResp)
	})
}

func StartServer(ctx context.Context, localNode *data.PeerInfo, peers *data.PeerMap, tlsInfo *data.TLSPaths) {

	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}
	jsonBuf, err := json.Marshal(localNode.ToPeerBody())
	if err != nil {
		log.Fatal("Error marshalling local node to json: ", err)
	}
	log.Printf("NodeJson: %s", string(jsonBuf))

	sessionManager := NewSessionManager("/home/atomic/Downloads/gocalsend")

	infoHandler := createInfoHandler(jsonBuf)
	prepUploadHandler := createPrepareUploadHandler(sessionManager)
	uploadHandler := createUploadHandler(sessionManager)
	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", createRegisterHandler(localNode, peers))
	mux.Handle("/api/localsend/v1/info", infoHandler)
	mux.Handle("/api/localsend/v2/info", infoHandler)
	mux.Handle("/api/localsend/v2/prepare-upload", prepUploadHandler)
	mux.Handle("/api/localsend/v1/upload", uploadHandler)
	mux.Handle("/api/localsend/v2/upload", uploadHandler)
	mux.HandleFunc("/testing/sessions", SessionReader)
	mux.HandleFunc("/", reqLogger)

	var srv http.Server
	port := fmt.Sprintf(":%d", localNode.Port)
	fmt.Printf("Server running at %d\n", localNode.Port)

	// Might have to use InsecureSkipVerify here with a VerifyConnection function to check against the known fingerprints?
	// TODO: Look into VerifyConnection
	if tlsInfo != nil {
		log.Println("Setup https api")
		srv = http.Server{
			Addr:    port,
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		}
		log.Fatal(srv.ListenAndServeTLS(tlsInfo.CertPath, tlsInfo.KeyPath))
	} else {
		srv = http.Server{
			Addr:    port,
			Handler: mux,
		}
		log.Fatal(srv.ListenAndServe())
	}
}
