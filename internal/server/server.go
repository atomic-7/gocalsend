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
	r.ParseForm()
	log.Printf("INFO: %v", r.Form)
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

func StartServer(ctx context.Context, port string, tlsport string, peers *data.PeerMap, tlsInfo *data.TLSPaths) {

	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", CreateRegisterHandler(peers))
	mux.HandleFunc("/api/localsend/v1/info", InfoHandler)
	mux.HandleFunc("/api/localsend/v2/info", InfoHandler)
	mux.HandleFunc("/", ReqLogger)

	var srv http.Server
	srv = http.Server{
		Addr:    port,
		Handler: mux,
	}

	// Might have to use InsecureSkipVerify here with a VerifyConnection function to check against the known fingerprints?
	// TODO: Look into VerifyConnection
	// did not work???
	var tlsrv http.Server
	if tlsInfo != nil {
		tlsrv = http.Server{
			Addr:    tlsport,
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				// InsecureSkipVerify: true,
			},
		}
	}

	errChan := make(chan error)

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			errChan <- err
		}
	}()
	go func() {
		err := tlsrv.ListenAndServeTLS(tlsInfo.CertPath, tlsInfo.KeyPath)
		if err != nil {
			errChan <- err
		}
	}()

	for {
		select {
		case <-ctx.Done():
			srv.Shutdown(ctx)
			tlsrv.Shutdown(ctx)
		case err := <-errChan:
			srv.Shutdown(ctx)
			tlsrv.Shutdown(ctx)
			log.Fatal("Error running api: ", err)
		}

	}

}
