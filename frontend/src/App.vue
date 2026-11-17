<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import FileList from './components/FileList.vue'
import PreviewPanel from './components/PreviewPanel.vue'
import OptionsDrawer from './components/OptionsDrawer.vue'
import UpdateDialog from './components/UpdateDialog.vue'
import HistoryDialog from './components/HistoryDialog.vue'
import Icon from './components/Icon.vue'
import {
  compress,
  getVersion,
  isWails,
  onFileDrop,
  openDir,
  openFiles,
  openFolder,
  recompress,
  saveAll,
  saveResult,
} from './composables/useBackend'
import { formatRatio, formatSize } from './highlight'
import type { FileResult, MinifyOptions } from './types'

const files = ref<FileResult[]>([])
const selectedId = ref('')
const drawerOpen = ref(false)
const updateOpen = ref(false)
const historyOpen = ref(false)
const version = ref('1.0.0')
const busy = ref(false)
const dragging = ref(false)

const options = reactive<MinifyOptions>({
  JSMangle: true,
  KeepCopyright: true,
  CSSPrecision: 1,
})

const selected = computed(() => files.value.find((f) => f.id === selectedId.value) ?? null)
const okFiles = computed(() => files.value.filter((f) => !f.error))
const totalOriginal = computed(() => okFiles.value.reduce((s, f) => s + f.originalSize, 0))
const totalMinified = computed(() => okFiles.value.reduce((s, f) => s + f.minifiedSize, 0))
const totalRatio = computed(() => (totalOriginal.value ? 1 - totalMinified.value / totalOriginal.value : 0))
const failedCount = computed(() => files.value.filter((f) => f.error).length)
const subtitle = computed(() => {
  if (!files.value.length) return '尚未导入文件'
  if (busy.value) return '正在压缩…'
  const base = `${okFiles.value.length} 个文件 · 共节省 ${formatSize(Math.max(0, totalOriginal.value - totalMinified.value))}`
  return failedCount.value ? `${base} · ${failedCount.value} 个失败` : base
})

let toastTimer: ReturnType<typeof setTimeout> | null = null
const toastText = ref('')
const toastVisible = ref(false)
function toast(text: string) {
  toastText.value = text
  toastVisible.value = true
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toastVisible.value = false), 2200)
}

async function addPaths(paths: string[]) {
  if (!paths.length || busy.value) return
  busy.value = true
  try {
    const batch = await compress(paths, { ...options })
    const results = batch.results
    for (const r of results) {
      const idx = files.value.findIndex((f) => f.path === r.path)
      if (idx >= 0) files.value.splice(idx, 1, r)
      else files.value.push(r)
    }
    const firstOk = results.find((r) => !r.error)
    selectedId.value = firstOk?.id ?? results[0]?.id ?? selectedId.value

    if (!results.length) {
      toast(batch.warnings[0] ?? '没有找到可压缩的文件')
      return
    }
    const parts = [`已处理 ${results.length} 个文件`]
    if (batch.skipped) parts.push(`跳过 ${batch.skipped} 个不支持的文件`)
    if (batch.truncated) parts.push('已达单次导入上限')
    if (batch.warnings.length) {
      const extra = batch.warnings.length > 1 ? ` 等 ${batch.warnings.length} 条` : ''
      parts.push(batch.warnings[0] + extra)
    }
    toast(parts.join(' · '))
  } catch (e: any) {
    toast('压缩失败：' + String(e?.message ?? e))
  } finally {
    busy.value = false
  }
}

async function pickFiles() {
  if (isWails()) {
    const paths = await openFiles()
    await addPaths(paths)
  } else {
    await addPaths([])
  }
}

async function pickFolder() {
  if (!isWails()) {
    toast('浏览器预览模式不支持文件夹')
    return
  }
  const dir = await openFolder()
  if (dir) await addPaths([dir])
}

async function recompressAll() {
  const ids = files.value.map((f) => f.id)
  if (!ids.length) return
  busy.value = true
  try {
    const results = await recompress(ids, { ...options })
    files.value = results
    if (!results.some((r) => r.id === selectedId.value)) {
      selectedId.value = results.find((r) => !r.error)?.id ?? results[0]?.id ?? ''
    }
    toast('已按当前选项重新压缩')
  } catch (e: any) {
    toast('重新压缩失败：' + String(e?.message ?? e))
  } finally {
    busy.value = false
  }
}

function removeFile(id: string) {
  const idx = files.value.findIndex((f) => f.id === id)
  if (idx < 0) return
  files.value.splice(idx, 1)
  if (selectedId.value === id) selectedId.value = files.value[0]?.id ?? ''
}

async function copyResult() {
  const f = selected.value
  if (!f || f.error) return
  try {
    await navigator.clipboard.writeText(f.output)
    toast('已复制压缩结果到剪贴板')
  } catch {
    toast('复制失败，请手动选择代码')
  }
}

async function exportOne() {
  const f = selected.value
  if (!f || f.error) return
  const dir = await openDir()
  if (!dir) return
  try {
    const path = await saveResult(f.id, dir)
    toast(`已导出 ${path.split(/[\\/]/).pop()}`)
  } catch (e: any) {
    toast('导出失败：' + String(e?.message ?? e))
  }
}

async function exportAll() {
  if (!okFiles.value.length) return
  const dir = await openDir()
  if (!dir) return
  try {
    const n = await saveAll(okFiles.value.map((f) => f.id), dir)
    toast(`已导出 ${n} 个文件`)
  } catch (e: any) {
    toast('导出失败：' + String(e?.message ?? e))
  }
}

function applyOptions(opts: MinifyOptions) {
  Object.assign(options, opts)
  drawerOpen.value = false
  void recompressAll()
}

function onDrop(e: DragEvent) {
  dragging.value = false
  const list = Array.from(e.dataTransfer?.files ?? []).map((f) => (f as any).path || f.name)
  if (list.length) void addPaths(list)
}

onMounted(async () => {
  version.value = await getVersion()
  onFileDrop((paths) => void addPaths(paths))
})
</script>

<template>
  <div class="app" @dragover.prevent="dragging = true" @dragleave="dragging = false" @drop.prevent="onDrop">
    <!-- 工具栏 -->
    <header class="toolbar">
      <div class="brand"><span class="brand-icon"><Icon name="bolt" :size="14" /></span>GoMinify</div>
      <div class="divider"></div>
      <div class="tb-title">压缩工作区</div>
      <div class="tb-sub">{{ subtitle }}</div>

      <div class="spacer"></div>
      <button class="btn btn-ghost" @click="historyOpen = true"><Icon name="clock" :size="15" />历史</button>
      <button class="btn" @click="drawerOpen = true"><Icon name="tune" :size="15" />压缩选项</button>
      <button class="ver-chip" title="软件更新" @click="updateOpen = true">
        <span class="dot-ver" :class="{ hot: updateOpen }"></span>v{{ version }}
      </button>
    </header>

    <!-- 主体 -->
    <main class="main">
      <FileList
        v-if="files.length"
        :files="files"
        :selected-id="selectedId"
        @select="selectedId = $event"
        @remove="removeFile"
        @add="pickFiles"
        @add-folder="pickFolder"
      />

      <PreviewPanel v-if="selected" :file="selected" @copy="copyResult" @export="exportOne" />

      <!-- 空状态 -->
      <section v-if="!files.length" class="empty">
        <div class="empty-inner" :class="{ dragging }" @click="pickFiles">
          <div class="empty-icon"><Icon name="upload" :size="28" /></div>
          <div class="empty-title">拖入文件或文件夹，即刻压缩</div>
          <div class="empty-desc">
            支持 JS / CSS / HTML / JSON，可一次拖入多个文件或整个目录<br />
            或 <b>点击选择文件</b> / <b class="link" @click.stop="pickFolder">选择文件夹</b>，结果不会离开你的电脑
          </div>
          <div class="chips">
            <span class="chip">.js</span><span class="chip">.css</span><span class="chip">.html</span><span class="chip">.json</span>
          </div>
          <div class="empty-foot">压缩在本地完成 · 无需联网 · 支持 5 MB 以内单文件</div>
        </div>
      </section>
    </main>

    <!-- 底部状态栏 -->
    <footer class="statusbar">
      <div class="sum">
        <span><b>{{ files.length }}</b> 个文件</span>
        <span class="sep"></span>
        <span>{{ formatSize(totalOriginal) }} → <b>{{ formatSize(totalMinified) }}</b></span>
        <span class="sep"></span>
        <span class="save">总体积减少 {{ formatRatio(totalRatio).replace('-', '') }}</span>
      </div>
      <div class="spacer"></div>
      <button class="btn" :disabled="!files.length || busy" @click="recompressAll">
        <Icon name="refresh" :size="15" />全部压缩
      </button>
      <button class="btn" :disabled="!selected || !!selected.error" @click="exportOne">
        <Icon name="download" :size="15" />导出选中
      </button>
      <button class="btn btn-primary" :disabled="!okFiles.length" @click="exportAll">
        <Icon name="folder" :size="15" />全部导出
      </button>
    </footer>

    <OptionsDrawer :open="drawerOpen" :options="options" @close="drawerOpen = false" @apply="applyOptions" />
    <UpdateDialog :open="updateOpen" :version="version" @close="updateOpen = false" />
    <HistoryDialog :open="historyOpen" @close="historyOpen = false" />

    <div class="toast" :class="{ show: toastVisible }">{{ toastText }}</div>
  </div>
</template>

<style scoped>
.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--win);
  position: relative;
  overflow: hidden;
}
.toolbar {
  height: 54px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  border-bottom: 1px solid var(--border);
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.brand-icon {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: linear-gradient(135deg, #4e8ff7, #37d399);
  color: #0a0d10;
  display: flex;
  align-items: center;
  justify-content: center;
}
.divider {
  width: 1px;
  height: 18px;
  background: var(--border-2);
}
.tb-title {
  font-size: 13px;
  font-weight: 600;
}
.tb-sub {
  font-size: 11.5px;
  color: var(--text-3);
}
.spacer {
  flex: 1;
}
.ver-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--border-2);
  background: var(--panel-2);
  color: var(--text-2);
  font-family: var(--mono);
  font-size: 11px;
  padding: 5px 10px;
  border-radius: 999px;
  cursor: pointer;
}
.ver-chip:hover {
  color: var(--text);
  border-color: #39424f;
}
.dot-ver {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--green);
}
.main {
  flex: 1;
  display: flex;
  min-height: 0;
}
.empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}
.empty-inner {
  width: 100%;
  max-width: 560px;
  text-align: center;
  border: 1.5px dashed var(--border-2);
  border-radius: 14px;
  padding: 56px 40px;
  background: rgba(20, 24, 30, 0.4);
  transition: all 0.2s;
  cursor: pointer;
}
.empty-inner:hover,
.empty-inner.dragging {
  border-color: rgba(78, 143, 247, 0.6);
  background: rgba(78, 143, 247, 0.04);
}
.empty-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 20px;
  border-radius: 18px;
  background: linear-gradient(140deg, rgba(78, 143, 247, 0.18), rgba(55, 211, 153, 0.12));
  border: 1px solid rgba(78, 143, 247, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #7fb0ff;
}
.empty-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}
.empty-desc {
  font-size: 12.5px;
  color: var(--text-2);
  line-height: 1.8;
}
.empty-desc b {
  color: var(--accent);
  font-weight: 500;
}
.empty-desc b.link {
  text-decoration: underline;
  text-underline-offset: 3px;
}
.chips {
  display: flex;
  gap: 6px;
  justify-content: center;
  margin-top: 22px;
  flex-wrap: wrap;
}
.empty-foot {
  margin-top: 22px;
  font-size: 11.5px;
  color: var(--text-3);
}
.statusbar {
  height: 58px;
  flex-shrink: 0;
  border-top: 1px solid var(--border);
  background: var(--panel);
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 10px;
}
.sum {
  font-size: 12px;
  color: var(--text-2);
  display: flex;
  align-items: center;
  gap: 10px;
}
.sum b {
  color: var(--text);
  font-weight: 600;
}
.sep {
  width: 1px;
  height: 14px;
  background: var(--border-2);
}
.save {
  color: var(--green);
  font-family: var(--mono);
  font-weight: 600;
}
</style>