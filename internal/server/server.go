package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/atomic-7/gocalsend/internal/data"
)

func reqLogger(w http.ResponseWriter, r *http.Request) {
	log.Printf("RQ: %v", r)
}

func HandlePrepareUpload(w http.ResponseWriter, r *http.Request) {
	// TODO: Figure out how to parse the url parameters with parseForm but not the body
	var rawKeys map[string]json.RawMessage

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		log.Fatal("Could not read request body for prepare upload request: ", err)
	}
	err = json.Unmarshal(buf, &rawKeys)
	if err != nil {
		w.WriteHeader(500)
		log.Fatalf("Could not unmarshal raw keys from %v: %v ", string(buf[0:100]), err)
	}

	// Check required fields
	// TODO: Warn if the json has more fields than just info and files
	if _,ok := rawKeys["info"]; !ok {
		w.WriteHeader(500)	// change this, info is missing from post req
		log.Fatalf("posted json did not have an info key: %s\n", string(buf[0:100]))
	}
	if _,ok := rawKeys["files"]; !ok {
		w.WriteHeader(500)	// change this, files is missing from post req
		log.Fatalf("posted json did not have a files key: %s\n", string(buf[0:100]))
	}

	var peerInfo *data.PeerInfo
	var fileMap map[string]*data.File
	// Unmarshal raw msgs into their respective fields
	err = json.Unmarshal(rawKeys["info"], &peerInfo)
	if err != nil {
		w.WriteHeader(500)
		log.Fatalf("Failed to unmarshal info struct from %v: %v\n", string(buf[0:100]), err)
	}
	err = json.Unmarshal(rawKeys["files"], &fileMap)
	if err != nil {
		w.WriteHeader(500)
		log.Fatalf("Failed to unmarshal files map: %v\n", err)
	}
	
	log.Printf("Received upload prep info: %v\n", peerInfo)
	log.Println("Files to receive")
	for fk, fv := range fileMap {
		fmt.Printf("[File] %s: %v\n", fk, fv)
	}
	files := make(map[string]*data.File)
	files["example file"] = &data.File{
		Id:       "example file",
		FileName: "example.txt",
		Size:     0,
		FileType: "text",
		Metadata: nil,
	}
	session := &data.Session{
		SessionId: "not implemented yet",
		Files:     files,
	}
	resp, err := json.Marshal(session)
	if err != nil {
		w.WriteHeader(500)
		log.Fatal("Failed to marshal the example response: ", err)
	}
	w.Write(resp)
	// w.WriteHeader(403) // reject all requests for now
}

func SessionReader(w http.ResponseWriter, r *http.Request) {

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Failed to read request body: ", err)
	}
	
	log.Printf("SessBody: %s\n", string(buf))
	//var sess *data.Session
	sess := &data.Session{}
	sess.Files = make(map[string]*data.File)
	err = json.Unmarshal(buf, sess)
	if err != nil {
		log.Fatal("Failed to unmarshal into session: ", err)
	}
	log.Printf("Received session %s", sess.SessionId)
	for fk, fv := range sess.Files {
		fmt.Printf("[File] %s : %v\n", fk, fv)
	}
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

	infoHandler := createInfoHandler(jsonBuf)
	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", createRegisterHandler(localNode, peers))
	mux.Handle("/api/localsend/v1/info", infoHandler)
	mux.Handle("/api/localsend/v2/info", infoHandler)
	mux.HandleFunc("/api/localsend/v2/prepare-upload", HandlePrepareUpload)
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
