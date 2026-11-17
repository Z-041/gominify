#!/usr/bin/env bash
# 构建当前平台产物，并生成签名的 OTA 清单。
# 用法: ./scripts/release.sh <version> [notes]
# 环境变量: BASE_URL（必填，产物下载地址前缀）、OTA_KEY（私钥路径，默认 ota_private.key）
set -euo pipefail

VERSION="${1:?用法: ./scripts/release.sh <version> [notes]}"
NOTES="${2:-}"
BASE_URL="${BASE_URL:?请设置 BASE_URL，例如 https://dl.example.com/gominify}"
OTA_KEY="${OTA_KEY:-ota_private.key}"

if [[ ! -f "$OTA_KEY" ]]; then
  echo "未找到签名私钥 $OTA_KEY"
  echo "先生成: go run ./cmd/ota-publish -gen-key -key $OTA_KEY"
  echo "然后把公钥填入 internal/update/update.go 的 DefaultPublicKey"
  exit 1
fi

echo "==> 构建 v$VERSION ($(go env GOOS)/$(go env GOARCH))"
wails build -clean -ldflags "-X main.Version=$VERSION"

mkdir -p dist
EXT=""
[[ "$(go env GOOS)" == "windows" ]] && EXT=".exe"
cp "build/bin/GoMinify$EXT" "dist/GoMinify-$(go env GOOS)-$(go env GOARCH)$EXT"

echo "==> 生成并签名更新清单"
go run ./cmd/ota-publish \
  -dir dist \
  -version "$VERSION" \
  -notes "$NOTES" \
  -base-url "$BASE_URL" \
  -key "$OTA_KEY"

echo
echo "完成。将 dist/ 下的产物与 latest.json 上传到: $BASE_URL"