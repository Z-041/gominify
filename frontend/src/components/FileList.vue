<script setup lang="ts">
import Icon from './Icon.vue'
import { formatSize, formatRatio } from '../highlight'
import type { FileResult } from '../types'

defineProps<{ files: FileResult[]; selectedId: string }>()
const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'remove', id: string): void
  (e: 'add'): void
}>()

const ext = (f: FileResult) => (f.language === 'js' ? 'JS' : f.language === 'css' ? 'CSS' : f.language === 'html' ? 'HTML' : 'JSON')
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-head">
      待处理文件
      <span class="count">{{ files.length }}</span>
      <button class="add" title="添加文件" @click="emit('add')"><Icon name="add" :size="17" /></button>
    </div>

    <div class="file-list">
      <div
        v-for="f in files"
        :key="f.id"
        class="file-row"
        :class="{ active: f.id === selectedId, error: !!f.error }"
        @click="emit('select', f.id)"
      >
        <div class="file-icon" :class="f.error ? 'err' : f.language">
          <Icon v-if="f.error" name="error" :size="15" />
          <template v-else>{{ ext(f) }}</template>
        </div>
        <div class="file-meta">
          <div class="file-name">{{ f.name }}</div>
          <div v-if="!f.error" class="file-sub">
            {{ formatSize(f.originalSize) }} <span class="arrow">→</span>
            <span class="after">{{ formatSize(f.minifiedSize) }}</span>
          </div>
          <div v-else class="file-sub">
            <span class="fail">压缩失败{{ f.line ? ` · 第 ${f.line} 行` : '' }}</span>
          </div>
        </div>
        <div class="rate" :class="{ err: !!f.error }">{{ f.error ? '失败' : formatRatio(f.ratio) }}</div>
        <button class="icon-btn remove" title="移除" @click.stop="emit('remove', f.id)">
          <Icon name="close" :size="14" />
        </button>
      </div>
    </div>

    <div class="sidebar-foot">
      <div class="drop-hint" @click="emit('add')">拖入文件或点击添加</div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 316px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  background: var(--panel);
}
.sidebar-head {
  height: 42px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 0 14px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  color: var(--text-2);
  letter-spacing: 0.03em;
}
.count {
  margin-left: 6px;
  background: var(--panel-3);
  color: var(--text-2);
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 10px;
}
.add {
  margin-left: auto;
  width: 24px;
  height: 24px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-2);
  cursor: pointer;
  background: transparent;
  border: 0;
}
.add:hover {
  background: var(--panel-3);
  color: var(--text);
}
.file-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
}
.file-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 8px;
  border-radius: var(--r-sm);
  cursor: pointer;
  border: 1px solid transparent;
  margin-bottom: 2px;
}
.file-row:hover {
  background: var(--panel-2);
}
.file-row.active {
  background: var(--accent-dim);
  border-color: rgba(78, 143, 247, 0.35);
}
.file-row.error.active {
  background: var(--red-dim);
  border-color: rgba(240, 96, 93, 0.3);
}
.file-icon {
  width: 30px;
  height: 30px;
  border-radius: 7px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--mono);
  font-size: 9.5px;
  font-weight: 700;
}
.file-icon.js {
  background: rgba(232, 197, 71, 0.13);
  color: var(--yellow);
  border: 1px solid rgba(232, 197, 71, 0.25);
}
.file-icon.css {
  background: rgba(78, 143, 247, 0.13);
  color: #6fa8ff;
  border: 1px solid rgba(78, 143, 247, 0.25);
}
.file-icon.html,
.file-icon.json {
  background: rgba(55, 211, 153, 0.12);
  color: var(--green);
  border: 1px solid rgba(55, 211, 153, 0.24);
}
.file-icon.err {
  background: rgba(240, 96, 93, 0.15);
  color: var(--red);
  border: 1px solid rgba(240, 96, 93, 0.3);
}
.file-meta {
  flex: 1;
  min-width: 0;
}
.file-name {
  font-size: 12.5px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.file-sub {
  font-size: 11px;
  color: var(--text-3);
  margin-top: 2px;
  display: flex;
  align-items: center;
  gap: 5px;
  font-family: var(--mono);
}
.arrow {
  color: #3e4854;
}
.after {
  color: var(--text-2);
}
.fail {
  color: var(--red);
  font-family: inherit;
}
.rate {
  font-size: 11px;
  font-weight: 600;
  color: var(--green);
  background: var(--green-dim);
  border-radius: 4px;
  padding: 2px 5px;
  flex-shrink: 0;
  font-family: var(--mono);
}
.rate.err {
  color: var(--red);
  background: var(--red-dim);
}
.remove {
  width: 22px;
  height: 22px;
  opacity: 0;
}
.file-row:hover .remove {
  opacity: 1;
}
.sidebar-foot {
  flex-shrink: 0;
  border-top: 1px solid var(--border);
  padding: 12px 14px;
}
.drop-hint {
  border: 1px dashed var(--border-2);
  border-radius: var(--r-sm);
  padding: 9px;
  text-align: center;
  font-size: 11.5px;
  color: var(--text-3);
  cursor: pointer;
}
.drop-hint:hover {
  border-color: var(--accent);
  color: var(--text-2);
}
</style>