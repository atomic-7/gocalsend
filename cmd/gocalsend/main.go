package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/discovery"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"github.com/atomic-7/gocalsend/internal/sessions"
	"github.com/atomic-7/gocalsend/internal/tui"
	"github.com/atomic-7/gocalsend/internal/tui/hooks"
	"github.com/atomic-7/gocalsend/internal/uploader"
)

func main() {

	logOpts := log.Options{
		Level: log.DebugLevel,
	}
	charmLogger := log.NewWithOptions(os.Stderr, logOpts)
	slog.SetDefault(slog.New(charmLogger))

	appConf, err := config.Setup()
	if err != nil {
		slog.Error("error setting up config", slog.Any("err", err))
	}

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

	if appConf.Mode == config.AppMode(config.TUI) {
		logfile, err := os.Create("debug.log")
		if err != nil {
			os.Exit(1)
		}
		charmLogger = log.NewWithOptions(logfile, logOpts)
	} else {
		charmLogger = log.NewWithOptions(os.Stderr, logOpts)
	}
	slog.SetDefault(slog.New(charmLogger))

	node := &data.PeerInfo{
		Alias:       appConf.Alias,
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "headless", // TODO: decide depending on mode
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

	registratinator := discovery.NewRegistratinator(node)
	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Debug("config", slog.Int("mode", int(appConf.Mode)))
	var peers data.PeerTracker
	var eventHooks sessions.UIHooks
	if appConf.Mode == config.AppMode(config.TUI) {

		model := tui.NewModel(ctx, node, appConf)
		p := tea.NewProgram(&model, tea.WithAltScreen())
		peers = hooks.NewPeerMap(p)
		runAnnouncement := announcer(ctx, node, multicastAddr, peers, registratinator)
		eventHooks = hooks.NewHooks(p)
		sessionManager := sessions.NewSessionManager(ctx, appConf.DownloadFolder, eventHooks)
		model.Uploader = uploader.CreateUploader(node, sessionManager)
		// dlManager := sessions.NewSessionManager(appConf.DownloadFolder, uihooks)
		model.SetupSessionManagers(sessionManager)
		go server.StartServer(ctx, node, peers, sessionManager, appConf.TLSInfo, appConf.DownloadFolder)
		go discovery.MonitorMulticast(ctx, multicastAddr, node, peers, registratinator)
		runAnnouncement()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		go intervalRunner(ctx, runAnnouncement, ticker)
		slog.Info("starting tea program")
		if _, err := p.Run(); err != nil {
			slog.Error("Error runnnig bubble program", slog.Any("error", err))
			os.Exit(1)
		}

	} else if appConf.Mode == config.AppMode(config.CLI) {

		peerMap := data.NewPeerMap()
		pm := *peerMap.GetMap()
		pm["self"] = node // TODO: check if this is still necessary
		peerMap.ReleaseMap()
		peers = peerMap
		runAnnouncement := announcer(ctx, node, multicastAddr, peers, registratinator)
		eventHooks = &sessions.HeadlessUI{}
		sessionManager := sessions.NewSessionManager(ctx, appConf.DownloadFolder, eventHooks)

		go server.StartServer(ctx, node, peers, sessionManager, appConf.TLSInfo, appConf.DownloadFolder)
		go discovery.MonitorMulticast(ctx, multicastAddr, node, peers, registratinator)
		runAnnouncement()
		switch appConf.CliArgs["cmd"] {
		case "ls":
			sleepDuration := appConf.PeerDiscoveryTime * int(time.Second)
			time.Sleep(time.Duration(sleepDuration))
			pm = *peerMap.GetMap()
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
			if appConf.CliArgs["peer"] == "" {
				slog.Error("no peer specified")
				os.Exit(1)
			}
			time.Sleep(3 * time.Second)
			pm = *peerMap.GetMap()
			var target *data.PeerInfo
			for _, peer := range pm {
				if peer.Alias == appConf.CliArgs["peer"] {
					target = peer
				}
			}
			if target == nil {
				slog.Error("Peer is not available.", slog.String("peer", appConf.CliArgs["peer"]))
				os.Exit(1)
			}
			peerMap.ReleaseMap()
			slog.Debug("Peer", slog.Any("info", target))
			upl := uploader.CreateUploader(node, sessionManager)
			// passing the args will only work while cmd is passed as --cmd
			// this will need to be changed when the command will be passed directly
			upl.UploadFiles(target, flag.Args())

		case "rcv", "rec", "recv", "receive":
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			intervalRunner(ctx, runAnnouncement, ticker)

		default:
			slog.Error("unknown command", slog.String("cmd", appConf.CliArgs["cmd"]))
		}

	} else {
		slog.Error("unknown mode")
		os.Exit(1)
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

func announcer(ctx context.Context, node *data.PeerInfo, multicastAddr *net.UDPAddr, peers data.PeerTracker, registratinator *discovery.Registratinator) func() {
	return func() {
		err := discovery.AnnounceViaMulticast(node, multicastAddr)
		if err != nil {
			registratinator.RegisterAtSubnet(ctx, peers)
		}
	}
}
