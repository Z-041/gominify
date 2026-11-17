package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gominify/internal/history"
	"gominify/internal/minifier"
	"gominify/internal/update"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version 为当前构建版本，发布时用 -ldflags "-X main.Version=1.2.0" 注入。
var Version = "1.0.0"

// UpdateURL 可在构建时注入；为空则读取 GOMINIFY_UPDATE_URL，都没有则用默认地址。
var UpdateURL = ""

// FileResult 是给前端展示的压缩结果（不含二进制，直接带文本）。
type FileResult struct {
	ID           string  `json:"id"`
	Path         string  `json:"path"`
	RootPath     string  `json:"rootPath"`
	RelPath      string  `json:"relPath"`
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

// BatchResult 是一次导入（文件 + 目录）的汇总结果。
type BatchResult struct {
	Results   []FileResult `json:"results"`
	Skipped   int          `json:"skipped"`
	Warnings  []string     `json:"warnings"`
	Truncated bool         `json:"truncated"`
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

// resultEntry 保存压缩结果及其目录结构信息，供导出时还原路径。
type resultEntry struct {
	res  minifier.Result
	root string
	rel  string
	opts minifier.Options
}

// App 是绑定给前端的后端对象。
type App struct {
	ctx     context.Context
	mu      sync.Mutex
	results map[string]*resultEntry
	seq     int
	history *history.Store
}

func NewApp() *App {
	return &App{results: make(map[string]*resultEntry)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 清理上次 Windows 更新残留的 .old 文件
	if exe, err := os.Executable(); err == nil {
		update.CleanupLeftovers(exe)
	}
	store, err := history.Open()
	if err != nil {
		log.Printf("历史记录不可用，已降级为内存模式: %v", err)
		return
	}
	a.history = store
}

func (a *App) shutdown(ctx context.Context) {}

// CompressPaths 导入文件或目录并批量压缩。目录会递归展开，单个文件失败不影响其余文件。
func (a *App) CompressPaths(paths []string, opts minifier.Options) BatchResult {
	exp := minifier.Expand(paths, 0)
	out := BatchResult{
		Results:   make([]FileResult, 0, len(exp.Refs)),
		Skipped:   exp.Skipped,
		Warnings:  exp.Problems,
		Truncated: exp.Truncated,
	}
	for _, ref := range exp.Refs {
		out.Results = append(out.Results, a.compressRef(ref, opts))
	}
	return out
}

// Recompress 按已有结果 id 重新压缩，保持 id、顺序与目录结构不变。
func (a *App) Recompress(ids []string, opts minifier.Options) []FileResult {
	out := make([]FileResult, 0, len(ids))
	for _, id := range ids {
		a.mu.Lock()
		e, ok := a.results[id]
		a.mu.Unlock()
		if !ok {
			out = append(out, FileResult{ID: id, Error: "结果不存在或已过期"})
			continue
		}

		res := minifier.MinifyFile(e.res.Path, opts)
		a.mu.Lock()
		e.res = res
		e.opts = opts
		a.mu.Unlock()
		out = append(out, fileResult(id, e))
	}
	return out
}

func (a *App) compressRef(ref minifier.FileRef, opts minifier.Options) FileResult {
	id := a.nextID("f")
	entry := &resultEntry{res: minifier.MinifyFile(ref.Path, opts), root: ref.Root, rel: ref.Rel, opts: opts}

	a.mu.Lock()
	a.results[id] = entry
	a.mu.Unlock()

	return fileResult(id, entry)
}

func fileResult(id string, e *resultEntry) FileResult {
	res := e.res
	fr := FileResult{
		ID:           id,
		Path:         res.Path,
		RootPath:     e.root,
		RelPath:      e.rel,
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

// prepareTarget 计算导出路径，并返回对应的结果记录。
func (a *App) prepareTarget(id, destDir string) (*resultEntry, string, error) {
	a.mu.Lock()
	e, ok := a.results[id]
	a.mu.Unlock()
	if !ok {
		return nil, "", fmt.Errorf("结果不存在或已过期")
	}
	if e.res.Err != nil {
		return nil, "", fmt.Errorf("该文件压缩失败，无法导出")
	}
	if destDir == "" {
		destDir = filepath.Dir(e.res.Path)
	}

	var target string
	if e.rel == "" {
		target = filepath.Join(destDir, minifiedName(e.res.Name))
	} else {
		rel := filepath.FromSlash(e.rel)
		if !safeRel(rel) {
			return nil, "", fmt.Errorf("相对路径无效: %s", e.rel)
		}
		target = filepath.Join(destDir, filepath.Base(e.root), filepath.Dir(rel), minifiedName(filepath.Base(rel)))
	}

	if !withinDir(destDir, target) {
		return nil, "", fmt.Errorf("导出路径越界: %s", target)
	}
	return e, target, nil
}

// safeRel 校验相对路径不会逃逸出根目录。
func safeRel(rel string) bool {
	if rel == "" || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	clean := filepath.Clean(rel)
	return clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

// withinDir 判断 target 是否位于 dest 目录内。
func withinDir(dest, target string) bool {
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDest, absTarget)
	if err != nil {
		return false
	}
	return safeRel(rel)
}

func writeEntry(e *resultEntry, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, e.res.Output, 0o644)
}

// recordExport 把一次成功导出写入历史，DestDir 记忆最近导出目录。
func (a *App) recordExport(files, failed, original, minified int, destDir string, opts minifier.Options) {
	if files == 0 || a.history == nil {
		return
	}
	entry := history.Entry{
		FileCount:     files,
		Failed:        failed,
		OriginalTotal: original,
		MinifiedTotal: minified,
		DestDir:       destDir,
		JSMangle:      opts.JSMangle,
		KeepCopyright: opts.KeepCopyright,
		CSSPrecision:  opts.CSSPrecision,
	}
	if err := a.history.Add(entry); err != nil {
		log.Printf("写入历史失败: %v", err)
	}
}

// SaveResult 导出单个结果到指定目录。
func (a *App) SaveResult(id, destDir string) (string, error) {
	e, target, err := a.prepareTarget(id, destDir)
	if err != nil {
		return "", err
	}
	if err := writeEntry(e, target); err != nil {
		return "", err
	}
	dir := destDir
	if dir == "" {
		dir = filepath.Dir(e.res.Path)
	}
	a.recordExport(1, 0, e.res.OriginalSize, e.res.MinifiedSize, dir, e.opts)
	return target, nil
}

// SaveAll 批量导出成功的结果，返回成功数量；部分失败会返回聚合错误。
func (a *App) SaveAll(ids []string, destDir string) (int, error) {
	saved := 0
	var original, minified int
	var firstOpts minifier.Options
	actualDir := destDir
	seen := make(map[string]bool)
	var problems []string

	for _, id := range ids {
		e, target, err := a.prepareTarget(id, destDir)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		key := target
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			problems = append(problems, fmt.Sprintf("%s: 目标文件冲突（%s）", e.res.Name, filepath.Base(target)))
			continue
		}
		seen[key] = true

		if err := writeEntry(e, target); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", e.res.Name, err))
			continue
		}
		if saved == 0 {
			firstOpts = e.opts
			if actualDir == "" {
				actualDir = filepath.Dir(e.res.Path)
			}
		}
		saved++
		original += e.res.OriginalSize
		minified += e.res.MinifiedSize
	}

	if saved > 0 {
		a.recordExport(saved, len(problems), original, minified, actualDir, firstOpts)
	}
	if len(problems) > 0 {
		return saved, fmt.Errorf("已导出 %d 个，失败 %d 个：%s", saved, len(problems), strings.Join(problems, "；"))
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

// OpenFolder 选择要压缩的文件夹。
func (a *App) OpenFolder() (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "选择要压缩的文件夹"})
}

// OpenDir 选择导出目录，默认定位到上次导出的目录。
func (a *App) OpenDir() (string, error) {
	defaultDir := ""
	if last := a.history.LastDir(); last != "" {
		if info, err := os.Stat(last); err == nil && info.IsDir() {
			defaultDir = last
		}
	}
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:            "选择导出目录",
		DefaultDirectory: defaultDir,
	})
}

// GetHistory 返回压缩历史（最新在前）。
func (a *App) GetHistory() []history.Entry {
	return a.history.List()
}

// ClearHistory 清空压缩历史。
func (a *App) ClearHistory() error {
	return a.history.Clear()
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
