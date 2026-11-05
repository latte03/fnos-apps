package demo

import (
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

// Settings returns the seeded demo account configuration.
func Settings() store.Settings {
	return store.Settings{
		AccountID:        AccountID,
		APIToken:         APIToken,
		Protocol:         "quic",
		EdgeIPVersion:    "auto",
		AutoConnect:      true,
		AutoReconnect:    true,
		MaxReconnect:     10,
		LogRetentionDays: 30,
		MaxLogLines:      2000,
	}
}

// State points the boot auto-connect at the healthy demo tunnel.
func State() store.State {
	return store.State{
		TunnelID:    TunnelMainID,
		TunnelName:  "fnos-main",
		AutoConnect: true,
		LastState:   model.StateConnected,
	}
}

// Seed writes settings, one API key and the activity feed into st.
func Seed(st *store.Store) error {
	if err := st.SaveSettings(Settings()); err != nil {
		return err
	}
	if err := st.SaveState(State()); err != nil {
		return err
	}
	if len(st.Keys()) == 0 {
		if _, _, err := st.AddKey("Home Assistant"); err != nil {
			return err
		}
	}
	if len(st.Activity()) == 0 {
		events := []struct{ kind, title, tone string }{
			{"settings", "协议切换为 QUIC", "neutral"},
			{"dns", "CNAME pa.example.com 已创建", "teal"},
			{"route", "路由 ai.example.com 已更新", "blue"},
			{"connect", "隧道 fnos-main 已连接", "green"},
		}
		for _, event := range events {
			if _, err := st.AddActivity(event.kind, event.title, event.tone); err != nil {
				return err
			}
		}
	}
	return nil
}
