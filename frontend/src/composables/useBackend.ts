import type { FileResult, MinifyOptions, UpdateInfo, UpdateProgress } from '../types'

/** Wails 注入的运行时对象；浏览器里直接打开时为空，走 mock。 */
const rt = () => (window as any).runtime
const api = () => (window as any).go?.main?.App

export const isWails = (): boolean => !!api()

export function onFileDrop(cb: (paths: string[]) => void): void {
  const r = rt()
  if (r?.OnFileDrop) {
    r.OnFileDrop((_x: number, _y: number, paths: string[]) => cb(paths), true)
  }
}

export function onUpdateProgress(cb: (p: UpdateProgress) => void): void {
  const r = rt()
  if (r?.EventsOn) {
    r.EventsOn('update:progress', (p: UpdateProgress) => cb(p))
  }
}

export async function openFiles(): Promise<string[]> {
  if (!api()) return []
  try {
    return (await api().OpenFiles()) ?? []
  } catch {
    return []
  }
}

export async function openDir(): Promise<string> {
  if (!api()) return ''
  try {
    return (await api().OpenDir()) ?? ''
  } catch {
    return ''
  }
}

export async function getVersion(): Promise<string> {
  if (!api()) return '1.0.0'
  try {
    return await api().GetVersion()
  } catch {
    return '1.0.0'
  }
}

export async function compress(paths: string[], opts: MinifyOptions): Promise<FileResult[]> {
  if (api()) {
    return (await api().CompressFiles(paths, opts)) ?? []
  }
  return mockCompress(paths)
}

export async function saveResult(id: string, dir: string): Promise<string> {
  if (!api()) return `${dir || '导出目录'}/${id}`
  return await api().SaveResult(id, dir)
}

export async function saveAll(ids: string[], dir: string): Promise<number> {
  if (!api()) return ids.length
  return await api().SaveAll(ids, dir)
}

export async function checkUpdate(): Promise<UpdateInfo> {
  if (!api()) {
    await sleep(600)
    return { available: true, current: '1.0.0', latest: '1.1.0', notes: '新增文件夹模式，修复大文件预览卡顿。', size: 12 * 1024 * 1024 }
  }
  return await api().CheckUpdate()
}

export async function applyUpdate(): Promise<void> {
  if (!api()) {
    for (let i = 0; i <= 100; i += 10) {
      emitMockProgress(i)
      await sleep(120)
    }
    return
  }
  await api().ApplyUpdate()
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

let mockCb: ((p: UpdateProgress) => void) | null = null
export function setMockProgressHandler(cb: (p: UpdateProgress) => void) {
  mockCb = cb
}
function emitMockProgress(percent: number) {
  mockCb?.({ percent, done: percent * 120000, total: 12 * 1024 * 1024 })
}

/* ---------------- 浏览器预览用的示例数据 ---------------- */

const cssOriginal = `/* Design tokens */
:root {
  --bg: #090B0E;
  --fg: #E7EAEE;
  --accent: #4E8FF7;
}

/* base */
*,
*::after,
*::before {
  box-sizing: border-box;
  margin: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  background: var(--bg);
  color: var(--fg);
}
`

const cssMinified =
  ':root{--bg:#090b0e;--fg:#e7eaee;--accent:#4e8ff7}*,:after,:before{box-sizing:border-box;margin:0}body{font-family:-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,sans-serif;font-size:14px;line-height:1.5;background:var(--bg);color:var(--fg)}'

const jsMinified =
  'function add(n,t){return n+t}const cache=new Map;async function load(n){if(cache.has(n))return cache.get(n);const t=await fetch(n),e=await t.json();return cache.set(n,e),e}export{add,load};'

async function mockCompress(paths: string[]): Promise<FileResult[]> {
  await sleep(280)
  const list = paths.length ? paths : ['app.css', 'vendor.js']
  return list.map((p, i) => {
    const isJS = /\.m?js$/i.test(p)
    const output = isJS ? jsMinified : cssMinified
    const original = isJS ? jsMinified.length * 2.6 : cssOriginal.length
    const name = p.split(/[\\/]/).pop() || p
    return {
      id: `mock-${i}-${Date.now()}`,
      path: p,
      name,
      language: isJS ? 'js' : 'css',
      originalSize: Math.round(original),
      minifiedSize: output.length,
      ratio: 1 - output.length / original,
      elapsedMs: isJS ? 41 : 23,
      output,
      source: isJS ? jsMinified : cssOriginal,
      error: '',
      line: 0,
    }
  })
}