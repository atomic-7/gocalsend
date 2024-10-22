package main

import (
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/uploader"
	"net"
	"os"
)

func main() {

	peer := data.PeerInfo{
		Alias: "Dumb Uploader",
		Version: "2.0",
		DeviceType: "headless",
		Fingerprint: "NONONONONO",
		Download: false,
		Announce: false,
		Port: 53317,
		IP:   net.IPv4(127, 0, 0, 1),
	}

	upl := uploader.CreateUploader(&peer)

	upl.UploadFiles(&peer, os.Args[1:])

}
