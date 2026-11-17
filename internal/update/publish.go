package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BuildManifest 扫描 dir 下形如 GoMinify-<os>-<arch>[.exe] 的产物，生成更新清单。
func BuildManifest(dir, version, notes, baseURL string) (*Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	assets := make(map[string]Asset)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "GoMinify-") {
			continue
		}
		key := strings.TrimPrefix(name, "GoMinify-")
		key = strings.TrimSuffix(key, ".exe")
		if len(strings.Split(key, "-")) != 2 {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		assets[key] = Asset{
			URL:    strings.TrimRight(baseURL, "/") + "/" + name,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(data)),
		}
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("目录 %s 下没有找到 GoMinify-<os>-<arch> 产物", dir)
	}

	return &Manifest{
		Version: version,
		Notes:   notes,
		PubDate: time.Now().UTC().Format(time.RFC3339),
		Assets:  assets,
	}, nil
}

// SignManifest 用 ed25519 私钥给清单签名，防止清单被替换。
func SignManifest(m *Manifest, priv ed25519.PrivateKey) error {
	if len(priv) != ed25519.PrivateKeySize {
		return fmt.Errorf("私钥长度错误: %d", len(priv))
	}
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, m.Payload()))
	return nil
}
