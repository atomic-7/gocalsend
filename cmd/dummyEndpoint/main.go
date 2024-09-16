package main

import (
	"encoding/json"
	"github.com/atomic-7/gocalsend/internal/data"
	"io"
	"log"
	"net/http"
)

func Register(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength == 0 {
		log.Printf("Received empty request from %s", r.RemoteAddr)
	}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Error receiving request on the register route: ", err)
	}
	log.Printf("REC: %s", string(buf))

	peer := &data.PeerBody{}
	err = json.Unmarshal(buf, peer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Fatal("Error unmarshalling request body: ", err)
	}
	log.Printf("Register: %v", peer)

	w.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", http.HandlerFunc(Register))
	log.Println("Now listening on port 8123")
	log.Fatal(http.ListenAndServe(":8123", mux))
}
