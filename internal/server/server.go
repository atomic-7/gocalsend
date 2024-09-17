package server

import (
	"crypto/tls"
	"encoding/json"
	"github.com/atomic-7/gocalsend/internal/data"
	"io"
	"log"
	"net/http"
)

func ReqLogger(w http.ResponseWriter, r *http.Request) {
	log.Printf("RQ: %v", r)
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Printf("INFO: %v", r.Form)
}

func CreateRegisterHandler(peers *data.PeerMap) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		log.Println("Incoming registry via http")
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading register body: ", err)
		}
		var peer data.PeerInfo
		json.Unmarshal(buf, &peer)
		log.Printf("Registering %s via http register route", peer.Alias) // remember to lock the mutex
		peers.LockMap()
		defer peers.UnlockMap()
		if _, ok := peers.Map[peer.Fingerprint]; ok {
			log.Printf("%s was already a known peer", peer.Alias)
		} else {
			peers.Map[peer.Fingerprint] = &peer
		}
	})
}

func RunServer(port string, peers *data.PeerMap, tlsInfo *data.TLSPaths) {

	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", CreateRegisterHandler(peers))
	mux.HandleFunc("/api/localsend/v1/info", InfoHandler)
	mux.HandleFunc("/api/localsend/v2/info", InfoHandler)
	mux.HandleFunc("/", ReqLogger)

	// TODO: Setup a seperate mux and server for the register endpoint that remains http in any case
	tlsInfo = nil
	var srv http.Server
	var err error
	if tlsInfo != nil {
		srv = http.Server{
			Addr: port,
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		err = srv.ListenAndServeTLS(tlsInfo.Cert, tlsInfo.PrivateKey)
	} else {
		srv = http.Server{
			Addr:    port,
			Handler: mux,
		}
		err = srv.ListenAndServe()
	}
	log.Fatal("Error runinng rest endpoints: ", err)
}
