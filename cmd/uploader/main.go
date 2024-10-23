package main

import (
	"context"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/uploader"
	"log"
	"net"
	"os"
	"time"
)

func main() {

	node := data.PeerInfo{
		Alias:       "Dumb Uploader",
		Version:     "2.0",
		DeviceType:  "headless",
		Fingerprint: "NONONONONO",
		Protocol:    "http",
		Download:    false,
		Announce:    false,
		Port:        53320,
		IP:          net.IPv4(192, 168, 117, 77),
	}

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
		IP:          net.IPv4(192, 168, 117, 77),
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

	time.Sleep(1000)
	upl.UploadFiles(&peer, os.Args[1:])

}
