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

func ReqLogger(w http.ResponseWriter, r *http.Request) {
	log.Printf("RQ: %v", r)
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	// apparently the localsend implementation expects some response here? https://github.com/localsend/localsend/blob/main/common/lib/src/discovery/http_target_discovery.dart
	r.ParseForm()
	// default response is 200 ok if nothing is written to the response writer
	log.Printf("%s: %v", r.URL, r.Form)
}

func CreateRegisterHandler(peers *data.PeerMap) http.Handler {
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

func StartServer(ctx context.Context, port string, peers *data.PeerMap, tlsInfo *data.TLSPaths) {

	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", CreateRegisterHandler(peers))
	mux.HandleFunc("/api/localsend/v1/info", InfoHandler)
	mux.HandleFunc("/api/localsend/v2/info", InfoHandler)
	mux.HandleFunc("/", ReqLogger)

	var srv http.Server

	// Might have to use InsecureSkipVerify here with a VerifyConnection function to check against the known fingerprints?
	// TODO: Look into VerifyConnection
	if tlsInfo != nil {
		srv = http.Server{
			Addr:    port,
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
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
