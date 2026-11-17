package minifier

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// MaxFileSize 是单文件大小上限，避免大文件拖慢界面。
const MaxFileSize = 5 << 20

// MaxExpandFiles 是一次导入允许的最大文件数。
const MaxExpandFiles = 1000

// maxExpandBytes 是一次导入的原始内容预算（source + output 约 2 倍）。
const maxExpandBytes = 64 << 20

// FileRef 指向一个待压缩文件。Root 为导入时传入的目录，Rel 为相对该目录的路径；
// 直接导入的单文件 Root、Rel 均为空。
type FileRef struct {
	Path string
	Root string
	Rel  string
}

// Expansion 是一次路径展开的结果。Problems 只做提示，不影响其余文件。
type Expansion struct {
	Refs      []FileRef
	Problems  []string
	Skipped   int
	Truncated bool
}

// skipDirs 是递归时直接跳过的目录名（小写）。
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"vendor":       true,
}

// Expand 把文件/目录混合的路径展开成待压缩文件列表。
// maxFiles <= 0 时使用 MaxExpandFiles。
func Expand(paths []string, maxFiles int) Expansion {
	var out Expansion
	if maxFiles <= 0 {
		maxFiles = MaxExpandFiles
	}

	seen := make(map[string]bool)
	var budget int64

	add := func(ref FileRef, size int64) bool {
		ref.Path = resolvePath(ref.Path)
		key := pathKey(ref.Path)
		if seen[key] {
			return true
		}
		if len(out.Refs) >= maxFiles || budget+size*2 > maxExpandBytes {
			out.Truncated = true
			return false
		}
		seen[key] = true
		budget += size * 2
		out.Refs = append(out.Refs, ref)
		return true
	}

	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			out.Problems = append(out.Problems, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		abs = filepath.Clean(abs)

		info, err := os.Stat(abs)
		if err != nil {
			out.Problems = append(out.Problems, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		if !info.IsDir() {
			if !isInputFile(abs) {
				out.Skipped++
				continue
			}
			if info.Size() > MaxFileSize {
				out.Problems = append(out.Problems, fmt.Sprintf("%s: 文件超过 5 MB，已跳过", abs))
				continue
			}
			if !add(FileRef{Path: abs}, info.Size()) {
				break
			}
			continue
		}

		root := abs
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			root = resolved
		}

		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				out.Problems = append(out.Problems, fmt.Sprintf("%s: %v", path, err))
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if path != root && skipDirs[strings.ToLower(d.Name())] {
					return fs.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() || !isInputFile(path) {
				if !d.Type().IsRegular() {
					return nil
				}
				out.Skipped++
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				out.Problems = append(out.Problems, fmt.Sprintf("%s: %v", path, err))
				return nil
			}
			if fi.Size() > MaxFileSize {
				out.Problems = append(out.Problems, fmt.Sprintf("%s: 文件超过 5 MB，已跳过", path))
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil || !validRel(rel) {
				out.Problems = append(out.Problems, fmt.Sprintf("%s: 相对路径无效，已跳过", path))
				return nil
			}
			if !add(FileRef{Path: path, Root: root, Rel: filepath.ToSlash(rel)}, fi.Size()) {
				return fs.SkipAll
			}
			return nil
		})
		if walkErr != nil && walkErr != fs.SkipAll {
			out.Problems = append(out.Problems, fmt.Sprintf("%s: %v", root, walkErr))
		}
		if out.Truncated {
			break
		}
	}
	return out
}

// isInputFile 判断是否是可压缩的输入文件，并排除已压缩产物（*.min.*），
// 避免导出目录被再次拖入时重复收录。
func isInputFile(path string) bool {
	if _, ok := DetectLanguage(path); !ok {
		return false
	}
	return !strings.Contains(strings.ToLower(filepath.Base(path)), ".min.")
}

// validRel 校验相对路径不会逃逸出根目录。
func validRel(rel string) bool {
	if rel == "" || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	clean := filepath.Clean(rel)
	return clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

// resolvePath 尽量解析符号链接/短路径，保证去重键一致。
func resolvePath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// pathKey 生成去重键。Windows 文件系统大小写不敏感，需要折叠；
// 其他平台保留大小写（A.js 与 a.js 是不同文件）。
func pathKey(p string) string {
	key := filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}
