package store_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/latte03/fnos-apps/apps/cloudflare-tunnel/server/internal/store"
)

func openTemp(t *testing.T) (*store.Store, store.Paths) {
	t.Helper()
	root := t.TempDir()
	paths := store.Paths{
		ConfigDir: filepath.Join(root, "config"),
		DataDir:   filepath.Join(root, "data"),
	}
	st, err := store.Open(paths)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return st, paths
}

func TestSettingsDefaultsAndRoundTrip(t *testing.T) {
	st, paths := openTemp(t)
	defaults := st.Settings()
	if defaults.Protocol != "quic" || defaults.EdgeIPVersion != "auto" || !defaults.AutoConnect || !defaults.AutoReconnect {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	if defaults.MaxReconnect != 10 || defaults.LogRetentionDays != 30 || defaults.MaxLogLines != 2000 {
		t.Fatalf("unexpected numeric defaults: %+v", defaults)
	}

	token := "cf-super-secret-token-abcdef1234"
	next := defaults
	next.AccountID = "0123456789abcdef0123456789abcdef"
	next.APIToken = token
	next.Protocol = "http2"
	next.MaxReconnect = 4
	if err := st.SaveSettings(next); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	// Masking must never reveal more than the last four characters.
	preview := store.PreviewToken(token)
	if preview != "••••1234" {
		t.Fatalf("preview = %q", preview)
	}
	if strings.Contains(preview, "cf-super") {
		t.Fatal("preview leaks the token")
	}
	if got := store.PreviewToken(""); got != "" {
		t.Fatalf("empty preview = %q", got)
	}

	reopened, err := store.Open(store.Paths{ConfigDir: paths.ConfigDir, DataDir: paths.DataDir})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	loaded := reopened.Settings()
	if loaded.APIToken != token || loaded.AccountID != next.AccountID || loaded.Protocol != "http2" || loaded.MaxReconnect != 4 {
		t.Fatalf("settings did not round trip: %+v", loaded)
	}
}

func TestAPIKeyLifecycle(t *testing.T) {
	st, paths := openTemp(t)
	key, token, err := st.AddKey("Home Assistant")
	if err != nil {
		t.Fatalf("AddKey: %v", err)
	}
	if !strings.HasPrefix(token, "ct_") || len(token) != 3+64 {
		t.Fatalf("unexpected token shape: %q", token)
	}
	if key.Prefix != token[:7] {
		t.Fatalf("prefix = %q, want %q", key.Prefix, token[:7])
	}
	if key.Hash == token || key.Hash == "" {
		t.Fatal("token must not be stored in clear")
	}
	if key.Hash != store.HashToken(token) {
		t.Fatal("hash mismatch")
	}

	raw, err := os.ReadFile(filepath.Join(paths.ConfigDir, store.KeysFile))
	if err != nil {
		t.Fatalf("read keys file: %v", err)
	}
	if strings.Contains(string(raw), token) {
		t.Fatal("apikeys.json contains the clear text token")
	}
	if !strings.Contains(string(raw), key.Hash) {
		t.Fatal("apikeys.json is missing the hash")
	}

	auth, ok := st.Authenticate(token)
	if !ok || auth.ID != key.ID {
		t.Fatalf("Authenticate failed for a valid token")
	}
	if auth.LastUsedAt == "" {
		t.Fatal("LastUsedAt was not recorded")
	}
	if _, ok := st.Authenticate("ct_deadbeef"); ok {
		t.Fatal("Authenticate accepted an invalid token")
	}
	if _, ok := st.Authenticate(""); ok {
		t.Fatal("Authenticate accepted an empty token")
	}

	removed, err := st.RemoveKey(key.ID)
	if err != nil || !removed {
		t.Fatalf("RemoveKey = %v, %v", removed, err)
	}
	if _, ok := st.Authenticate(token); ok {
		t.Fatal("removed key still authenticates")
	}
	if removed, _ := st.RemoveKey(key.ID); removed {
		t.Fatal("RemoveKey reported a second removal")
	}
}

func TestSecretFilesArePrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows")
	}
	st, paths := openTemp(t)
	if err := st.SaveSettings(store.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if _, _, err := st.AddKey("cli"); err != nil {
		t.Fatalf("AddKey: %v", err)
	}
	for _, name := range []string{store.SettingsFile, store.KeysFile} {
		info, err := os.Stat(filepath.Join(paths.ConfigDir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("%s mode = %o, want 600", name, mode)
		}
	}
}

func TestActivityIsBounded(t *testing.T) {
	st, _ := openTemp(t)
	for i := 0; i < store.MaxActivity+5; i++ {
		if _, err := st.AddActivity("connect", "entry", "green"); err != nil {
			t.Fatalf("AddActivity: %v", err)
		}
	}
	items := st.Activity()
	if len(items) != store.MaxActivity {
		t.Fatalf("got %d items, want %d", len(items), store.MaxActivity)
	}
	if items[0].TS == "" || items[0].ID == "" || items[0].Tone != "green" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
}

func TestCorruptSettingsFallsBack(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, store.SettingsFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(store.Paths{ConfigDir: configDir, DataDir: dataDir})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := st.Settings().Protocol; got != "quic" {
		t.Fatalf("expected defaults after a corrupt file, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(configDir, store.SettingsFile+".corrupt")); err != nil {
		t.Fatalf("corrupt file was not preserved: %v", err)
	}
}

func TestStatePersistence(t *testing.T) {
	st, paths := openTemp(t)
	if err := st.SaveState(store.State{TunnelID: "tun-1", TunnelName: "fnos-main", AutoConnect: true, LastState: "connected"}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	reopened, err := store.Open(store.Paths{ConfigDir: paths.ConfigDir, DataDir: paths.DataDir})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	state := reopened.State()
	if state.TunnelID != "tun-1" || state.LastState != "connected" || state.UpdatedAt == "" {
		t.Fatalf("unexpected state: %+v", state)
	}
}
