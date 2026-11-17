package minifier

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// makeTree 在临时目录下按映射创建文件，值即文件内容。
func makeTree(t *testing.T, root string, files map[string]string) {
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

func rels(exp Expansion) []string {
	out := make([]string, 0, len(exp.Refs))
	for _, r := range exp.Refs {
		out = append(out, r.Rel)
	}
	sort.Strings(out)
	return out
}

func TestExpandFolderKeepsStructure(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, map[string]string{
		"a.js":                "var a=1",
		"css/b.css":           "a{color:red}",
		"sub/deep/c.html":     "<p> hi </p>",
		"node_modules/x.js":   "var x=1",
		"a.min.js":            "var a=1",
		"readme.md":           "# doc",
		"sub/deep/skip.txt":   "x",
		"sub/deep/data.json":  "{\"a\": 1}",
		"sub/.git/config.css": "a{color:blue}",
	})

	exp := Expand([]string{root}, 0)
	got := rels(exp)
	want := []string{"a.js", "css/b.css", "sub/deep/c.html", "sub/deep/data.json"}
	if len(got) != len(want) {
		t.Fatalf("文件数 = %d (%v)，期望 %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rel[%d] = %q，期望 %q（完整: %v）", i, got[i], want[i], got)
		}
	}
	if exp.Skipped == 0 {
		t.Fatal("Skipped 应大于 0（md/txt/min.js 等）")
	}
	if exp.Truncated {
		t.Fatal("不应截断")
	}
	for _, r := range exp.Refs {
		if r.Root == "" {
			t.Fatalf("%s 的 Root 为空", r.Path)
		}
		if !filepath.IsAbs(r.Path) {
			t.Fatalf("%s 不是绝对路径", r.Path)
		}
	}
}

func TestExpandMissingPathIsProblemOnly(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, map[string]string{"a.js": "var a=1"})

	exp := Expand([]string{filepath.Join(root, "nope.js"), filepath.Join(root, "a.js")}, 0)
	if len(exp.Refs) != 1 {
		t.Fatalf("Refs = %d，期望 1", len(exp.Refs))
	}
	if len(exp.Problems) != 1 {
		t.Fatalf("Problems = %v，期望 1 条", exp.Problems)
	}
	if ref := exp.Refs[0]; ref.Root != "" || ref.Rel != "" {
		t.Fatalf("单文件 Root/Rel 应为空，实际 %q/%q", ref.Root, ref.Rel)
	}
}

func TestExpandDeduplicatesOverlappingPaths(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, map[string]string{"a.js": "var a=1", "b.css": "a{color:red}"})

	exp := Expand([]string{root, root, filepath.Join(root, "a.js")}, 0)
	if len(exp.Refs) != 2 {
		t.Fatalf("Refs = %d，期望 2（去重后）", len(exp.Refs))
	}
}

func TestExpandTruncatesAtLimit(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, map[string]string{
		"a.js": "var a=1", "b.js": "var b=1", "c.js": "var c=1",
	})
	exp := Expand([]string{root}, 2)
	if len(exp.Refs) != 2 {
		t.Fatalf("Refs = %d，期望 2", len(exp.Refs))
	}
	if !exp.Truncated {
		t.Fatal("应标记 Truncated")
	}
}

func TestExpandSkipsMinifiedProducts(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, map[string]string{
		"app.js":       "var a=1",
		"app.min.js":   "var a=1",
		"site.min.css": "a{color:red}",
	})
	exp := Expand([]string{root}, 0)
	if len(exp.Refs) != 1 || exp.Refs[0].Rel != "app.js" {
		t.Fatalf("只应收录 app.js，实际 %v", rels(exp))
	}
}

func TestValidRel(t *testing.T) {
	cases := map[string]bool{
		"a.js":                          true,
		filepath.FromSlash("css/b.css"): true,
		"..":                            false,
		"../x.js":                       false,
		filepath.Join("a", "b.js"):      true,
		filepath.Join("..", "x.js"):     false,
	}
	for rel, want := range cases {
		if got := validRel(rel); got != want {
			t.Errorf("validRel(%q) = %v，期望 %v", rel, got, want)
		}
	}
}
