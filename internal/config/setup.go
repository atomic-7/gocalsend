package config

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func Setup() (*Config, error) {
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

	// TODO: Expand ~ to home dir here
	appConf, err := Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			appConf, err = Default()
			if err != nil {
				os.Exit(1)
			}
			slog.Info("No config file found. Generating a new one.")
			appConf.Store(configPath)
		} else {
			os.Exit(1)
		}
	}

	// these are only here for the cli. maybe this can be passed on more elegantly
	cmd := "recv"
	peer := ""

	flag.StringVar(&cmd, "cmd", cmd, "The command to execute. (recv, send, ls)")
	flag.StringVar(&cmd, "peer", peer, "Peer to send to. Find available with '--cmd=ls'")
	flag.IntVar(&appConf.Port, "port", appConf.Port, "The port to listen for the api endpoints")
	flag.StringVar(&appConf.TLSInfo.Cert, "cert", appConf.TLSInfo.Cert, "The filename of the tls certificate")
	flag.StringVar(&appConf.TLSInfo.Key, "key", appConf.TLSInfo.Key, "The filename of the tls private key")
	flag.StringVar(&appConf.TLSInfo.Dir, "credentials", appConf.TLSInfo.Dir, "The path to the tls credentials")
	flag.BoolVar(&appConf.UseTLS, "usetls", appConf.UseTLS, "Use https (usetls=true) or use http (usetls=false)")
	flag.StringVar(&appConf.LogLevel, "loglevel", appConf.LogLevel, "Log level can be 'info', 'debug' or 'none'")
	flag.IntVar(&appConf.PeerDiscoveryTime, "lstime", appConf.PeerDiscoveryTime, "time to wait for peer discovery")
	flag.StringVar(&appConf.DownloadFolder, "out", appConf.DownloadFolder, "path to where incoming files are saved")
	flag.StringVar(&configPath, "config", configPath, "Path to the config.toml file")
	flag.Parse()

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

	if cmd == "none" {
		appConf.Mode = AppMode(TUI)
	} else {
		appConf.Mode = AppMode(CLI)
	}
	appConf.CliArgs["cmd"] = cmd
	appConf.CliArgs["peer"] = peer

	return appConf, nil
}
