package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.4", -1},
		{"v1.10.0", "1.9.9", 1},
		{"1.0.0", "v1.0.0", 0},
		{"2.0.0", "2.0.0-beta", 1},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q,%q) = %d; want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestAssetKey(t *testing.T) {
	cases := []struct{ os, arch, want string }{
		{"windows", "amd64", "windows-amd64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"linux", "amd64", "linux-amd64"},
	}
	for _, c := range cases {
		if got := AssetKey(c.os, c.arch); got != c.want {
			t.Errorf("AssetKey(%q,%q) = %q; want %q", c.os, c.arch, got, c.want)
		}
	}
}

func newSignedManifest(t *testing.T) (*Manifest, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := &Manifest{
		Version: "1.2.0",
		Notes:   "fix crash",
		PubDate: "2026-11-17T00:00:00Z",
		Assets: map[string]Asset{
			"linux-amd64": {URL: "https://example.com/app", SHA256: strings.Repeat("a", 64), Size: 10},
		},
	}
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, m.Payload()))
	return m, pub
}

func TestVerifyManifest(t *testing.T) {
	m, pub := newSignedManifest(t)
	if err := VerifyManifest(pub, m); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	m.Notes = "tampered"
	if err := VerifyManifest(pub, m); err == nil {
		t.Fatal("tampered manifest accepted")
	}
}

func TestManifestAssetFor(t *testing.T) {
	m, _ := newSignedManifest(t)
	a, ok := m.AssetFor("linux", "amd64")
	if !ok || a.Size != 10 {
		t.Fatalf("asset lookup failed: %v %v", a, ok)
	}
	if _, ok := m.AssetFor("darwin", "arm64"); ok {
		t.Fatal("missing asset should not be found")
	}
}

func TestCheck(t *testing.T) {
	m, _ := newSignedManifest(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(m)
	}))
	defer srv.Close()

	c := &Checker{ManifestURL: srv.URL, Client: srv.Client(), CurrentVersion: "1.0.0", GOOS: "linux", GOARCH: "amd64"}
	got, has, err := c.Check(context.Background())
	if err != nil || !has {
		t.Fatalf("Check = %v, %v; want update", has, err)
	}
	if got.Version != "1.2.0" {
		t.Errorf("version = %q", got.Version)
	}

	c.CurrentVersion = "1.2.0"
	_, has, err = c.Check(context.Background())
	if err != nil || has {
		t.Fatalf("same version should not update: %v, %v", has, err)
	}
}

func TestDownloadVerifiesSHA256(t *testing.T) {
	payload := []byte("fake-binary-content")
	sum := sha256.Sum256(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	asset := Asset{URL: srv.URL, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}
	dest := filepath.Join(dir, "update.bin")
	if err := Download(context.Background(), srv.Client(), asset, dest, nil); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if b, _ := os.ReadFile(dest); string(b) != string(payload) {
		t.Error("downloaded content mismatch")
	}

	bad := Asset{URL: srv.URL, SHA256: strings.Repeat("0", 64), Size: int64(len(payload))}
	badDest := filepath.Join(dir, "bad.bin")
	if err := Download(context.Background(), srv.Client(), bad, badDest, nil); err == nil {
		t.Fatal("hash mismatch accepted")
	}
	if _, err := os.Stat(badDest); !os.IsNotExist(err) {
		t.Error("corrupt download should be removed")
	}
}

func TestApplyUpdateUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix replace semantics")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "gominify")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	newBin := filepath.Join(dir, "gominify.new")
	if err := os.WriteFile(newBin, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyUpdate(newBin, target); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	b, _ := os.ReadFile(target)
	if string(b) != "new" {
		t.Errorf("target content = %q; want new", b)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("target not executable after update")
	}
}

func TestApplyUpdateKeepsTargetOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix replace semantics")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "app")
	os.WriteFile(target, []byte("old"), 0o755)
	if err := ApplyUpdate(filepath.Join(dir, "missing.new"), target); err == nil {
		t.Fatal("expected error for missing new binary")
	}
	if b, _ := os.ReadFile(target); string(b) != "old" {
		t.Error("original binary should be untouched on failure")
	}
}

var _ = time.Now
