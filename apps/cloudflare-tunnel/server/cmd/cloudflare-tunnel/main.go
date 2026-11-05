// Command cloudflare-tunnel is the fnOS backend for the Cloudflare Tunnel app:
// it supervises one token based cloudflared connector, manages the tunnel
// through the Cloudflare API and serves the management UI and JSON API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/cloudflare"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/config"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/demo"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/httpapi"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/runtime"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

const (
	appVersion    = "2026.10.7-10"
	demoVersion   = "2026.10.7"
	bootDelay     = 200 * time.Millisecond
	stopTimeout   = 12 * time.Second
	mirrorLogName = "cloudflared.log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetPrefix("cloudflare-tunnel: ")

	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(2)
	}
	if err := run(cfg); err != nil {
		log.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	if cfg.Demo {
		root, err := os.MkdirTemp("", "cloudflare-tunnel-demo-*")
		if err != nil {
			// Some containers mount /tmp read-only; the runtime log directory
			// (@appdata) is always writable on fnOS.
			if mkErr := os.MkdirAll(cfg.LogDir, 0o700); mkErr == nil {
				root, err = os.MkdirTemp(cfg.LogDir, "demo-*")
			}
		}
		if err != nil {
			return fmt.Errorf("create demo directory: %w", err)
		}
		cfg.DemoRoot = root
		cfg.ConfigDir, cfg.DataDir, cfg.LogDir = config.DemoDirs(root)
		log.Printf("demo mode: data under %s (no network, no cloudflared)", root)
	}
	for _, dir := range []string{cfg.ConfigDir, cfg.DataDir, cfg.LogDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	st, err := store.Open(store.Paths{ConfigDir: cfg.ConfigDir, DataDir: cfg.DataDir})
	if err != nil {
		return err
	}
	if cfg.Demo {
		if err := demo.Seed(st); err != nil {
			return fmt.Errorf("seed demo data: %w", err)
		}
	}

	var cloudAPI httpapi.CloudAPI
	if cfg.Demo {
		cloudAPI = demo.NewProvider()
	} else {
		cloudAPI = cloudflare.New(cloudflare.Options{BaseURL: cfg.APIBase})
	}

	version := demoVersion
	if !cfg.Demo {
		version = detectCloudflaredVersion(cfg.Cloudflared)
	}

	srv := httpapi.New(httpapi.Options{
		Version:            appVersion,
		Demo:               cfg.Demo,
		Listen:             cfg.Listen,
		MetricsPort:        cfg.MetricsPort,
		ConfigDir:          cfg.ConfigDir,
		DataDir:            cfg.DataDir,
		LogDir:             cfg.LogDir,
		CloudflaredPath:    cfg.Cloudflared,
		CloudflaredVersion: version,
		Store:              st,
		Cloud:              cloudAPI,
	})

	settings := st.Settings()
	mirror, err := openMirrorLog(cfg.LogDir, settings.LogRetentionDays)
	if err != nil {
		log.Printf("warning: cloudflared log mirror disabled: %v", err)
	}

	supOptions := runtime.Options{
		Binary:             cfg.Cloudflared,
		MetricsPort:        cfg.MetricsPort,
		CloudflaredVersion: version,
		Protocol:           settings.Protocol,
		EdgeIPVersion:      settings.EdgeIPVersion,
		AutoReconnect:      settings.AutoReconnect,
		MaxReconnect:       settings.MaxReconnect,
		MaxLogLines:        settings.MaxLogLines,
		Mirror:             mirror,
	}
	if cfg.Demo {
		supOptions.Factory = demo.Factory{}
		supOptions.Prober = demo.Prober{}
	}
	sup := runtime.New(supOptions, srv)
	srv.SetEngine(sup)
	defer func() {
		if mirror != nil {
			_ = mirror.Close()
		}
	}()

	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Listen, err)
	}
	httpServer := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.Serve(listener) }()
	log.Printf("listening on http://%s (demo=%t, cloudflared=%q)", cfg.Listen, cfg.Demo, version)

	go func() {
		time.Sleep(bootDelay)
		if err := bootConnect(context.Background(), st, sup, cloudAPI); err != nil {
			log.Printf("auto-connect skipped: %v", err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-signalCh:
		log.Printf("received %s, shutting down", sig)
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	if err := sup.Shutdown(ctx); err != nil {
		log.Printf("warning: cloudflared did not stop cleanly: %v", err)
	}
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("warning: http shutdown incomplete: %v", err)
	}
	if cfg.DemoRoot != "" {
		_ = os.RemoveAll(cfg.DemoRoot)
	}
	log.Printf("stopped")
	return nil
}

// bootConnect reconnects the tunnel recorded in state.json when autoConnect is
// on and the app was connected when it stopped.
func bootConnect(ctx context.Context, st *store.Store, sup *runtime.Supervisor, cloud httpapi.CloudAPI) error {
	settings := st.Settings()
	state := st.State()
	if !settings.AutoConnect || !state.AutoConnect {
		return nil
	}
	if state.TunnelID == "" || !model.IsLiveState(state.LastState) {
		return nil
	}
	if !settings.Configured() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	token, err := cloud.TunnelToken(ctx, settings.AccountID, settings.APIToken, state.TunnelID)
	if err != nil {
		return fmt.Errorf("cannot fetch the tunnel token: %w", err)
	}
	log.Printf("auto-connecting tunnel %s", state.TunnelName)
	return sup.Connect(state.TunnelID, state.TunnelName, token)
}

// detectCloudflaredVersion runs `cloudflared --version` once at startup.
func detectCloudflaredVersion(path string) string {
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	for i, field := range fields {
		if field == "version" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// openMirrorLog opens logs/cloudflared.log, dropping it first when it is older
// than the configured retention window.
func openMirrorLog(dir string, retentionDays int) (*os.File, error) {
	if dir == "" {
		return nil, errors.New("no log directory")
	}
	path := filepath.Join(dir, mirrorLogName)
	if info, err := os.Stat(path); err == nil && retentionDays > 0 {
		if time.Since(info.ModTime()) > time.Duration(retentionDays)*24*time.Hour {
			_ = os.Remove(path)
		}
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
}
