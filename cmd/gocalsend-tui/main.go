package main

import (
	"log/slog"
	"os"

	// tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"

	"github.com/atomic-7/gocalsend/internal/config"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/encryption"
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

	peers := data.NewPeerMap()
	pm := *peers.GetMap()
	pm["self"] = node
	peers.ReleaseMap()
}
