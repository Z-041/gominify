package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifest(t *testing.T) {
	dir := t.TempDir()
	linux := []byte("linux-binary")
	windows := []byte("windows-binary")
	if err := os.WriteFile(filepath.Join(dir, "GoMinify-linux-amd64"), linux, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "GoMinify-windows-amd64.exe"), windows, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := BuildManifest(dir, "1.2.0", "修复若干问题", "https://dl.example.com/gominify")
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	if m.Version != "1.2.0" {
		t.Errorf("version = %q", m.Version)
	}
	if len(m.Assets) != 2 {
		t.Fatalf("assets = %d; want 2 (%v)", len(m.Assets), m.Assets)
	}

	la, ok := m.Assets["linux-amd64"]
	if !ok {
		t.Fatal("missing linux-amd64 asset")
	}
	sum := sha256.Sum256(linux)
	if la.SHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("sha256 mismatch: %q", la.SHA256)
	}
	if la.Size != int64(len(linux)) {
		t.Errorf("size = %d; want %d", la.Size, len(linux))
	}
	if la.URL != "https://dl.example.com/gominify/GoMinify-linux-amd64" {
		t.Errorf("url = %q", la.URL)
	}
	if _, ok := m.Assets["windows-amd64"]; !ok {
		t.Error("missing windows-amd64 asset")
	}
}

func TestSignManifestRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := &Manifest{Version: "2.0.0", Notes: "big release", PubDate: "2026-11-17T00:00:00Z", Assets: map[string]Asset{
		"linux-amd64": {URL: "https://x/y", SHA256: "abc", Size: 3},
	}}
	if err := SignManifest(m, priv); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if m.Signature == "" {
		t.Fatal("signature empty")
	}
	if err := VerifyManifest(pub, m); err != nil {
		t.Fatalf("verify: %v", err)
	}
	m.Notes = "tampered"
	if err := VerifyManifest(pub, m); err == nil {
		t.Fatal("tampered manifest verified")
	}
}
