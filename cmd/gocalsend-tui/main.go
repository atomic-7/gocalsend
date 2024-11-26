package main

import (
	"context"
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
	"github.com/atomic-7/gocalsend/internal/tui"
)

func main() {

	logfile, err := os.Create("debug.log")
	if err != nil {
		os.Exit(1)
	}
	logOpts := log.Options{
		Level: log.InfoLevel,
	}
	charmLogger := log.NewWithOptions(logfile, logOpts)
	slog.SetDefault(slog.New(charmLogger))

	appConf, err := config.Setup()

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
	charmLogger = log.NewWithOptions(logfile, logOpts)
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

	registratinator := discovery.NewRegistratinator(node)
	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	model := tui.NewModel(appConf)
	model.Context = ctx
	model.CancelF = cancel
	p := tea.NewProgram(tui.NewModel(appConf))
	peers := tui.NewPeerMap(p)

	runAnnouncement := func() {
		err := discovery.AnnounceViaMulticast(node, multicastAddr)
		if err != nil {
			registratinator.RegisterAtSubnet(ctx, peers)
		}
	}
	go server.StartServer(ctx, node, peers, appConf.TLSInfo, appConf.DownloadFolder)
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
