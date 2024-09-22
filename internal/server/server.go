package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
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

func createRegisterHandler(peers *data.PeerMap) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		log.Println("Incoming registry via api")
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading register body: ", err)
		}
		var peer data.PeerInfo
		json.Unmarshal(buf, &peer)
		log.Printf("Registering %s via api register route", peer.Alias) // remember to lock the mutex
		pm := *peers.GetMap()
		defer peers.ReleaseMap()
		if _, ok := pm[peer.Fingerprint]; ok {
			log.Printf("%s was already a known peer", peer.Alias)
		} else {
			pm[peer.Fingerprint] = &peer
		}
	})
}

func StartServer(ctx context.Context, port string, peers *data.PeerMap, tlsInfo *data.TLSPaths, nodeJson []byte) {

	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}

	infoHandler := createInfoHandler(nodeJson)
	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", createRegisterHandler(peers))
	mux.Handle("/api/localsend/v1/info", infoHandler)
	mux.Handle("/api/localsend/v2/info", infoHandler)
	mux.HandleFunc("/", reqLogger)

	var srv http.Server

	// Might have to use InsecureSkipVerify here with a VerifyConnection function to check against the known fingerprints?
	// TODO: Look into VerifyConnection
	if tlsInfo != nil {
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
