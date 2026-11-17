// Package update 实现 GoMinify 的 OTA 自更新：清单校验、签名验证、下载与原子替换。
package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DefaultManifestURL 是默认更新清单地址，发布时改成自己的域名。
const DefaultManifestURL = "https://raw.githubusercontent.com/Z-041/gominify/main/ota/latest.json"

// DefaultPublicKey 是内置的 ed25519 公钥（base64），构建时可用 -ldflags 覆盖。
// 留空表示不校验签名，仅用于本地调试，正式发布必须填写。
var DefaultPublicKey = "I2NEKMq88px7bNhUISP+bi584RNXPZaF2VOGZ3FY/6M="

// Asset 描述某个平台对应的更新产物。
type Asset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Manifest 是远端更新清单。
type Manifest struct {
	Version   string           `json:"version"`
	Notes     string           `json:"notes"`
	PubDate   string           `json:"pub_date"`
	Assets    map[string]Asset `json:"assets"`
	Signature string           `json:"signature"`
}

// Payload 返回参与签名的规范化内容（排除 signature 本身）。
func (m *Manifest) Payload() []byte {
	type canonical struct {
		Version string           `json:"version"`
		Notes   string           `json:"notes"`
		PubDate string           `json:"pub_date"`
		Assets  map[string]Asset `json:"assets"`
	}
	b, _ := json.Marshal(canonical{m.Version, m.Notes, m.PubDate, m.Assets})
	return b
}

// AssetFor 返回当前平台的产物。
func (m *Manifest) AssetFor(goos, goarch string) (Asset, bool) {
	a, ok := m.Assets[AssetKey(goos, goarch)]
	return a, ok
}

// AssetKey 生成清单里的平台键，如 windows-amd64、darwin-arm64。
func AssetKey(goos, goarch string) string {
	return goos + "-" + goarch
}

// CompareVersions 比较语义化版本，返回 -1/0/1。支持可选的 v 前缀与 -prerelease 后缀。
func CompareVersions(a, b string) int {
	pa, prea := splitVersion(a)
	pb, preb := splitVersion(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	// 主版本相同：正式版高于预发布版
	if prea == "" && preb != "" {
		return 1
	}
	if prea != "" && preb == "" {
		return -1
	}
	return strings.Compare(prea, preb)
}

func splitVersion(v string) ([]int, string) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		nums = append(nums, n)
	}
	return nums, pre
}

// ParsePublicKey 解析 base64 的 ed25519 公钥。
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "ed25519:"))
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("公钥解析失败: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("公钥长度错误: %d", len(b))
	}
	return ed25519.PublicKey(b), nil
}

// VerifyManifest 校验清单签名，防止更新包被篡改。
func VerifyManifest(pub ed25519.PublicKey, m *Manifest) error {
	if len(pub) != ed25519.PublicKeySize {
		return errors.New("公钥无效")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(m.Signature, "ed25519:")))
	if err != nil {
		return fmt.Errorf("签名解码失败: %w", err)
	}
	if !ed25519.Verify(pub, m.Payload(), sig) {
		return errors.New("更新清单签名校验失败")
	}
	return nil
}

// Checker 负责拉取清单并判断是否有新版本。
type Checker struct {
	ManifestURL    string
	Client         *http.Client
	CurrentVersion string
	GOOS           string
	GOARCH         string
	PublicKey      ed25519.PublicKey
}

func (c *Checker) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Check 返回远端清单以及是否存在更新。
func (c *Checker) Check(ctx context.Context) (*Manifest, bool, error) {
	if c.ManifestURL == "" {
		return nil, false, errors.New("未配置更新地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ManifestURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "GoMinify-Updater/"+c.CurrentVersion)
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("检查更新失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("更新服务返回 %d", resp.StatusCode)
	}

	var m Manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return nil, false, fmt.Errorf("更新清单格式错误: %w", err)
	}
	if m.Version == "" {
		return nil, false, errors.New("更新清单缺少版本号")
	}
	if len(c.PublicKey) == ed25519.PublicKeySize {
		if err := VerifyManifest(c.PublicKey, &m); err != nil {
			return nil, false, err
		}
	}
	if _, ok := m.AssetFor(c.GOOS, c.GOARCH); !ok {
		return nil, false, fmt.Errorf("当前平台 %s 暂无更新包", AssetKey(c.GOOS, c.GOARCH))
	}
	return &m, CompareVersions(m.Version, c.CurrentVersion) > 0, nil
}

// Download 下载更新包到 dest，并校验 SHA-256；失败时不会留下残缺文件。
func Download(ctx context.Context, client *http.Client, asset Asset, dest string, progress func(done, total int64)) error {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载更新包失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载更新包返回 %d", resp.StatusCode)
	}

	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(part)
	}()

	h := sha256.New()
	total := asset.Size
	if total <= 0 {
		total = resp.ContentLength
	}
	var done int64
	buf := make([]byte, 256*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			h.Write(buf[:n])
			done += int64(n)
			if progress != nil {
				progress(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	sum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(sum, asset.SHA256) {
		return fmt.Errorf("更新包校验失败: 期望 %s，实际 %s", asset.SHA256, sum)
	}
	if asset.Size > 0 && done != asset.Size {
		return fmt.Errorf("更新包大小不符: 期望 %d，实际 %d", asset.Size, done)
	}
	return os.Rename(part, dest)
}

// ApplyUpdate 将 newBinary 原子替换到 target。失败时保留原程序不变。
func ApplyUpdate(newBinary, target string) error {
	if _, err := os.Stat(newBinary); err != nil {
		return fmt.Errorf("新版本文件不可用: %w", err)
	}
	mode := os.FileMode(0o755)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm() | 0o111
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".gominify-update-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { tmp.Close(); os.Remove(tmpName) }

	src, err := os.Open(newBinary)
	if err != nil {
		cleanup()
		return err
	}
	if _, err := io.Copy(tmp, src); err != nil {
		src.Close()
		cleanup()
		return err
	}
	src.Close()
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if runtime.GOOS == "windows" {
		old := target + ".old"
		_ = os.Remove(old)
		if err := os.Rename(target, old); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("无法替换正在运行的程序: %w", err)
		}
		if err := os.Rename(tmpName, target); err != nil {
			_ = os.Rename(old, target)
			os.Remove(tmpName)
			return err
		}
		return nil // .old 在下次启动时清理
	}

	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// CleanupLeftovers 清理 Windows 更新后残留的 .old 文件。
func CleanupLeftovers(target string) {
	if runtime.GOOS != "windows" {
		return
	}
	_ = os.Remove(target + ".old")
}

// Restart 启动新版本进程；调用方随后应退出当前进程。
func Restart(exe string) error {
	cmd := exec.Command(exe)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
