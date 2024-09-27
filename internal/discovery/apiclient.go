package discovery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/atomic-7/gocalsend/internal/data"
)

type Registratinator struct {
	Protocol  string
	Payload   []byte
	client    *http.Client
	tlsClient *http.Client
}

func NewRegistratinator(localNode *data.PeerInfo) *Registratinator {
	jsonBuf, err := json.Marshal(localNode.ToPeerBody())
	if err != nil {
		log.Fatal("Error marshalling local node to json: ", err)
	}
	client := &http.Client{
		Transport: &http.Transport{
			// DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(2 * time.Second),
		},
		Timeout: time.Duration(2 * time.Second), // This timeout leads to a segfault if io.ReadAll(req.body) is running
	}
	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//TODO: Look into writing a custom verification function to at least check against the fingerprint
				InsecureSkipVerify: true,
			},
			// DisableKeepAlives:     true,
			ResponseHeaderTimeout: time.Duration(2 * time.Second),
		},
		Timeout: time.Duration(2 * time.Second),
	}
	return &Registratinator{
		client:    client,
		tlsClient: tlsClient,
		Payload:   jsonBuf,
	}
}

func (regi *Registratinator) registerClient(ctx context.Context, url string) error {

	log.Printf("Using: %s, sending %d bytes", url, len(regi.Payload))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(regi.Payload))
	if err != nil {
		log.Fatal("Error creating post request to %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Content", "application/json")
	req.Header.Set("User-Agent", "go1.23.0 linux/amd64")
	req.Close = true
	var resp *http.Response
	if strings.HasPrefix(url, "https") {
		resp, err = regi.tlsClient.Do(req)
	} else {
		resp, err = regi.client.Do(req)
	}

	if err != nil {
		return err
	}

	// The desktop clients log claims the mobile clients register route fails when answering to the phones multicast and falls back to multicast after the register route fails
	// It seems we do not even get any sort of headers, let alone a body as shown by the timout error message
	// The mobile client answers the curl call with the correct response though
	// This works even with encryption on at the peer
	//curl --json '{"alias":"Gocalsend","version":"2.0","deviceModel":"cli","deviceType":"headless","fingerprint":"3d7b158a3f1279bab4c1926b1375bfbd05af954dbaaef7e4ff3ead226dbe9288","port":53320,"protocol":"https","download":false}' --insecure https://192.168.117.39:53317/api/localsend/v2/register
	// Hititng the register endpoint of the desktop causes it to have an unhandled exception
	// when done via gocalsend, however it still seems to answer
	// 	ERROR:flutter/runtime/dart_vm_initializer.cc(41)] Unhandled Exception: Null check operator used on a null value
	// #0      ConnectivityPlusLinuxPlugin._startListenConnectivity (package:connectivity_plus/src/connectivity_plus_linux.dart:68)
	// <asynchronous suspension>
	// TODO: Compare desktop client behaviour with phone client for unencrypted setups

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Println("Caught expected EOF??: ", err)
			return err
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println("Caught unexpected EOF: ", err)
			return err
		}
	}
	log.Printf("Response: %v", resp)
	if resp.ContentLength != 0 {
		log.Printf("Peer responds with %s", string(body))
	}
	var peerResponse data.PeerBody
	err = json.Unmarshal(body, &peerResponse)
	if err != nil {
		log.Printf("Error unmarshalling response from %s: %v", peerResponse.Alias, err)
		return err
	}

	log.Println("Sent off local node info!")
	return nil
}
// Send a post request to /api/localsend/v2/register with the node data
func (regi *Registratinator) RegisterAt(ctx context.Context, peer *data.PeerInfo) error {

	// TODO: Verify that peer.Protocol is not a malicious string
	url := fmt.Sprintf("%s://%s:%d/api/localsend/v2/register", peer.Protocol, peer.IP, peer.Port)
	return regi.registerClient(ctx, url)
}

// Falback fallback: try registering by hitting every live ip in the subnet
func (regi *Registratinator) RegisterAtSubnet(ctx context.Context, knownPeers *data.PeerMap) error {
	// not actually trying to reach google, just getting my local ip address
	// This should cause no communication with another server
	// should probably replace this with iterating over the ip adresses bound to the used interface
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("Could not dial 8.8.8.8 to determine local ip addr: %v", err)
		return err
	}
	prefix, err := netip.ParsePrefix(conn.LocalAddr().(*net.UDPAddr).IP.String() + "/24")
	if err != nil {
		log.Printf("Error parsing prefix: %v", err)
		return err
	}
	log.Printf("Prefix: %v\n", prefix)
	prefix = prefix.Masked()
	urls := make([]*netip.Addr, 0, 256)
	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		urls = append(urls, &addr)
	}

	// TODO: Optimize with waitgroup, currently each timeout runs out completely before moving on to the next address
	for _, addr := range(urls[100:110]) {
		err := regi.registerClient(ctx, fmt.Sprintf("http://%s:53317/api/localsend/v2/register", addr.String()))
		if err != nil {
			log.Printf("[SUBREG] Could not reach %s\n", addr.String())	
		}
	}
	return nil
}
