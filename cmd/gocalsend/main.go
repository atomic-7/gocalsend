package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/uploader"
	"github.com/charmbracelet/log"
)

func main() {

	var port int
	var certName string
	var keyName string
	var credDir string
	var useTLS bool
	var logLevel string
	var command string
	var peerAlias string
	var lsTime int
	var downloadBase string

	flag.IntVar(&port, "port", 53317, "The port to listen for the api endpoints")
	flag.StringVar(&certName, "cert", "cert.pem", "The filename of the tls certificate")
	flag.StringVar(&keyName, "key", "key.pem", "The filename of the tls private key")
	flag.StringVar(&credDir, "credentials", "./cert", "The path to the tls credentials")
	flag.BoolVar(&useTLS, "usetls", true, "Use https (usetls=true) or use http (usetls=false)")
	flag.StringVar(&logLevel, "loglevel", "info", "Log level can be 'info', 'debug' or 'none'")
	flag.StringVar(&command, "cmd", "receive", "command to execute: rec, receive, snd, send, ls")
	flag.StringVar(&peerAlias, "peer", "", "alias of the peer to send to. find out with gocalsend --cmd=ls")
	flag.IntVar(&lsTime, "lstime", 4, "time to wait for peer discovery")
	flag.StringVar(&downloadBase, "out", "", "path to where incoming files are saved")
	flag.Parse()
	// TODO: implement log level none
	logOpts := log.Options{
		Level: log.DebugLevel,
	}
	switch logLevel {
	case "info":
		logOpts.Level = log.InfoLevel
	case "debug":
		logOpts.Level = log.DebugLevel
		logOpts.ReportCaller = true
	case "none":
		log.Warn("Log level none is not implemented yet")
	case "default":
		logOpts.Level = log.InfoLevel
	}
	charmLogger := log.NewWithOptions(os.Stdout, logOpts)
	slog.SetDefault(slog.New(charmLogger))

	if downloadBase != "" {
		if downloadBase[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				slog.Error("could not get user home directory", slog.Any("error", err))
				os.Exit(1)
			}
			downloadBase = home + downloadBase[1:]
		}
	} else {
		downloadBase, err := os.UserHomeDir()
		if err != nil {
			slog.Error("could not get user home directory", slog.Any("error", err))
			os.Exit(1)
		}
		downloadBase += "/Downloads/gocalsend"
	}

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
	pm := *peers.GetMap()
	pm["self"] = node
	peers.ReleaseMap()

	registratinator := discovery.NewRegistratinator(node)
	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runAnnouncement := func() {
		err := discovery.AnnounceViaMulticast(node, multicastAddr)
		if err != nil {
			registratinator.RegisterAtSubnet(ctx, peers)
		}
	}

	go server.StartServer(ctx, node, peers, tlsInfo, downloadBase)
	go discovery.MonitorMulticast(ctx, multicastAddr, peers, registratinator)
	runAnnouncement()
	switch command {
	case "ls":
		sleepDuration := lsTime * int(time.Second)
		time.Sleep(time.Duration(sleepDuration))
		pm = *peers.GetMap()
		if len(pm) != 1 {
			charmLogger.Printf("Peerlist:")
			for _, peer := range pm {
				if peer.Alias == node.Alias {
					continue
				}
				charmLogger.Printf(peer.Alias)
			}
		} else {
			slog.Info("Found no peers")
		}

	case "snd", "send":
		if peerAlias == "" {
			slog.Error("no peer specified")
			os.Exit(1)
		}
		time.Sleep(3 * time.Second)
		pm = *peers.GetMap()
		var target *data.PeerInfo
		for _, peer := range pm {
			if peer.Alias == peerAlias {
				target = peer
			}
		}
		if target == nil {
			slog.Error("Peer is not available.", slog.String("peer", peerAlias))
			os.Exit(1)
		}
		peers.ReleaseMap()
		slog.Debug("Peer", slog.Any("info", target))
		upl := uploader.CreateUploader(node)
		upl.UploadFiles(target, flag.Args())
	case "rcv", "rec", "recv", "receive":
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		intervalRunner(ctx, runAnnouncement, ticker)
	default:
		slog.Error("unknown command", slog.String("cmd", command))
	}
}

func intervalRunner(ctx context.Context, f func(), ticker *time.Ticker) {
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			f()
		}
	}
}
