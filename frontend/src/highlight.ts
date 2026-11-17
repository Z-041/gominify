const escapeHtml = (s: string): string =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/** CSS：按 选择器 { 声明; } 结构着色，够用于压缩结果预览。 */
function highlightCSS(code: string): string {
  let out = ''
  let buf = ''
  let inDecl = false

  const flushSelector = () => {
    if (!buf) return
    out += `<span class="tok-sel">${escapeHtml(buf)}</span>`
    buf = ''
  }
  const flushDecl = () => {
    if (!buf.trim()) {
      buf = ''
      return
    }
    const idx = buf.indexOf(':')
    if (idx > 0) {
      out += `<span class="tok-prop">${escapeHtml(buf.slice(0, idx))}</span><span class="tok-punct">:</span><span class="tok-val">${escapeHtml(buf.slice(idx + 1))}</span>`
    } else {
      out += escapeHtml(buf)
    }
    buf = ''
  }

  for (const ch of code) {
    if (ch === '{') {
      flushSelector()
      out += '<span class="tok-punct">{</span>'
      inDecl = true
    } else if (ch === '}') {
      if (inDecl) flushDecl()
      out += '<span class="tok-punct">}</span>'
      inDecl = false
    } else if (ch === ';' && inDecl) {
      flushDecl()
      out += '<span class="tok-punct">;</span>'
    } else {
      buf += ch
    }
  }
  if (buf) out += escapeHtml(buf)
  return out
}

const JS_KEYWORDS =
  'function|return|const|let|var|if|else|for|while|of|in|new|await|async|export|import|from|class|try|catch|throw|typeof|instanceof|this|true|false|null|undefined'

const jsRe = new RegExp(
  `(\\/\\/[^\\n]*|\\/\\*[\\s\\S]*?\\*\\/)|("(?:[^"\\\\]|\\\\.)*"|'(?:[^'\\\\]|\\\\.)*'|\`(?:[^\`\\\\]|\\\\.)*\`)|\\b(${JS_KEYWORDS})\\b|\\b(\\d+(?:\\.\\d+)?)\\b`,
  'g',
)

function highlightJS(code: string): string {
  let out = ''
  let last = 0
  for (const m of code.matchAll(jsRe)) {
    const i = m.index ?? 0
    out += escapeHtml(code.slice(last, i))
    const [raw] = m
    if (m[1]) out += `<span class="tok-comment">${escapeHtml(raw)}</span>`
    else if (m[2]) out += `<span class="tok-str">${escapeHtml(raw)}</span>`
    else if (m[3]) out += `<span class="tok-key">${escapeHtml(raw)}</span>`
    else out += `<span class="tok-num">${escapeHtml(raw)}</span>`
    last = i + raw.length
  }
  out += escapeHtml(code.slice(last))
  return out
}

export function highlight(code: string, lang: string): string {
  if (lang === 'js') return highlightJS(code)
  if (lang === 'css') return highlightCSS(code)
  return escapeHtml(code)
}

export function formatSize(bytes: number): string {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

export function formatRatio(r: number): string {
  if (!r || r <= 0) return '0%'
  return `-${Math.round(r * 100)}%`
}