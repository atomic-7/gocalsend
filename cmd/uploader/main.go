package main

import (
	"context"
	"flag"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/uploader"
	"log"
	"net"
	"time"
)

func main() {

	node := data.PeerInfo{
		Alias:       "Dumb Uploader",
		Version:     "2.0",
		DeviceType:  "headless",
		DeviceModel: "cli",
		Fingerprint: "NONONONONO",
		Protocol:    "https",
		Download:    false,
		Announce:    false,
		Port:        53320,
		IP:          net.IPv4(192, 168, 117, 77),
	}

	var peerIP int
	flag.IntVar(&peerIP, "peer", 77, "Peer in the 192.168.117.255/24 subnet")
	flag.Parse()

	// TODO: make this part of main localsend
	// TODO: Figure out why tls with the reference localsend implemetations works
	// but not between my self written instances. Could be related to how certs are handled

	peer := data.PeerInfo{
		Alias:       "Smart Cookie",
		Version:     "2.0",
		DeviceType:  "Smartphone",
		Fingerprint: "no idea",
		Protocol:    "https",
		Download:    false,
		Announce:    false,
		Port:        53317,
		//IP:          net.IPv4(127, 0, 0, 1),
		IP: net.IPv4(192, 168, 117, byte(peerIP)),
	}

	credDir := "./cert"
	certName := "cert.pem"
	keyName := "key.pem"
	useTLS := true
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
		node.Fingerprint = "NONONONONO"
	}

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	peers := data.NewPeerMap()
	registratinator := discovery.NewRegistratinator(&node)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := discovery.AnnounceViaMulticast(&node, multicastAddr)
	if err != nil {
		log.Fatal("Could not announce via Multicast")
	}
	go server.StartServer(ctx, &node, peers, nil)
	go discovery.MonitorMulticast(ctx, multicastAddr, peers, registratinator)

	upl := uploader.CreateUploader(&node)

	time.Sleep(5000)
	upl.UploadFiles(&peer, flag.Args())

}
