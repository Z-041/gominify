package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gominify/internal/minifier"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("创建目录: %v", err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatalf("写文件: %v", err)
		}
	}
}

func TestCompressPathsExpandsFolder(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.js":              "var a = 1;",
		"css/b.css":         "a { color: red; }",
		"a.min.js":          "var a=1",
		"node_modules/x.js": "var x = 2;",
	})

	app := NewApp()
	batch := app.CompressPaths([]string{root}, minifier.Options{CSSPrecision: 1})

	if len(batch.Results) != 2 {
		t.Fatalf("结果数 = %d，期望 2（跳过 node_modules 与 .min.js）", len(batch.Results))
	}
	rels := map[string]bool{}
	for _, r := range batch.Results {
		if r.Error != "" {
			t.Fatalf("%s 压缩失败: %s", r.Name, r.Error)
		}
		if r.RootPath == "" || r.RelPath == "" {
			t.Fatalf("%s 缺少 rootPath/relPath: %q/%q", r.Name, r.RootPath, r.RelPath)
		}
		rels[r.RelPath] = true
	}
	if !rels["a.js"] || !rels["css/b.css"] {
		t.Fatalf("相对路径不完整: %v", rels)
	}
}

func TestSaveAllKeepsDirectoryStructure(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"a.js":      "var a = 1;",
		"sub/c.css": "c { color: blue; }",
	})

	app := NewApp()
	batch := app.CompressPaths([]string{root}, minifier.Options{})
	ids := make([]string, 0, len(batch.Results))
	for _, r := range batch.Results {
		ids = append(ids, r.ID)
	}

	dest := t.TempDir()
	n, err := app.SaveAll(ids, dest)
	if err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	if n != len(ids) {
		t.Fatalf("导出数量 = %d，期望 %d", n, len(ids))
	}

	base := filepath.Base(root)
	for _, name := range []string{"a.min.js", filepath.Join("sub", "c.min.css")} {
		target := filepath.Join(dest, base, name)
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("缺少导出文件 %s: %v", target, err)
		}
	}
}

func TestSaveAllReportsConflict(t *testing.T) {
	tmp := t.TempDir()
	rootA := filepath.Join(tmp, "a", "src")
	rootB := filepath.Join(tmp, "b", "src")
	writeFiles(t, rootA, map[string]string{"x.js": "var a = 1;"})
	writeFiles(t, rootB, map[string]string{"x.js": "var b = 2;"})

	app := NewApp()
	batch := app.CompressPaths([]string{rootA, rootB}, minifier.Options{})
	if len(batch.Results) != 2 {
		t.Fatalf("结果数 = %d，期望 2", len(batch.Results))
	}
	ids := []string{batch.Results[0].ID, batch.Results[1].ID}

	n, err := app.SaveAll(ids, t.TempDir())
	if err == nil {
		t.Fatal("同名根目录导出应返回冲突错误")
	}
	if n != 1 {
		t.Fatalf("成功数量 = %d，期望 1", n)
	}
	if !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("错误信息未提示冲突: %v", err)
	}
}

func TestRecompressKeepsIDAndStructure(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"a.js": "var a = 1;"})

	app := NewApp()
	batch := app.CompressPaths([]string{root}, minifier.Options{JSMangle: true})
	first := batch.Results[0]

	out := app.Recompress([]string{first.ID}, minifier.Options{JSMangle: false})
	if len(out) != 1 {
		t.Fatalf("Recompress 结果数 = %d，期望 1", len(out))
	}
	got := out[0]
	if got.ID != first.ID {
		t.Fatalf("id 变化: %q -> %q", first.ID, got.ID)
	}
	if got.RootPath != first.RootPath || got.RelPath != first.RelPath {
		t.Fatalf("目录结构丢失: %q/%q -> %q/%q", first.RootPath, first.RelPath, got.RootPath, got.RelPath)
	}
	if got.Error != "" || got.Output == "" {
		t.Fatalf("重新压缩失败: %+v", got)
	}
}

func TestForgetRemovesResult(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"a.js": "var a = 1;"})

	app := NewApp()
	batch := app.CompressPaths([]string{filepath.Join(root, "a.js")}, minifier.Options{})
	id := batch.Results[0].ID

	app.Forget([]string{id})
	if _, err := app.SaveResult(id, t.TempDir()); err == nil {
		t.Fatal("Forget 之后不应还能导出")
	}
}

func TestSaveResultSingleFile(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"solo.css": "a { color: red; }"})

	app := NewApp()
	batch := app.CompressPaths([]string{filepath.Join(root, "solo.css")}, minifier.Options{})
	target, err := app.SaveResult(batch.Results[0].ID, t.TempDir())
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}
	if filepath.Base(target) != "solo.min.css" {
		t.Fatalf("导出文件名 = %q，期望 solo.min.css", filepath.Base(target))
	}
}
