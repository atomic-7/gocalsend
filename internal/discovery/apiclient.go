package discovery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
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
		slog.Error("error marshalling local node to json", slog.Any("error", err))
		os.Exit(1)
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

// Send a register request to the specified url. Also used to send requests to peers that are unknown
func (regi *Registratinator) registerClient(ctx context.Context, regurl *url.URL) error {

	slog.Debug("registering via api", slog.String("url", regurl.String()), slog.Int("numBytes", len(regi.Payload)))

	req, err := http.NewRequestWithContext(ctx, "POST", regurl.String(), bytes.NewReader(regi.Payload))
	if err != nil {
		slog.Error("failed to create post request", slog.String("url", regurl.String()), slog.String("source", "registratinator"))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Content", "application/json")
	req.Header.Set("User-Agent", "go1.23.0 linux/amd64")
	// req.Close = true
	var resp *http.Response
	if regurl.Scheme == "https" {
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
			slog.Debug("caught expected eof??", slog.Any("error", err))
			return err
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			slog.Debug("caught unexpected eof", slog.Any("error", err))
			return err
		}
	}
	slog.Debug("raw registry response", slog.Any("bytes", resp))
	if resp.ContentLength != 0 {
		slog.Debug("peer text response", slog.String("response", string(body)))
	}
	var peerResponse data.PeerBody
	err = json.Unmarshal(body, &peerResponse)
	if err != nil {
		slog.Error("failed to unmarshal registry response", slog.String("peer", peerResponse.Alias), slog.Any("error", err))
		return err
	}

	slog.Debug("Sent off local node info!")
	return nil
}

// Send a post request to /api/localsend/v2/register with the node data
func (regi *Registratinator) RegisterAt(ctx context.Context, peer *data.PeerInfo) error {

	regURL, err := url.Parse("/api/localsend/v2/register")
	if err != nil {
		slog.Error("unable to parse registry url", slog.String("url", "/api/localsend/v2/register"), slog.Any("error", err))
		os.Exit(1)
	}
	regURL.Host = fmt.Sprintf("%s:%d", peer.IP, peer.Port)
	if peer.Protocol == "http" {
		regURL.Scheme = "http"
	} else {
		regURL.Scheme = "https"
	}

	return regi.registerClient(ctx, regURL)
}

// Falback fallback: try registering by hitting every live ip in the subnet
func (regi *Registratinator) RegisterAtSubnet(ctx context.Context, knownPeers data.PeerTracker) error {
	// not actually trying to reach google, just getting my local ip address
	// This should cause no communication with another server
	// should probably replace this with iterating over the ip adresses bound to the used interface
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		slog.Error("could not dial out to determine local ip address", slog.String("host", "8.8.8.8"), slog.Any("error", err))
		return err
	}
	prefix, err := netip.ParsePrefix(conn.LocalAddr().(*net.UDPAddr).IP.String() + "/24")
	if err != nil {
		slog.Error("error parsing network prefix", slog.Any("error", err))
		return err
	}
	slog.Debug("network prefix", slog.Any("prefix", prefix))
	prefix = prefix.Masked()
	urls := make([]*netip.Addr, 0, 256)
	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		urls = append(urls, &addr)
	}

	regURL, err := url.Parse("/api/localsend/v2/register")
	if err != nil {
		slog.Error("unable to parse registry url", slog.String("url", "/api/localsend/v2/register"), slog.Any("error", err))
	}
	regURL.Scheme = "http"
	// TODO: Optimize with waitgroup, currently each timeout runs out completely before moving on to the next address
	for _, addr := range urls[100:110] {
		regURL.Host = fmt.Sprintf("%s:53317", addr.String())
		err := regi.registerClient(ctx, regURL)
		if err != nil {
			slog.Debug("[subreg] could not reach client", slog.String("target", addr.String()))
		}
		// Add clients that answered here?
	}
	return nil
}
