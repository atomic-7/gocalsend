package discovery

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/atomic-7/gocalsend/internal/data"
)

// Send a post request to /api/localsend/v2/register with the node data
func SendTo(ctx context.Context, peer *data.PeerInfo, nodeJson []byte) error {
	log.Println("Called SendTo")

	url := fmt.Sprintf("https://%s:%d/api/localsend/v2/register", peer.IP, peer.Port)
	log.Printf("Using: %s with %s", url, string(nodeJson))

	// TODO: make this a req with context
	req, err := http.NewRequest("post", url, bytes.NewReader(nodeJson))
	if err != nil {
		log.Fatal("Error creating post request to %s", url)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "go1.23.0 linux/amd64")
	req.Close = true
	// Try using a custom client with InsecureSkipVerify: true in tlsConfig
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//TODO: Look into writing a custom verification function to at least check against the fingerprint
				InsecureSkipVerify: true,
			},
		},
	}
	// resp, err := http.DefaultClient.Do(req)
	//TODO: Reuse Client
	resp, err := client.Do(req)

	body, err := io.ReadAll(resp.Body)
	if !errors.Is(err, io.EOF) {
		return err
	}

	log.Println("%s responds with %s", peer.Alias, body)
	defer resp.Body.Close()


	// resp, err := http.Post(url, "application/json", bytes.NewReader(nodeJson))
	if err == io.ErrUnexpectedEOF {
		log.Println("This eof was unexpected: %v", err)
	}
	if err != nil {
		if errors.Is(err, io.EOF) {
			// Sending node info to peer failed: Post "http://192.168.117.39:53317/api/localsend/v2/register": EOF
			log.Println("How the hell did we get here?")
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println("Caught unexpected EOF: ", err)
		}
		return err
	}
	// don't know if the response is  going to be interesting
	log.Printf("Response to post req should be %d bytes", resp.ContentLength)
	if resp.ContentLength != 0 {

	}
	log.Println("Sent off local node info!")
	return nil
}
