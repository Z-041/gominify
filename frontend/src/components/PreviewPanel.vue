<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { formatSize, formatRatio, highlight } from '../highlight'
import type { FileResult } from '../types'

const props = defineProps<{ file: FileResult | null }>()
const emit = defineEmits<{ (e: 'copy'): void; (e: 'export'): void }>()

const tab = ref<'out' | 'diff'>('out')

watch(
  () => props.file?.id,
  () => {
    tab.value = 'out'
  },
)

const outHtml = computed(() => (props.file ? highlight(props.file.output, props.file.language) : ''))
const srcHtml = computed(() => (props.file ? highlight(props.file.source || '', props.file.language) : ''))

const errorLines = computed(() => {
  const f = props.file
  if (!f || !f.error) return []
  const lines = (f.source || '').split('\n')
  if (!f.line || !lines.length) return []
  const start = Math.max(0, f.line - 3)
  const end = Math.min(lines.length, f.line + 2)
  return lines.slice(start, end).map((text, i) => ({ no: start + i + 1, text, bad: start + i + 1 === f.line }))
})
</script>

<template>
  <section v-if="file" class="preview">
    <div class="preview-head">
      <div class="tabs">
        <button class="tab" :class="{ active: tab === 'out' }" @click="tab = 'out'">压缩结果</button>
        <button class="tab" :class="{ active: tab === 'diff' }" :disabled="!!file.error" @click="tab = 'diff'">前后对比</button>
      </div>
      <div class="preview-actions">
        <button class="icon-btn" title="复制结果" :disabled="!!file.error" @click="emit('copy')"><Icon name="copy" :size="16" /></button>
        <button class="icon-btn" title="导出" :disabled="!!file.error" @click="emit('export')"><Icon name="download" :size="16" /></button>
      </div>
    </div>

    <!-- 成功：单栏结果 -->
    <div v-if="!file.error && tab === 'out'" class="code-wrap">
      <div class="code-head">
        <span class="dot"></span>
        {{ file.name }} · {{ formatSize(file.minifiedSize) }} · 压缩耗时 {{ file.elapsedMs }}ms
      </div>
      <div class="code">
        <div class="gutter">1</div>
        <div class="code-body" v-html="outHtml"></div>
      </div>
    </div>

    <!-- 成功：双栏对比 -->
    <div v-else-if="!file.error" class="diff-grid">
      <div class="diff-col">
        <div class="diff-col-head"><b>原始</b><span class="size">{{ formatSize(file.originalSize) }}</span></div>
        <div class="diff-body">
          <div class="code"><div class="code-body" v-html="srcHtml"></div></div>
        </div>
      </div>
      <div class="diff-col">
        <div class="diff-col-head">
          <span class="dot"></span><b>压缩后</b>
          <span class="size">{{ formatSize(file.minifiedSize) }} · {{ formatRatio(file.ratio) }}</span>
        </div>
        <div class="diff-body">
          <div class="code"><div class="code-body" v-html="outHtml"></div></div>
        </div>
      </div>
    </div>

    <!-- 失败 -->
    <div v-else class="code-wrap">
      <div class="error-panel">
        <div class="error-head">
          <Icon name="error" :size="18" />
          <b>{{ file.name }} 压缩失败</b>
          <span class="err-code">{{ file.line ? `第 ${file.line} 行` : '解析错误' }}</span>
        </div>
        <div v-if="errorLines.length" class="error-line">
          <div v-for="l in errorLines" :key="l.no" :class="{ badline: l.bad }">
            <span class="ln">{{ l.no }}</span><span :class="{ bad: l.bad }">{{ l.text }}</span>
          </div>
        </div>
        <div class="error-tip">
          {{ file.error }}<br />
          该文件已跳过，其余文件不受影响。修改源码后可重新拖入，或从列表移除该文件。
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.preview {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #0e1216;
}
.preview-head {
  height: 42px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--border);
  padding: 0 14px;
  gap: 4px;
}
.tabs {
  display: flex;
  gap: 2px;
}
.tab {
  font-family: inherit;
  font-size: 12px;
  color: var(--text-3);
  padding: 6px 10px;
  border-radius: 5px;
  cursor: pointer;
  border: 1px solid transparent;
  background: transparent;
}
.tab:hover {
  color: var(--text-2);
}
.tab.active {
  color: var(--text);
  background: var(--panel-2);
  border-color: var(--border);
}
.tab:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.preview-actions {
  margin-left: auto;
  display: flex;
  gap: 2px;
}
.preview-actions .icon-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.code-wrap {
  flex: 1;
  overflow: auto;
  min-height: 0;
}
.code-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 11px;
  color: var(--text-3);
  border-bottom: 1px solid rgba(35, 42, 51, 0.6);
  font-family: var(--mono);
  background: #0c1014;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--green);
}
.code {
  font-family: var(--mono);
  font-size: 12.5px;
  line-height: 1.85;
  color: #c8d0da;
  padding: 14px 18px 30px;
  display: flex;
}
.gutter {
  color: #3a434f;
  text-align: right;
  user-select: none;
  padding-right: 16px;
  flex-shrink: 0;
  font-size: 11.5px;
}
.code-body {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
  word-break: break-all;
  user-select: text;
}
.diff-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 0;
}
.diff-col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
}
.diff-col:last-child {
  border-right: 0;
}
.diff-col-head {
  padding: 7px 16px;
  font-size: 11px;
  color: var(--text-3);
  font-family: var(--mono);
  border-bottom: 1px solid rgba(35, 42, 51, 0.6);
  background: #0c1014;
  display: flex;
  align-items: center;
  gap: 6px;
}
.diff-col-head b {
  color: var(--text-2);
  font-weight: 500;
}
.size {
  margin-left: auto;
}
.diff-body {
  flex: 1;
  overflow: auto;
  padding: 12px 14px 24px;
}
.diff-body .code {
  padding: 0;
  font-size: 12px;
}
.error-panel {
  margin: 20px;
  border: 1px solid rgba(240, 96, 93, 0.35);
  background: var(--red-dim);
  border-radius: var(--r-md);
  padding: 18px 20px;
}
.error-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--red);
}
.error-head b {
  font-size: 13.5px;
  color: var(--text);
}
.err-code {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-3);
  font-family: var(--mono);
}
.error-line {
  font-family: var(--mono);
  font-size: 12px;
  color: #c9a0a0;
  background: rgba(0, 0, 0, 0.25);
  border-radius: 6px;
  padding: 10px 12px;
  line-height: 1.9;
  overflow-x: auto;
  white-space: pre;
}
.ln {
  color: #6b4b4b;
  margin-right: 14px;
  display: inline-block;
  width: 26px;
  text-align: right;
}
.bad {
  color: #ff9a97;
}
.badline {
  background: rgba(240, 96, 93, 0.08);
}
.error-tip {
  font-size: 12px;
  color: var(--text-2);
  margin-top: 12px;
  line-height: 1.7;
  user-select: text;
}
</style>