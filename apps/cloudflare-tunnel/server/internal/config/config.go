// Package config resolves runtime paths and flags for the fnOS app.
//
// Precedence: explicit flag > environment variable > built-in default.
// cmd/main passes CT_* variables; the flags mirror them (see CONTRACT.md §3).
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Defaults used when neither flag nor env is present (developer runs).
const (
	DefaultListen      = "0.0.0.0:18960"
	DefaultMetricsPort = 20241
)

// Config is the fully resolved runtime configuration.
type Config struct {
	Listen      string
	ConfigDir   string
	DataDir     string
	LogDir      string
	Cloudflared string
	MetricsPort int
	Demo        bool
	APIBase     string

	// DemoRoot is the temp directory that backs an in-memory demo run. It is
	// empty in production and only informational otherwise.
	DemoRoot string
}

// Env keys understood by the server.
const (
	EnvListen      = "CT_LISTEN"
	EnvConfigDir   = "CT_CONFIG_DIR"
	EnvDataDir     = "CT_DATA_DIR"
	EnvLogDir      = "CT_LOG_DIR"
	EnvCloudflared = "CT_CLOUDFLARED"
	EnvMetricsPort = "CT_METRICS_PORT"
	EnvDemo        = "CT_DEMO"
	EnvAPIBase     = "CF_API_BASE"
)

// Load parses args (without the program name) and merges the environment.
func Load(args []string, getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	base := filepath.Join(os.TempDir(), "cloudflare-tunnel")

	fs := flag.NewFlagSet("cloudflare-tunnel", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	listen := fs.String("listen", envOr(getenv, EnvListen, DefaultListen), "HTTP listen address")
	configDir := fs.String("config-dir", envOr(getenv, EnvConfigDir, filepath.Join(base, "config")), "persistent configuration directory")
	dataDir := fs.String("data-dir", envOr(getenv, EnvDataDir, filepath.Join(base, "data")), "persistent state directory")
	logDir := fs.String("log-dir", envOr(getenv, EnvLogDir, filepath.Join(base, "logs")), "ephemeral log directory")
	cloudflared := fs.String("cloudflared", envOr(getenv, EnvCloudflared, filepath.Join(base, "cloudflared")), "cloudflared binary")
	metricsPort := fs.Int("metrics-port", envIntOr(getenv, EnvMetricsPort, DefaultMetricsPort), "loopback metrics port for cloudflared")
	demo := fs.Bool("demo", envBool(getenv, EnvDemo), "run with seeded demo data, no network and no cloudflared")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	cfg := Config{
		Listen:      strings.TrimSpace(*listen),
		ConfigDir:   filepath.Clean(*configDir),
		DataDir:     filepath.Clean(*dataDir),
		LogDir:      filepath.Clean(*logDir),
		Cloudflared: filepath.Clean(*cloudflared),
		MetricsPort: *metricsPort,
		Demo:        *demo,
		APIBase:     strings.TrimSpace(getenv(EnvAPIBase)),
	}
	if cfg.Listen == "" {
		return Config{}, fmt.Errorf("listen address is empty")
	}
	if cfg.MetricsPort < 1 || cfg.MetricsPort > 65535 {
		return Config{}, fmt.Errorf("metrics port out of range: %d", cfg.MetricsPort)
	}
	return cfg, nil
}

// DemoDirs returns the throwaway directory layout used by -demo.
func DemoDirs(root string) (configDir, dataDir, logDir string) {
	return filepath.Join(root, "config"), filepath.Join(root, "data"), filepath.Join(root, "logs")
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envIntOr(getenv func(string) string, key string, fallback int) int {
	v := strings.TrimSpace(getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(getenv func(string) string, key string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
