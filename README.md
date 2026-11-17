# GoMinify

本地 JS / CSS 压缩器，Go + Wails 桌面应用，双击即用，代码不离开本机。

- 引擎：`tdewolff/minify`（纯 Go，支持 JS / CSS / HTML / JSON）
- 界面：Wails v2 + Vue 3 + TypeScript
- 分发：单可执行文件；内置 OTA 自更新，升级不必重新下载完整安装包

---

## 当前进度

| 模块 | 状态 |
|---|---|
| 压缩引擎 `internal/minifier` | ✅ 完成，含目录递归展开 |
| OTA 更新 `internal/update` | ✅ 完成，10 个单测 |
| OTA 发布工具 `cmd/ota-publish` | ✅ 完成，可生成并签名清单 |
| 压缩历史 `internal/history` | ✅ 完成，本地 JSON 持久化 + 最近导出目录 |
| Wails 骨架 + 界面 | ✅ 可构建运行（Windows 已验证） |
| 文件夹递归模式 / 历史记录 | ✅ 完成，导入目录保持结构导出 |

---

## 目录结构

```
gominify/
├── .github/workflows/       # CI：三平台构建 + Release + OTA 清单
├── main.go                  # Wails 入口，窗口与资源嵌入
├── app.go                   # 绑定给前端的 API：压缩、导出、更新
├── internal/
│   ├── minifier/            # 压缩引擎封装（扩展名路由、选项、目录展开、行号提取）
│   ├── history/             # 压缩历史：本地 JSON 持久化、最近导出目录
│   └── update/              # OTA：清单校验、签名、下载、原子替换
├── cmd/ota-publish/         # 发布工具：扫描产物 → 生成清单 → ed25519 签名
├── frontend/                # Vue 3 + TS + Vite
│   └── src/
│       ├── App.vue          # 主界面
│       └── components/      # 文件列表、预览、选项抽屉、更新弹窗
├── scripts/release.sh       # 打版本并生成更新清单
└── ota/latest.example.json  # 清单格式示例
```

## 开发环境

- Go ≥ 1.26
- Node.js ≥ 18、npm
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Linux 额外依赖：`libgtk-3-dev libwebkit2gtk-4.1-dev`，构建加 `-tags webkit2_41`

## 开发与构建

```bash
wails dev                      # 热重载开发
wails build                    # 当前平台打包，产物在 build/bin/
wails build -tags webkit2_41   # Ubuntu 24.04 等只有 webkit2gtk-4.1 的系统
go test ./internal/...         # 引擎与更新模块单测
```

Windows / macOS 的安装包需在对应系统上构建（或在 CI 里用矩阵构建）。

---

## CI 三平台构建与发布

`.github/workflows/release.yml` 在 GitHub Actions 上原生构建四个平台：

| 平台 | Runner | OTA 产物 | 安装包 |
|---|---|---|---|
| windows-amd64 | windows-latest | `GoMinify-windows-amd64.exe` | 同左 |
| linux-amd64 | ubuntu-24.04 | `GoMinify-linux-amd64` | 同左 |
| darwin-amd64 | macos-15-intel | `GoMinify-darwin-amd64` | `.app.zip` |
| darwin-arm64 | macos-15 | `GoMinify-darwin-arm64` | `.app.zip` |

发布流程：

1. 仓库 Secrets 添加 `OTA_PRIVATE_KEY`，内容为本机 `ota_private.key` 的文本（base64 seed）
2. 推送 tag 自动发布：

```bash
git tag v1.0.2
git push origin v1.0.2
```

CI 会完成：四平台构建 → 创建 Release 并上传产物与 macOS 安装包 → 用私钥签名 `ota/latest.json` → 提交回 main。

也可以在 Actions 页面手动运行 `Build & Release`：填入版本号，`publish` 关闭时只构建并保留 artifact（3 天），不创建 Release、不改清单。

注意事项：

- Linux 产物依赖 webkit2gtk-4.1（Ubuntu 24.04+），22.04 及更早版本无法直接运行
- macOS 未做代码签名/公证，首次打开需右键选择“打开”，或执行 `xattr -dr com.apple.quarantine GoMinify.app`
- `raw.githubusercontent.com` 有约 5 分钟 CDN 缓存，发布后客户端可能短时读到旧清单
- 若 main 分支保护阻止 bot 推送，可改用 PAT Secret 或将清单回写改成 PR

---

## OTA 自更新机制

设计目标是「只下增量替换包，不重新下载整个应用」。

### 1. 检查

应用启动后可手动点右上角版本号触发检查：

```
GET https://<你的域名>/gominify/latest.json
```

清单格式（见 `ota/latest.example.json`）：

```json
{
  "version": "1.1.0",
  "notes": "新增文件夹模式，修复大文件预览卡顿。",
  "pub_date": "2026-11-20T08:00:00Z",
  "assets": {
    "windows-amd64": { "url": "https://.../GoMinify-windows-amd64.exe", "sha256": "…", "size": 11275504 },
    "darwin-arm64":  { "url": "https://.../GoMinify-darwin-arm64",     "sha256": "…", "size": 10999800 },
    "linux-amd64":   { "url": "https://.../GoMinify-linux-amd64",      "sha256": "…", "size": 11275504 }
  },
  "signature": "base64(ed25519(清单规范化内容))"
}
```

只有 `version` 高于当前版本、且存在当前平台产物时才提示更新。

### 2. 校验（两道）

1. **清单签名**：ed25519，防止有人伪造更新地址或篡改版本信息
2. **产物 SHA-256**：下载后逐个字节校验，失败立刻删除临时文件，原程序不动

> 正式发布前必须：`go run ./cmd/ota-publish -gen-key`，把输出的公钥填进
> `internal/update/update.go` 的 `DefaultPublicKey`；私钥只留在发布机上，绝不入库。

### 3. 替换与重启

- **Linux / macOS**：下载到临时目录 → 校验 → 复制到程序同目录的临时文件 → `rename` 原子替换 → 启动新进程并退出旧进程
- **Windows**：运行中的 exe 不能直接覆盖，采用「当前 exe 改名为 `.old` → 新文件就位 → 启动新进程」，`.old` 在下次启动时清理
- 替换前任何一步失败，原程序保持不变（有单测覆盖）

### 4. 发布一个新版本

```bash
# 1) 构建当前平台产物（每个平台各跑一次，或在 CI 矩阵里跑）
./scripts/release.sh 1.1.0 "新增文件夹模式"

# 2) 把 dist/ 里的 GoMinify-<os>-<arch>* 和 latest.json 上传到静态托管
# 3) 手机/桌面端下次检查更新即可收到
```

也可以直接调用发布工具：

```bash
go run ./cmd/ota-publish \
  -dir dist -version 1.1.0 -notes "修复若干问题" \
  -base-url https://dl.example.com/gominify \
  -key ota_private.key
```

### 5. 配置更新地址

优先级：`-ldflags "-X main.UpdateURL=..."` > 环境变量 `GOMINIFY_UPDATE_URL` > 代码内默认值。

```bash
wails build -ldflags "-X main.Version=1.1.0 -X main.UpdateURL=https://dl.example.com/gominify/latest.json"
```

---

## 安全说明

- 所有压缩在本机完成，不上传源码
- 更新包强制 SHA-256 校验；配置公钥后还会验证 ed25519 签名
- 更新只替换程序本体，不触碰用户文件
- 单文件上限 5 MB，防止超大文件拖垮界面

## 后续计划

- 深色/浅色主题
- 可选：接入 esbuild 作为 JS 压缩备选通道，进一步提升压缩率

---

内容由 AI 生成