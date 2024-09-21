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
	var useTLS bool

	flag.IntVar(&port, "port", 53317, "The port to listen for the api endpoints")
	flag.StringVar(&certName, "cert", "cert.pem", "The filename of the tls certificate")
	flag.StringVar(&keyName, "key", "key.pem", "The filename of the tls private key")
	flag.StringVar(&credDir, "credentials", "./cert", "The path to the tls credentials")
	flag.BoolVar(&useTLS, "usetls", true, "Use https (usetls=true) or use http (usetls=false)")
	flag.Parse()

	// TODO: Read cert, make sha256 and set to fingerprint
	node := &data.PeerInfo{
		Alias:       "Gocalsend",
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "headless",
		Fingerprint: "",
		Port:        port,
		Protocol:    "http", // changing this to https might work to prevent the other client from going with the info route. It does not
		Download:    false,
		IP:          nil,
		Announce:    false,
	}

	var tlsInfo *data.TLSPaths

	if useTLS {
		log.Println("Setting up https")
		tlsInfo = data.CreateTLSPaths(credDir, certName, keyName)
		err := encryption.SetupTLSCerts(node.Alias, tlsInfo)
		if err != nil {
			log.Fatal("Failed to setup tls certificates")
		}
		fingerprint, err := encryption.GetFingerPrint(tlsInfo)
		if err != nil {
			log.Fatal("Could not calculate fingerprint, something went wrong during certificate setup")
		}
		node.Fingerprint = fingerprint
		node.Protocol = "https"
		log.Printf("Calculated fingerprint: %s", node.Fingerprint)
	} else {
		//TODO: generate a random string here
		node.Fingerprint = "nonononono"
	}

	peers := data.NewPeerMap()
	jsonBuf, err := json.Marshal(node.ToPeerBody())
	if err != nil {
		log.Fatal("Error marshalling local node to json: ", err)
	}
	registratinator := discovery.NewRegistratinator(jsonBuf, node.Protocol)

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	// When we multicast first, registry via our http endpoint is fine. Me calling their endpoint results in a crash
	// discovery.AnnounceMulticast(node, multicastAddr)
	log.Println("gocalsending now!")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go server.StartServer(ctx, fmt.Sprintf(":%d", node.Port), peers, tlsInfo)
	discovery.MonitorMulticast(ctx, multicastAddr, peers, registratinator)
}
