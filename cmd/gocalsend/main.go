package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"log"
	"net"
)

func main() {

	var port int
	var certName string
	var keyName string
	var credDir string

	flag.IntVar(&port, "port", 53317, "The port to listen for the api endpoints")
	flag.StringVar(&certName, "cert", "cert.pem", "The filename of the tls certificate")
	flag.StringVar(&keyName, "key", "key.pem", "The filename of the tls private key")
	flag.StringVar(&credDir, "credentials", "./cert", "The path to the tls credentials")
	flag.Parse()

	tlsInfo := data.CreateTLSPaths(credDir, certName, keyName)

	// TODO: Read cert, make sha256 and set to fingerprint
	node := &data.PeerInfo{
		Alias:       "Gocalsend",
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "server",
		Fingerprint: "",
		Port:        port,
		Protocol:    "https", // changing this to https might work to prevent the other client from going with the info route
		Download:    false,
		IP:          nil,
		Announce:    false,
	}

	err := encryption.SetupTLSCerts(node.Alias, tlsInfo)
	if err != nil {
		log.Fatal("Failed to setup tls certificates")
	}
	fingerprint, err := encryption.GetFingerPrint(tlsInfo)
	if err != nil {
		log.Fatal("Could not calculate fingerprint, something went wrong during certificate setup")
	}
	node.Fingerprint = fingerprint
	log.Printf("Calculated fingerprint: %s", node.Fingerprint)

	peers := data.NewPeerMap()
	jsonBuf, err := json.Marshal(node.ToPeerBody())
	if err != nil {
		log.Fatal("Error marshalling local node to json: ", err)
	}

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	// When we multicast first, registry via our http endpoint is fine. Me calling their endpoint results in a crash
	// AnnounceMulticast(node, multicastAddr)
	log.Println("gocalsending now!")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go server.StartServer(ctx, fmt.Sprintf(":%d", node.Port+1), fmt.Sprintf(":%d", node.Port), peers, tlsInfo)
	discovery.MonitorMulticast(ctx, multicastAddr, peers, jsonBuf)
}
