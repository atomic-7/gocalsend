package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/sessions"
	"github.com/atomic-7/gocalsend/internal/uploader"
	"github.com/charmbracelet/log"
)

func main() {

	var command string
	var peerAlias string

	// setup logger so config loading can log, reconfigure later
	logOpts := log.Options{
		Level: log.InfoLevel,
	}
	charmLogger := log.NewWithOptions(os.Stdout, logOpts)
	slog.SetDefault(slog.New(charmLogger))

	appConf, err := config.Setup()
	if err != nil {
		slog.Error("failed to setup configuration. exiting.", slog.Any("err", err))
	}

	flag.StringVar(&command, "cmd", "receive", "command to execute: rec, receive, snd, send, ls")
	flag.StringVar(&peerAlias, "peer", "", "alias of the peer to send to. find out with gocalsend --cmd=ls")

	// TODO: implement log level none
	logOpts = log.Options{
		Level: log.DebugLevel,
	}
	switch appConf.LogLevel {
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
	charmLogger = log.NewWithOptions(os.Stdout, logOpts)
	slog.SetDefault(slog.New(charmLogger))

	node := &data.PeerInfo{
		Alias:       appConf.Alias,
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "headless",
		Fingerprint: "",
		Port:        appConf.Port,
		Protocol:    "http",
		Download:    false,
		IP:          nil,
		Announce:    false,
	}

	if appConf.UseTLS {
		slog.Debug("setting up tls",
			slog.String("dir", appConf.TLSInfo.Dir),
			slog.String("cert", appConf.TLSInfo.Cert),
			slog.String("key", appConf.TLSInfo.Key),
		)
		err := encryption.SetupTLSCerts(appConf.Alias, appConf.TLSInfo)
		if err != nil {
			slog.Error("failed to setup tls certificates")
			os.Exit(1)
		}
		fingerprint, err := encryption.GetFingerPrint(appConf.TLSInfo)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hui := sessions.HeadlessUI{}
	sessionManager := sessions.NewSessionManager(ctx, appConf.DownloadFolder, &hui)
	registratinator := discovery.NewRegistratinator(node)
	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	runAnnouncement := func() {
		err := discovery.AnnounceViaMulticast(node, multicastAddr)
		if err != nil {
			registratinator.RegisterAtSubnet(ctx, peers)
		}
	}

	go server.StartServer(ctx, node, peers, sessionManager, appConf.TLSInfo, appConf.DownloadFolder)
	go discovery.MonitorMulticast(ctx, multicastAddr, node, peers, registratinator)
	runAnnouncement()
	switch command {
	case "ls":
		sleepDuration := appConf.PeerDiscoveryTime * int(time.Second)
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
		upl := uploader.CreateUploader(node, sessionManager)
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
