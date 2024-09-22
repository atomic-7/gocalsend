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
	mux.HandleFunc("/", reqLogger)

	var srv http.Server
	port := fmt.Sprintf(":%d", localNode.Port)

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
