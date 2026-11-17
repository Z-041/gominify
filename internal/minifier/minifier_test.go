package minifier

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	cases := map[string]string{
		"app.js":        "js",
		"vendor.min.js": "js",
		"style.CSS":     "css",
		"index.html":    "html",
		"data.json":     "json",
	}
	for name, want := range cases {
		got, ok := DetectLanguage(name)
		if !ok || got != want {
			t.Errorf("DetectLanguage(%q) = %q, %v; want %q, true", name, got, ok, want)
		}
	}
	if _, ok := DetectLanguage("readme.txt"); ok {
		t.Error("readme.txt should not be supported")
	}
}

func TestMinifyCSSRemovesWhitespaceAndComments(t *testing.T) {
	src := []byte("/* header */\n.card {\n  color: #ffffff;\n  margin: 0 0 0 0;\n}\n")
	out, err := Minify("a.css", src, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) >= len(src) {
		t.Errorf("output not smaller: %d >= %d", len(out), len(src))
	}
	s := string(out)
	if strings.Contains(s, "/*") {
		t.Errorf("comment not removed: %q", s)
	}
	if !strings.Contains(s, ".card{") {
		t.Errorf("selector whitespace not compacted: %q", s)
	}
	if !strings.Contains(s, "color:#fff") {
		t.Errorf("#ffffff not shortened to #fff: %q", s)
	}
}

func TestMinifyJSRemovesComments(t *testing.T) {
	src := []byte("// sum two numbers\nfunction add(a, b) {\n  return a + b; // inline\n}\n")
	out, err := Minify("a.js", src, Options{JSMangle: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(out)
	if len(out) >= len(src) {
		t.Errorf("output not smaller: %d >= %d", len(out), len(src))
	}
	if strings.Contains(s, "// sum two numbers") || strings.Contains(s, "// inline") {
		t.Errorf("comment not removed: %q", s)
	}
	if !strings.Contains(s, "add") {
		t.Errorf("function name lost: %q", s)
	}
}

func TestMinifyJSON(t *testing.T) {
	out, err := Minify("d.json", []byte("{ \"a\": 1, \"b\": [ 1, 2 ] }"), Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "{\"a\":1,\"b\":[1,2]}" {
		t.Errorf("unexpected json minify result: %q", out)
	}
}

func TestMinifyInvalidJSReturnsError(t *testing.T) {
	_, err := Minify("bad.js", []byte("function f( {"), Options{})
	if err == nil {
		t.Fatal("expected error for invalid js, got nil")
	}
}

func TestUnsupportedExtension(t *testing.T) {
	if _, err := Minify("a.txt", []byte("x"), Options{}); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestLineOf(t *testing.T) {
	if got := LineOf(errors.New("js: parse error: line 118: unexpected token")); got != 118 {
		t.Errorf("LineOf = %d; want 118", got)
	}
	if got := LineOf(errors.New("boom")); got != 0 {
		t.Errorf("LineOf without line info = %d; want 0", got)
	}
}

func TestKeepCopyrightOnlyForJSAndCSS(t *testing.T) {
	jsSrc := []byte("/*! MIT License */\nvar a = 1;\n")
	jsOut, err := Minify("a.js", jsSrc, Options{KeepCopyright: true})
	if err != nil {
		t.Fatalf("js: %v", err)
	}
	if !strings.HasPrefix(string(jsOut), "/*! MIT License */") {
		t.Errorf("js 许可注释丢失: %q", jsOut)
	}

	htmlSrc := []byte("<script>/*! MIT License */var a=1</script>")
	htmlOut, err := Minify("a.html", htmlSrc, Options{KeepCopyright: true})
	if err != nil {
		t.Fatalf("html: %v", err)
	}
	if strings.HasPrefix(string(htmlOut), "/*!") {
		t.Errorf("html 不应前置 JS 注释: %q", htmlOut)
	}

	jsonSrc := []byte(`{"note":"/*! not a comment */"}`)
	jsonOut, err := Minify("a.json", jsonSrc, Options{KeepCopyright: true})
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if !json.Valid(jsonOut) {
		t.Errorf("json 被许可注释破坏: %q", jsonOut)
	}
}

func TestMinifyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "style.css")
	src := []byte("body {\n  color: #000000;\n}\n")
	if err := os.WriteFile(p, src, 0o644); err != nil {
		t.Fatal(err)
	}
	res := MinifyFile(p, Options{})
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.OriginalSize != len(src) || res.MinifiedSize >= res.OriginalSize {
		t.Errorf("bad sizes: original=%d minified=%d", res.OriginalSize, res.MinifiedSize)
	}
	if res.Language != "css" {
		t.Errorf("language = %q; want css", res.Language)
	}
}
