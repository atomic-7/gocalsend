package server

import (
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

func RunServer(port string, peers *data.PeerMap) {

	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", CreateRegisterHandler(peers))
	// TODO: create route handlers for /api/localsend/v1/info and /api/loclasend/v2/info?fingerprint=asdsadasdas
	mux.HandleFunc("/api/localsend/v1/info", InfoHandler)
	mux.HandleFunc("/api/localsend/v2/info", InfoHandler)
	mux.HandleFunc("/", ReqLogger)

	err := http.ListenAndServe(port, mux)
	log.Fatalf("Error running server at port %s: %s", port, err)
}
