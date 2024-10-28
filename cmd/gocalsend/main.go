package main

import (
	"context"
	"flag"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"log/slog"
	"net"
	"os"
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

	// TODO: use slog with charms handler
	// slog.SetDefault()
	logOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logHandler := slog.NewTextHandler(os.Stdout, logOpts)
	slog.SetDefault(slog.New(logHandler))

	// TODO: Figure out if it makes more sense to serialize this once or to have it serialized wherever it is needed
	node := &data.PeerInfo{
		Alias:       "Gocalsend",
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "headless",
		Fingerprint: "",
		Port:        port,
		Protocol:    "http",
		Download:    false,
		IP:          nil,
		Announce:    false,
	}

	var tlsInfo *data.TLSPaths

	if useTLS {
		slog.Debug("setting up tls",
			slog.String("dir", credDir),
			slog.String("cert", certName),
			slog.String("key", keyName),
		)
		tlsInfo = data.CreateTLSPaths(credDir, certName, keyName)
		err := encryption.SetupTLSCerts(node.Alias, tlsInfo)
		if err != nil {
			slog.Error("failed to setup tls certificates")
			os.Exit(1)
		}
		fingerprint, err := encryption.GetFingerPrint(tlsInfo)
		if err != nil {
			slog.Error("could not calculate fingerprint, something went wrong during certificate setup")
			os.Exit(1)
		}
		node.Fingerprint = fingerprint
		node.Protocol = "https"
		slog.Debug("finished tls setup", slog.String("fingerprint", node.Fingerprint))
	} else {
		//TODO: generate a random string here
		node.Fingerprint = "nonononono"
	}

	peers := data.NewPeerMap()
	registratinator := discovery.NewRegistratinator(node)

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	// When we multicast first, registry via our http endpoint is fine. Me calling their endpoint results in a crash because the mobile client cannot handle http and https on the same port

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := discovery.AnnounceViaMulticast(node, multicastAddr)
	if err != nil {
		registratinator.RegisterAtSubnet(ctx, peers)
	}
	go server.StartServer(ctx, node, peers, tlsInfo)
	discovery.MonitorMulticast(ctx, multicastAddr, peers, registratinator)
}
