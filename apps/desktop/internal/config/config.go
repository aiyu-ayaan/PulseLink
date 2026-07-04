// Package config loads and persists the backend's configuration.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config holds all runtime settings. It is serialised to a JSON file so the
// future settings UI can read and write the same document.
type Config struct {
	// Server is the WebSocket/HTTP listen configuration.
	Server ServerConfig `json:"server"`
	// Database is the SQLite file path.
	DatabasePath string `json:"databasePath"`
	// LogLevel is one of debug, info, warn, error.
	LogLevel string `json:"logLevel"`
	// DeviceName is how this PC advertises itself for discovery.
	DeviceName string `json:"deviceName"`
	// Permissions controls which features the mobile device is allowed to access.
	Permissions FeaturePermissions `json:"permissions"`
	// MaxCompatibilityMode uses software gamma adjustment for brightness controls.
	MaxCompatibilityMode bool `json:"maxCompatibilityMode"`
}

// FeaturePermissions controls which features the mobile device is allowed to access.
type FeaturePermissions struct {
	Media         bool `json:"media"`
	Volume        bool `json:"volume"`
	Brightness    bool `json:"brightness"`
	Clipboard     bool `json:"clipboard"`
	Notifications bool `json:"notifications"`
	Apps          bool `json:"apps"`
	Power         bool `json:"power"`
	SysInfo       bool `json:"sysinfo"`
	Input         bool `json:"input"`
	FileTransfer  bool `json:"filetransfer"`
}

// ServerConfig configures the network listener.
type ServerConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	EnableTLS bool   `json:"enableTls"`
	// CertFile/KeyFile are used when EnableTLS is true. If both are empty a
	// self-signed certificate is generated at startup.
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

// Default returns the built-in configuration used on first run.
func Default() Config {
	host, _ := os.Hostname()
	if host == "" {
		host = "PulseLink-PC"
	}
	return Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9843,
			// ponytail: MVP is plain ws:// — the Android client and the
			// http-served desktop UI both connect with ws, and a self-signed
			// cert can't be trusted by the phone without pairing/cert-trust
			// wiring. Flip to true (self-signed cert auto-generates) once that
			// lands. The TLS code path already exists in transport.Server.
			EnableTLS: false,
		},
		DatabasePath: filepath.Join(DataDir(), "pulselink.db"),
		LogLevel:     "info",
		DeviceName:   host,
		Permissions: FeaturePermissions{
			Media:         true,
			Volume:        true,
			Brightness:    true,
			Clipboard:     true,
			Notifications: true,
			Apps:          true,
			Power:         true,
			SysInfo:       true,
			Input:         true,
			FileTransfer:  true,
		},
		MaxCompatibilityMode: false,
	}
}

// DataDir is the per-user directory where PulseLink stores its data.
func DataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base, _ = os.Getwd()
	}
	return filepath.Join(base, "PulseLink")
}

// Load reads config from path. If the file does not exist it writes and returns
// the defaults, so the first run bootstraps a config file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		return cfg, Save(path, cfg)
	}
	if err != nil {
		return Config{}, err
	}
	cfg := Default() // defaults fill any missing fields
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes cfg to path as indented JSON, creating parent directories.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
