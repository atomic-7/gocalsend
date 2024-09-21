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
	"time"

	"github.com/atomic-7/gocalsend/internal/data"
)

type Registratinator struct {
	Protocol string
	Payload []byte
	client *http.Client
}

func NewRegistratinator( nodeJson []byte, protocol string) *Registratinator {
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second), // This timeout leads to a segfault if io.ReadAll(req.body) is running
	}
	if protocol == "https" {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				//TODO: Look into writing a custom verification function to at least check against the fingerprint
				InsecureSkipVerify: true,
			},
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(3 * time.Second),
		}

	} else {
		client.Transport = &http.Transport{
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(3 * time.Second),
		}
	}
	return &Registratinator{
		client: client,
		Protocol: protocol,
		Payload: nodeJson,

	}	
}

// TODO: Add register for peers that indicate http
// Send a post request to /api/localsend/v2/register with the node data
func (regi *Registratinator) RegisterAt(ctx context.Context, peer *data.PeerInfo) error {
	log.Println("Called SendTo")

	url := fmt.Sprintf("%s://%s:%d/api/localsend/v2/register", regi.Protocol, peer.IP, peer.Port)
	log.Printf("Using: %s, sending %d bytes", url, len(regi.Payload))

	req, err := http.NewRequestWithContext(ctx, "post", url, bytes.NewReader(regi.Payload))
	if err != nil {
		log.Fatal("Error creating post request to %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go1.23.0 linux/amd64")
	req.Close = true

	resp, err := regi.client.Do(req)

	if err != nil {
		log.Fatal("Error doing request: ", err)
	}

	// It seems we do not even get any sort of headers, let alone a body as shown by the timout error message
	body, err := io.ReadAll(resp.Body)
	///body := []byte{97}
	log.Println("OKE????")
	if err != nil {
		log.Println("SendTo|ReadAll|", err)
		if err == io.ErrUnexpectedEOF {
			log.Println("This eof was unexpected: %v", err)
		}
		if !errors.Is(err, io.EOF) {
			log.Println("Caught expected EOF??: ", err)
			return err
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println("Caught unexpected EOF: ", err)
			return err
		}
	}
	defer resp.Body.Close()
	if resp.ContentLength != 0 {
		log.Println("Peer %s responds with %s", peer.Alias, string(body))
	}
	log.Println("Huh")

	log.Println("Sent off local node info!")
	return nil
}
