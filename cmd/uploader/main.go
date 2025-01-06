package main

import (
	"context"
	"flag"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/uploader"
	"log/slog"
	"net"
	"os"
	"time"
)

func main() {

	node := data.PeerInfo{
		Alias:       "Dumb Uploader",
		Version:     "2.0",
		DeviceType:  "headless",
		DeviceModel: "cli",
		Fingerprint: "NONONONONO",
		Protocol:    "http",
		Download:    false,
		Announce:    false,
		Port:        53320,
		IP:          net.IPv4(192, 168, 117, 77),
	}

	var peerIP int
	var useTLS bool
	flag.IntVar(&peerIP, "peer", 77, "Peer in the 192.168.117.255/24 subnet")
	flag.BoolVar(&useTLS, "usetls", true, "encrypt connection with tls")
	flag.Parse()

	// TODO: make this part of main localsend
	// TODO: Figure out why tls with the reference localsend implemetations works
	// but not between my self written instances. Could be related to how certs are handled

	peer := data.PeerInfo{
		Alias:       "Smart Cookie",
		Version:     "2.0",
		DeviceType:  "Smartphone",
		Fingerprint: "no idea",
		Protocol:    "http",
		Download:    false,
		Announce:    false,
		Port:        53317,
		//IP:          net.IPv4(127, 0, 0, 1),
		IP: net.IPv4(192, 168, 117, byte(peerIP)),
	}
	credDir := "./cert"
	certName := "cert.pem"
	keyName := "key.pem"
	var tlsInfo *data.TLSPaths
	if useTLS {
		slog.Info("Setting up https")
		node.Protocol = "https"
		peer.Protocol = "https"
		tlsInfo = data.CreateTLSPaths(credDir, certName, keyName)
		err := encryption.SetupTLSCerts(node.Alias, tlsInfo)
		if err != nil {
			slog.Error("Failed to setup tls certificates")
			os.Exit(1)
		}
		fingerprint, err := encryption.GetFingerPrint(tlsInfo)
		if err != nil {
			slog.Error("Could not calculate fingerprint, something went wrong during certificate setup")
			os.Exit(1)
		}
		node.Fingerprint = fingerprint
		node.Protocol = "https"
		slog.Info("Calculated fingerprint", slog.String("fingerprint", node.Fingerprint))
	} else {
		//TODO: generate a random one here
		node.Fingerprint = "NONONONONO"
	}

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	peers := data.NewPeerMap()
	registratinator := discovery.NewRegistratinator(&node)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := discovery.AnnounceViaMulticast(&node, multicastAddr)
	if err != nil {
		slog.Error("Could not announce via Multicast")
		os.Exit(1)
	}
	outFolder, err := os.UserHomeDir()
	if err != nil {
		slog.Error("could not find user home directory", slog.Any("error", err))
	}
	hui := server.HeadlessUI{}
	sessionManager := server.NewSessionManager(outFolder, &hui)
	go server.StartServer(ctx, &node, peers, sessionManager, tlsInfo, outFolder)
	go discovery.MonitorMulticast(ctx, multicastAddr, &node, peers, registratinator)

	uplman := server.NewSessionManager(outFolder, &hui)
	upl := uploader.CreateUploader(&node, uplman)

	time.Sleep(5000)
	upl.UploadFiles(&peer, flag.Args())

}
