export interface FileResult {
  id: string
  path: string
  rootPath: string
  relPath: string
  name: string
  language: string
  originalSize: number
  minifiedSize: number
  ratio: number
  elapsedMs: number
  output: string
  source: string
  error: string
  line: number
}

export interface BatchResult {
  results: FileResult[]
  skipped: number
  warnings: string[]
  truncated: boolean
}

export interface HistoryEntry {
  time: string
  fileCount: number
  failed: number
  originalTotal: number
  minifiedTotal: number
  destDir: string
  jsMangle: boolean
  keepCopyright: boolean
  cssPrecision: number
}

export interface MinifyOptions {
  JSMangle: boolean
  KeepCopyright: boolean
  CSSPrecision: number
}

export interface UpdateInfo {
  available: boolean
  current: string
  latest: string
  notes: string
  size: number
}

export interface UpdateProgress {
  percent: number
  done: number
  total: number
}
