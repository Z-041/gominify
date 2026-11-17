// Package minifier 封装 tdewolff/minify，按文件扩展名路由到对应压缩器。
package minifier

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
)

// Options 是暴露给 UI 的压缩选项。
type Options struct {
	JSMangle      bool // 混淆局部变量名
	KeepCopyright bool // 保留 /*! ... */ 许可注释
	CSSPrecision  int  // CSS 数值精度 0-3
}

// Result 是单个文件的压缩结果。
type Result struct {
	Path         string
	Name         string
	Language     string
	OriginalSize int
	MinifiedSize int
	Ratio        float64 // 体积减少比例，如 0.71 表示减少 71%
	Elapsed      time.Duration
	Output       []byte
	Source       []byte
	Err          error
}

var supportedExt = map[string]string{
	".js":   "js",
	".mjs":  "js",
	".cjs":  "js",
	".css":  "css",
	".html": "html",
	".htm":  "html",
	".json": "json",
}

// DetectLanguage 根据文件名推断语言类型，支持返回 false 表示不支持。
func DetectLanguage(name string) (string, bool) {
	lang, ok := supportedExt[strings.ToLower(filepath.Ext(name))]
	return lang, ok
}

func mimeOf(lang string) string {
	switch lang {
	case "js":
		return "application/javascript"
	case "css":
		return "text/css"
	case "html":
		return "text/html"
	case "json":
		return "application/json"
	}
	return ""
}

func newMinifier(opts Options) *minify.M {
	m := minify.New()
	m.Add("text/css", &css.Minifier{Precision: opts.CSSPrecision})
	m.Add("text/html", &html.Minifier{KeepComments: false, KeepWhitespace: false})
	m.Add("application/javascript", &js.Minifier{KeepVarNames: !opts.JSMangle})
	m.Add("application/json", &json.Minifier{})
	return m
}

var licenseRe = regexp.MustCompile(`(?s)/\*!.*?\*/`)

// Minify 压缩单段内容。name 用于判断语言，src 为原始内容。
func Minify(name string, src []byte, opts Options) ([]byte, error) {
	lang, ok := DetectLanguage(name)
	if !ok {
		return nil, fmt.Errorf("不支持的文件类型: %s", filepath.Ext(name))
	}
	if opts.CSSPrecision < 0 || opts.CSSPrecision > 3 {
		opts.CSSPrecision = 1
	}

	var licenses [][]byte
	// 许可注释前缀只对 JS/CSS 合法；HTML/JSON 前置 /*! ... */ 会破坏文档结构。
	if opts.KeepCopyright && (lang == "js" || lang == "css") {
		licenses = licenseRe.FindAll(src, -1)
	}

	var buf bytes.Buffer
	if err := newMinifier(opts).Minify(mimeOf(lang), &buf, bytes.NewReader(src)); err != nil {
		return nil, fmt.Errorf("%s: %w", lang, err)
	}

	out := buf.Bytes()
	if len(licenses) > 0 {
		prefix := bytes.Join(licenses, []byte("\n"))
		out = append(append(prefix, '\n'), out...)
	}
	return out, nil
}

// MinifyFile 读取并压缩单个文件，错误通过 Result.Err 返回，便于批量处理。
func MinifyFile(path string, opts Options) Result {
	start := time.Now()
	res := Result{Path: path, Name: filepath.Base(path)}

	src, err := os.ReadFile(path)
	if err != nil {
		res.Err = err
		return res
	}
	lang, ok := DetectLanguage(path)
	if !ok {
		res.Err = fmt.Errorf("不支持的文件类型: %s", filepath.Ext(path))
		return res
	}
	res.Language = lang
	res.Source = src
	res.OriginalSize = len(src)

	out, err := Minify(path, src, opts)
	res.Elapsed = time.Since(start)
	if err != nil {
		res.Err = err
		return res
	}
	res.Output = out
	res.MinifiedSize = len(out)
	if res.OriginalSize > 0 {
		res.Ratio = 1 - float64(res.MinifiedSize)/float64(res.OriginalSize)
	}
	return res
}

var (
	lineWordRe = regexp.MustCompile(`line\s+(\d+)`)
	posRe      = regexp.MustCompile(`(?:^|[:\s])(\d+):(\d+):`)
)

// LineOf 尽力从解析错误中提取行号，取不到返回 0。
func LineOf(err error) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	if m := lineWordRe.FindStringSubmatch(msg); len(m) == 2 {
		if n, e := strconv.Atoi(m[1]); e == nil {
			return n
		}
	}
	if m := posRe.FindStringSubmatch(msg); len(m) == 3 {
		if n, e := strconv.Atoi(m[1]); e == nil {
			return n
		}
	}
	return 0
}
