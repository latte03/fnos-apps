package httpapi

import (
	"net/http"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

type cloudflaredInfo struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
}

type healthResponse struct {
	Status      string          `json:"status"`
	Version     string          `json:"version"`
	Demo        bool            `json:"demo"`
	StartedAt   string          `json:"startedAt"`
	Cloudflared cloudflaredInfo `json:"cloudflared"`
}

// handleHealth is the only public endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Version:   s.opts.Version,
		Demo:      s.opts.Demo,
		StartedAt: model.FormatTS(s.startedAt),
		Cloudflared: cloudflaredInfo{
			Installed: s.cloudflaredInstalled(),
			Version:   s.opts.CloudflaredVersion,
		},
	})
}

type metaResponse struct {
	Version            string `json:"version"`
	Demo               bool   `json:"demo"`
	DataDir            string `json:"dataDir"`
	ConfigDir          string `json:"configDir"`
	LogDir             string `json:"logDir"`
	Listen             string `json:"listen"`
	MetricsPort        int    `json:"metricsPort"`
	CloudflaredVersion string `json:"cloudflaredVersion"`
}

func (s *Server) handleMeta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, metaResponse{
		Version:            s.opts.Version,
		Demo:               s.opts.Demo,
		DataDir:            s.opts.DataDir,
		ConfigDir:          s.opts.ConfigDir,
		LogDir:             s.opts.LogDir,
		Listen:             s.opts.Listen,
		MetricsPort:        s.opts.MetricsPort,
		CloudflaredVersion: s.opts.CloudflaredVersion,
	})
}

type accountResponse struct {
	Configured  bool   `json:"configured"`
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	Zones       int    `json:"zones"`
}

func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	current := s.store.Settings()
	resp := accountResponse{Configured: current.Configured(), AccountID: current.AccountID}
	if resp.Configured {
		resp.AccountName = s.cloud.AccountName(r.Context(), current.AccountID, current.APIToken)
		if zones, err := s.cloud.ListZones(r.Context(), current.APIToken); err == nil {
			resp.Zones = len(zones)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

type activityResponse struct {
	Items []model.Activity `json:"items"`
}

func (s *Server) handleActivity(w http.ResponseWriter, _ *http.Request) {
	items := s.store.Activity()
	if items == nil {
		items = []model.Activity{}
	}
	writeJSON(w, http.StatusOK, activityResponse{Items: items})
}
