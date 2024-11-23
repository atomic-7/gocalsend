package config

import (
	"errors"
	"github.com/pelletier/go-toml"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/atomic-7/gocalsend/internal/data"
)

type Config struct {
	Version           int
	Port              int
	Alias             string
	DownloadFolder    string
	UseTLS            bool
	LogLevel          string
	PeerDiscoveryTime int `comment:"Time to search for peers when sending"`
	TLSInfo           *data.TLSPaths
}

func Default() (*Config, error) {
	// TODO: get hostname and use it for alias generation
	// TODO: untangle tls paths
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("could not get user home directory", slog.Any("error", err))
		return nil, err
	}
	confdir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("failed to get user config directory", slog.Any("error", err))
		return nil, err
	}
	return &Config{
		Version:           0,
		Alias:          "gocalsend",
		DownloadFolder: filepath.Join(home, "Downloads", "gocalsend"),
		Port:           53317,
		UseTLS:         true,
		PeerDiscoveryTime: 4,
		LogLevel:          "info",
		TLSInfo: &data.TLSPaths{
			Dir: filepath.Join(confdir, "gocalsend"),
		},
	}, nil
}

var (
	IncorrectFileExt = filepath.ErrBadPattern
)

// Load a config file from path.
// An empty path uses os.UserConfigDir() to search for a gocalsend configuration at $UserConfigDir/gocalsend/config.toml
func Load(path string) (*Config, error) {
	if path == "" {
		confdir, err := os.UserConfigDir()
		if err != nil {
			slog.Error("Could not find default config path", slog.Any("error", err))
			return nil, err
		}
		path = filepath.Join(confdir, "gocalsend", "config.toml")
	}
	file, err := os.ReadFile(path)
	if err != nil {
		slog.Error("Could not read config file", slog.Any("error", err))
		return nil, err
	}
	config, err := Default()
	if err != nil {
		return nil, err
	}
	err = toml.Unmarshal(file, config)

	return config, nil
}

// Store a configuration to disk at path.
// An empty path uses os.UserConfigDir() to create a folder for gocalsend configurations
func (c *Config) Store(path string) error {
	logga := slog.Default().With(slog.String("path", path))
	logga.Debug("Storing config")
	bytes, err := toml.Marshal(c)
	if err != nil {
		logga.Error("Failed to marshal config file", slog.Any("error", err))
		return err
	}
	if path == "" {
		confdir, err := os.UserConfigDir()
		if err != nil {
			slog.Error("Could not find default config path", slog.Any("error", err))
			return err
		}
		path = filepath.Join(confdir, "gocalsend", "config.toml")
	}
	if filepath.Ext(path) != ".toml" {
		// TODO return IncorrectFileExt here
		err = errors.New("config file path has incorrect file extension")
		logga.Error("Incorrect config file extension", slog.Any("error", err))
		return err
	}
	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		logga.Error("Failed to create config file path", slog.Any("error", err))
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		logga.Error("Failed to create config file", slog.Any("error", err))
		return err
	}
	_, err = file.Write(bytes)
	if err != nil {
		logga.Error("Failed to write config file to disk", slog.Any("error", err))
		return err
	}
	err = file.Close()
	if err != nil {
		logga.Error("Failed to close config file", slog.Any("error", err))
		return err
	}
	return nil
}
