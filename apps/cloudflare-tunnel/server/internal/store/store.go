// Package store persists settings, API keys, runtime state and the activity
// feed under the fnOS data share. Secrets live only in 0600 files and are never
// returned to callers.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/model"
)

// File names inside the config/data directories.
const (
	SettingsFile = "settings.json"
	KeysFile     = "apikeys.json"
	StateFile    = "state.json"
	ActivityFile = "activity.json"
)

// MaxActivity bounds activity.json.
const MaxActivity = 100

// Settings defaults (CONTRACT.md §6).
const (
	DefaultProtocol         = "quic"
	DefaultEdgeIPVersion    = "auto"
	DefaultMaxReconnect     = 10
	DefaultLogRetentionDays = 30
	DefaultMaxLogLines      = 2000
)

// Settings is the on-disk account configuration.
type Settings struct {
	AccountID        string `json:"accountId"`
	APIToken         string `json:"apiToken"`
	Protocol         string `json:"protocol"`
	EdgeIPVersion    string `json:"edgeIpVersion"`
	AutoConnect      bool   `json:"autoConnect"`
	AutoReconnect    bool   `json:"autoReconnect"`
	MaxReconnect     int    `json:"maxReconnect"`
	LogRetentionDays int    `json:"logRetentionDays"`
	MaxLogLines      int    `json:"maxLogLines"`
}

// DefaultSettings returns the values used for a fresh install.
func DefaultSettings() Settings {
	return Settings{
		Protocol:         DefaultProtocol,
		EdgeIPVersion:    DefaultEdgeIPVersion,
		AutoConnect:      true,
		AutoReconnect:    true,
		MaxReconnect:     DefaultMaxReconnect,
		LogRetentionDays: DefaultLogRetentionDays,
		MaxLogLines:      DefaultMaxLogLines,
	}
}

// Configured reports whether an account id and token are present.
func (s Settings) Configured() bool {
	return s.AccountID != "" && s.APIToken != ""
}

// APIKey is a stored client key. Only the SHA-256 hash is persisted.
type APIKey struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	Hash       string `json:"hash"`
	CreatedAt  string `json:"createdAt"`
	LastUsedAt string `json:"lastUsedAt"`
}

// State is the boot reconnect hint persisted in data/state.json.
type State struct {
	TunnelID    string `json:"tunnelId"`
	TunnelName  string `json:"tunnelName"`
	AutoConnect bool   `json:"autoConnect"`
	LastState   string `json:"lastState"`
	UpdatedAt   string `json:"updatedAt"`
}

// Paths locates the two persistent directories.
type Paths struct {
	ConfigDir string
	DataDir   string
}

// Store is the thread safe persistence facade.
type Store struct {
	mu    sync.RWMutex
	paths Paths

	settings Settings
	keys     []APIKey
	state    State
	activity []model.Activity
}

type settingsFile struct {
	Version  int      `json:"version"`
	Settings Settings `json:"settings"`
}

type keysFile struct {
	Version int      `json:"version"`
	Keys    []APIKey `json:"keys"`
}

type activityFile struct {
	Version int              `json:"version"`
	Items   []model.Activity `json:"items"`
}

// Open loads every file, creating the directories when missing. Unreadable or
// corrupt files fall back to defaults instead of blocking startup; the broken
// file is kept next to the original with a .corrupt suffix.
func Open(paths Paths) (*Store, error) {
	if paths.ConfigDir == "" || paths.DataDir == "" {
		return nil, errors.New("store: config and data directories are required")
	}
	for _, dir := range []string{paths.ConfigDir, paths.DataDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("store: create %s: %w", dir, err)
		}
	}
	s := &Store{paths: paths, settings: DefaultSettings(), keys: []APIKey{}, activity: []model.Activity{}}

	var sf settingsFile
	if readJSON(filepath.Join(paths.ConfigDir, SettingsFile), &sf) && sf.Settings.Protocol != "" {
		s.settings = normaliseSettings(sf.Settings)
	}
	var kf keysFile
	if readJSON(filepath.Join(paths.ConfigDir, KeysFile), &kf) {
		s.keys = append(s.keys, kf.Keys...)
	}
	var st State
	if readJSON(filepath.Join(paths.DataDir, StateFile), &st) {
		s.state = st
	}
	var af activityFile
	if readJSON(filepath.Join(paths.DataDir, ActivityFile), &af) {
		s.activity = append(s.activity, af.Items...)
		if len(s.activity) > MaxActivity {
			s.activity = s.activity[:MaxActivity]
		}
	}
	return s, nil
}

// Paths returns the directories the store was opened with.
func (s *Store) Paths() Paths { return s.paths }

// Settings returns a copy of the current settings.
func (s *Store) Settings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SaveSettings persists next and makes it current.
func (s *Store) SaveSettings(next Settings) error {
	next = normaliseSettings(next)
	s.mu.Lock()
	defer s.mu.Unlock()
	payload := settingsFile{Version: 1, Settings: next}
	if err := writeJSON(filepath.Join(s.paths.ConfigDir, SettingsFile), payload, 0o600); err != nil {
		return err
	}
	s.settings = next
	return nil
}

// State returns the boot hint.
func (s *Store) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// SaveState persists the boot hint.
func (s *Store) SaveState(st State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st.UpdatedAt = model.NowRFC3339()
	if err := writeJSON(filepath.Join(s.paths.DataDir, StateFile), st, 0o600); err != nil {
		return err
	}
	s.state = st
	return nil
}

// Keys returns a copy of the key metadata.
func (s *Store) Keys() []APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]APIKey, len(s.keys))
	copy(out, s.keys)
	return out
}

// AddKey creates a key and returns the metadata plus the one-time token.
func (s *Store) AddKey(name string) (APIKey, string, error) {
	raw, err := randomHex(32)
	if err != nil {
		return APIKey{}, "", err
	}
	token := "ct_" + raw
	key := APIKey{
		ID:        randomID(),
		Name:      name,
		Prefix:    token[:7],
		Hash:      HashToken(token),
		CreatedAt: model.NowRFC3339(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := append(append([]APIKey{}, s.keys...), key)
	if err := writeJSON(filepath.Join(s.paths.ConfigDir, KeysFile), keysFile{Version: 1, Keys: next}, 0o600); err != nil {
		return APIKey{}, "", err
	}
	s.keys = next
	return key, token, nil
}

// RemoveKey deletes a key by id. It reports whether anything was removed.
func (s *Store) RemoveKey(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make([]APIKey, 0, len(s.keys))
	found := false
	for _, k := range s.keys {
		if k.ID == id {
			found = true
			continue
		}
		next = append(next, k)
	}
	if !found {
		return false, nil
	}
	if err := writeJSON(filepath.Join(s.paths.ConfigDir, KeysFile), keysFile{Version: 1, Keys: next}, 0o600); err != nil {
		return false, err
	}
	s.keys = next
	return true, nil
}

// Authenticate verifies a bearer token against the stored hashes and records
// the use. The hash comparison is constant time.
func (s *Store) Authenticate(token string) (APIKey, bool) {
	if token == "" {
		return APIKey{}, false
	}
	sum := HashToken(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.keys {
		if subtleEqual(s.keys[i].Hash, sum) {
			s.keys[i].LastUsedAt = model.NowRFC3339()
			// Best effort persistence: a shared disk hiccup must not break auth.
			_ = writeJSON(filepath.Join(s.paths.ConfigDir, KeysFile), keysFile{Version: 1, Keys: s.keys}, 0o600)
			return s.keys[i], true
		}
	}
	return APIKey{}, false
}

// Activity returns the feed, newest first.
func (s *Store) Activity() []model.Activity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Activity, len(s.activity))
	copy(out, s.activity)
	return out
}

// AddActivity prepends an entry and trims the feed to MaxActivity.
func (s *Store) AddActivity(kind, title, tone string) (model.Activity, error) {
	item := model.Activity{
		ID:    randomID(),
		TS:    model.NowRFC3339(),
		Kind:  kind,
		Title: title,
		Tone:  tone,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make([]model.Activity, 0, len(s.activity)+1)
	next = append(next, item)
	next = append(next, s.activity...)
	if len(next) > MaxActivity {
		next = next[:MaxActivity]
	}
	if err := writeJSON(filepath.Join(s.paths.DataDir, ActivityFile), activityFile{Version: 1, Items: next}, 0o600); err != nil {
		return item, err
	}
	s.activity = next
	return item, nil
}

func normaliseSettings(s Settings) Settings {
	if s.Protocol == "" {
		s.Protocol = DefaultProtocol
	}
	if s.EdgeIPVersion == "" {
		s.EdgeIPVersion = DefaultEdgeIPVersion
	}
	if s.MaxReconnect < 0 {
		s.MaxReconnect = 0
	}
	if s.LogRetentionDays == 0 {
		s.LogRetentionDays = DefaultLogRetentionDays
	}
	if s.MaxLogLines == 0 {
		s.MaxLogLines = DefaultMaxLogLines
	}
	return s
}

func readJSON(path string, dst any) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, dst); err != nil {
		_ = os.Rename(path, path+".corrupt")
		return false
	}
	return true
}

// writeJSON writes atomically through a 0600 temp file in the same directory.
func writeJSON(path string, v any, mode os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("store: marshal %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	return WriteFileAtomic(path, data, mode)
}

// WriteFileAtomic replaces path with data using a 0600 temp file and a rename.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("store: temp file for %s: %w", filepath.Base(path), err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("store: write %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return fmt.Errorf("store: chmod %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("store: sync %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("store: close %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("store: rename %s: %w", filepath.Base(path), err)
	}
	// os.CreateTemp uses 0600 already; re-apply in case of an existing file.
	_ = os.Chmod(path, mode)
	return nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("store: random: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func randomID() string {
	v, err := randomHex(8)
	if err != nil {
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}
	return v
}
