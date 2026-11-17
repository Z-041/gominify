// ota-publish 生成并签名 OTA 更新清单。
//
//	go run ./cmd/ota-publish -gen-key -key ota_private.key
//	go run ./cmd/ota-publish -dir dist -version 1.1.0 -notes "修复若干问题" \
//	    -base-url https://dl.example.com/gominify -key ota_private.key -out dist/latest.json
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gominify/internal/update"
)

func main() {
	dir := flag.String("dir", "dist", "包含平台产物的目录")
	version := flag.String("version", "", "版本号，如 1.1.0")
	notes := flag.String("notes", "", "更新说明")
	baseURL := flag.String("base-url", "", "产物下载地址前缀")
	keyPath := flag.String("key", "ota_private.key", "ed25519 私钥文件")
	out := flag.String("out", "", "输出清单路径，默认 <dir>/latest.json")
	genKey := flag.Bool("gen-key", false, "生成新的签名密钥对")
	flag.Parse()

	if *genKey {
		gen(*keyPath)
		return
	}
	if *version == "" {
		fatal("-version 必填")
	}
	if *baseURL == "" {
		fatal("-base-url 必填")
	}
	if *out == "" {
		*out = filepath.Join(*dir, "latest.json")
	}

	priv, err := loadKey(*keyPath)
	if err != nil {
		fatal("读取私钥: %v", err)
	}
	m, err := update.BuildManifest(*dir, *version, *notes, *baseURL)
	if err != nil {
		fatal("生成清单: %v", err)
	}
	if err := update.SignManifest(m, priv); err != nil {
		fatal("签名: %v", err)
	}

	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fatal("写出清单: %v", err)
	}
	fmt.Printf("清单已生成: %s\n版本: %s\n平台: %d 个\n", *out, m.Version, len(m.Assets))
	for k, a := range m.Assets {
		fmt.Printf("  %-16s %8d bytes  %s\n", k, a.Size, a.SHA256[:12]+"…")
	}
}

func gen(keyPath string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal("生成密钥: %v", err)
	}
	seed := base64.StdEncoding.EncodeToString(priv.Seed())
	if err := os.WriteFile(keyPath, []byte(seed+"\n"), 0o600); err != nil {
		fatal("写入私钥: %v", err)
	}
	fmt.Println("已生成密钥对")
	fmt.Println("私钥文件:", keyPath, "（务必保密，不要提交到仓库）")
	fmt.Println("公钥（填入 internal/update/update.go 的 DefaultPublicKey）:")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
}

func loadKey(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(string(trimSpace(b)))
	if err != nil {
		return nil, err
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	default:
		return nil, fmt.Errorf("密钥长度错误: %d", len(raw))
	}
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
