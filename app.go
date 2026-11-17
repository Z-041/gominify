package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gominify/internal/minifier"
	"gominify/internal/update"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version 为当前构建版本，发布时用 -ldflags "-X main.Version=1.2.0" 注入。
var Version = "1.0.0"

// UpdateURL 可在构建时注入；为空则读取 GOMINIFY_UPDATE_URL，都没有则用默认地址。
var UpdateURL = ""

// maxFileSize 单文件上限 5 MB，避免大文件把界面拖死。
const maxFileSize = 5 << 20

// FileResult 是给前端展示的压缩结果（不含二进制，直接带文本）。
type FileResult struct {
	ID           string  `json:"id"`
	Path         string  `json:"path"`
	Name         string  `json:"name"`
	Language     string  `json:"language"`
	OriginalSize int     `json:"originalSize"`
	MinifiedSize int     `json:"minifiedSize"`
	Ratio        float64 `json:"ratio"`
	ElapsedMS    int64   `json:"elapsedMs"`
	Output       string  `json:"output"`
	Source       string  `json:"source"`
	Error        string  `json:"error"`
	Line         int     `json:"line"`
}

// UpdateInfo 是检查更新返回给前端的信息。
type UpdateInfo struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Notes     string `json:"notes"`
	Size      int64  `json:"size"`
}

// UpdateProgress 通过事件推送给前端。
type UpdateProgress struct {
	Percent int   `json:"percent"`
	Done    int64 `json:"done"`
	Total   int64 `json:"total"`
}

// App 是绑定给前端的后端对象。
type App struct {
	ctx     context.Context
	mu      sync.Mutex
	results map[string]*minifier.Result
	seq     int
}

func NewApp() *App {
	return &App{results: make(map[string]*minifier.Result)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 清理上次 Windows 更新残留的 .old 文件
	if exe, err := os.Executable(); err == nil {
		update.CleanupLeftovers(exe)
	}
}

func (a *App) shutdown(ctx context.Context) {}

// CompressFiles 批量压缩。单个文件失败不会中断其余文件。
func (a *App) CompressFiles(paths []string, opts minifier.Options) []FileResult {
	out := make([]FileResult, 0, len(paths))
	for _, p := range paths {
		out = append(out, a.compressOne(p, opts))
	}
	return out
}

func (a *App) compressOne(path string, opts minifier.Options) FileResult {
	info, err := os.Stat(path)
	if err != nil {
		return FileResult{ID: a.nextID("e"), Path: path, Name: filepath.Base(path), Error: "无法读取文件: " + err.Error()}
	}
	if info.IsDir() {
		return FileResult{ID: a.nextID("e"), Path: path, Name: filepath.Base(path), Error: "暂不支持目录，请选择文件"}
	}
	if info.Size() > maxFileSize {
		return FileResult{ID: a.nextID("e"), Path: path, Name: filepath.Base(path), Error: "文件超过 5 MB，已跳过"}
	}

	res := minifier.MinifyFile(path, opts)
	id := a.nextID("f")
	a.mu.Lock()
	a.results[id] = &res
	a.mu.Unlock()

	fr := FileResult{
		ID:           id,
		Path:         path,
		Name:         res.Name,
		Language:     res.Language,
		OriginalSize: res.OriginalSize,
		MinifiedSize: res.MinifiedSize,
		Ratio:        res.Ratio,
		ElapsedMS:    res.Elapsed.Milliseconds(),
	}
	if res.Err != nil {
		fr.Error = res.Err.Error()
		fr.Line = minifier.LineOf(res.Err)
		return fr
	}
	fr.Output = string(res.Output)
	fr.Source = string(res.Source)
	return fr
}

func (a *App) nextID(prefix string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.seq++
	return fmt.Sprintf("%s%d-%d", prefix, a.seq, time.Now().UnixNano())
}

// minifiedName 生成 xxx.min.ext 形式的文件名。
func minifiedName(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if strings.HasSuffix(base, ".min") {
		return name
	}
	return base + ".min" + ext
}

// SaveResult 导出单个结果到指定目录。
func (a *App) SaveResult(id, destDir string) (string, error) {
	a.mu.Lock()
	res, ok := a.results[id]
	a.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("结果不存在或已过期")
	}
	if res.Err != nil {
		return "", fmt.Errorf("该文件压缩失败，无法导出")
	}
	if destDir == "" {
		destDir = filepath.Dir(res.Path)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(destDir, minifiedName(res.Name))
	if err := os.WriteFile(target, res.Output, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

// SaveAll 批量导出成功的结果，返回成功数量。
func (a *App) SaveAll(ids []string, destDir string) (int, error) {
	saved := 0
	var lastErr error
	for _, id := range ids {
		if _, err := a.SaveResult(id, destDir); err != nil {
			lastErr = err
			continue
		}
		saved++
	}
	if saved == 0 && lastErr != nil {
		return 0, lastErr
	}
	return saved, nil
}

// OpenFiles 打开系统多选文件对话框。
func (a *App) OpenFiles() ([]string, error) {
	return wruntime.OpenMultipleFilesDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择要压缩的文件",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Web 资源 (*.js;*.css;*.html;*.json)", Pattern: "*.js;*.mjs;*.cjs;*.css;*.html;*.htm;*.json"},
		},
	})
}

// OpenDir 选择导出目录。
func (a *App) OpenDir() (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "选择导出目录"})
}

// GetVersion 返回当前版本。
func (a *App) GetVersion() string { return Version }

func (a *App) checker() *update.Checker {
	url := UpdateURL
	if url == "" {
		url = os.Getenv("GOMINIFY_UPDATE_URL")
	}
	if url == "" {
		url = update.DefaultManifestURL
	}
	var pub ed25519.PublicKey
	if k, err := update.ParsePublicKey(update.DefaultPublicKey); err == nil {
		pub = k
	}
	return &update.Checker{
		ManifestURL:    url,
		CurrentVersion: Version,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		PublicKey:      pub,
	}
}

// CheckUpdate 检查是否有新版本。
func (a *App) CheckUpdate() (UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	m, has, err := a.checker().Check(ctx)
	if err != nil {
		return UpdateInfo{Current: Version}, err
	}
	info := UpdateInfo{Available: has, Current: Version, Latest: m.Version, Notes: m.Notes}
	if asset, ok := m.AssetFor(runtime.GOOS, runtime.GOARCH); ok {
		info.Size = asset.Size
	}
	return info, nil
}

// ApplyUpdate 下载、校验并替换当前程序，成功后重启。
// 前端通过 "update:progress" 事件拿到下载进度。
func (a *App) ApplyUpdate() error {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
	defer cancel()

	chk := a.checker()
	m, has, err := chk.Check(ctx)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("当前已是最新版本")
	}
	asset, ok := m.AssetFor(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return fmt.Errorf("当前平台没有可用更新包")
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)

	tmpDir, err := os.MkdirTemp("", "gominify-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	downloadPath := filepath.Join(tmpDir, "package.bin")

	wruntime.EventsEmit(a.ctx, "update:progress", UpdateProgress{Percent: 0})
	err = update.Download(ctx, nil, asset, downloadPath, func(done, total int64) {
		p := 0
		if total > 0 {
			p = int(done * 100 / total)
		}
		wruntime.EventsEmit(a.ctx, "update:progress", UpdateProgress{Percent: p, Done: done, Total: total})
	})
	if err != nil {
		return err
	}

	if err := update.ApplyUpdate(downloadPath, exe); err != nil {
		return err
	}
	wruntime.EventsEmit(a.ctx, "update:progress", UpdateProgress{Percent: 100})
	// 稍等前端收到 100% 再重启
	go func() {
		time.Sleep(600 * time.Millisecond)
		_ = update.Restart(exe)
		wruntime.Quit(a.ctx)
	}()
	return nil
}
