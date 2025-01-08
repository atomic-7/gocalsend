package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
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

	configPath := ""
	for idx, arg := range os.Args {
		if arg == "--config" || arg == "-config" {
			if len(os.Args) <= idx+1 {
				slog.Error("You need to specify a config file with the config flag")
				os.Exit(1)
			}
			configPath = os.Args[idx+1]
			break
		}
		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
			configPath = strings.SplitN(arg, "=", 2)[1]
			break
		}
	}
	if configPath != "" {
		slog.Debug("parsed path", slog.Any("path", configPath))
	}

	appConf, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			appConf, err = config.Default()
			if err != nil {
				os.Exit(1)
			}
			slog.Info("No config file found. Generating a new one.")
			appConf.Store(configPath)
		} else {
			os.Exit(1)
		}
	}

	flag.IntVar(&appConf.Port, "port", appConf.Port, "The port to listen for the api endpoints")
	flag.StringVar(&appConf.TLSInfo.Cert, "cert", appConf.TLSInfo.Cert, "The filename of the tls certificate")
	flag.StringVar(&appConf.TLSInfo.Key, "key", appConf.TLSInfo.Key, "The filename of the tls private key")
	flag.StringVar(&appConf.TLSInfo.Dir, "credentials", appConf.TLSInfo.Dir, "The path to the tls credentials")
	flag.BoolVar(&appConf.UseTLS, "usetls", appConf.UseTLS, "Use https (usetls=true) or use http (usetls=false)")
	flag.StringVar(&appConf.LogLevel, "loglevel", appConf.LogLevel, "Log level can be 'info', 'debug' or 'none'")
	flag.StringVar(&command, "cmd", "receive", "command to execute: rec, receive, snd, send, ls")
	flag.StringVar(&peerAlias, "peer", "", "alias of the peer to send to. find out with gocalsend --cmd=ls")
	flag.IntVar(&appConf.PeerDiscoveryTime, "lstime", appConf.PeerDiscoveryTime, "time to wait for peer discovery")
	flag.StringVar(&appConf.DownloadFolder, "out", appConf.DownloadFolder, "path to where incoming files are saved")
	flag.Parse()

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

	// setup the download folder
	if appConf.DownloadFolder != "" {
		if appConf.DownloadFolder[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				slog.Error("could not get user home directory", slog.Any("error", err))
				os.Exit(1)
			}
			appConf.DownloadFolder = filepath.Join(home, appConf.DownloadFolder[1:])
		}
	} else {
		// This might be redundant seeing as this is already part of the default config
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("could not get user home directory", slog.Any("error", err))
			os.Exit(1)
		}
		slog.Debug("user home", slog.String("home", home))
		appConf.DownloadFolder = filepath.Join(home, "Downloads", "gocalsend")
	}
	slog.Info("download folder", slog.String("out", appConf.DownloadFolder))

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

	hui := sessions.HeadlessUI{}
	sessionManager := sessions.NewSessionManager(appConf.DownloadFolder, &hui)
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
