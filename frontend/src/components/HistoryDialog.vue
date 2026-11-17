<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { clearHistory, getHistory } from '../composables/useBackend'
import { formatRatio, formatSize } from '../highlight'
import type { HistoryEntry } from '../types'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const entries = ref<HistoryEntry[]>([])
const loading = ref(false)
const confirming = ref(false)

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    confirming.value = false
    loading.value = true
    try {
      entries.value = await getHistory()
    } finally {
      loading.value = false
    }
  },
)

const totalSaved = computed(() =>
  entries.value.reduce((sum, e) => sum + Math.max(0, e.originalTotal - e.minifiedTotal), 0),
)

const ratioOf = (e: HistoryEntry) => (e.originalTotal > 0 ? 1 - e.minifiedTotal / e.originalTotal : 0)

const timeOf = (t: string) => {
  const d = new Date(t)
  return Number.isNaN(d.getTime()) ? t : d.toLocaleString('zh-CN', { hour12: false })
}

async function clearAll() {
  if (!confirming.value) {
    confirming.value = true
    return
  }
  await clearHistory()
  entries.value = []
  confirming.value = false
}
</script>

<template>
  <div v-if="open" class="modal-mask" @click.self="emit('close')">
    <div class="modal history-modal">
      <div class="modal-head">
        <div class="head-icon"><Icon name="clock" :size="17" /></div>
        <div>
          <div class="modal-title">压缩历史</div>
          <div class="modal-sub">
            {{ entries.length }} 次导出 · 累计节省 {{ formatSize(totalSaved) }}
          </div>
        </div>
        <button class="icon-btn" style="margin-left: auto" @click="emit('close')"><Icon name="close" :size="16" /></button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="empty">正在读取历史…</div>
        <div v-else-if="!entries.length" class="empty">
          <Icon name="clock" :size="22" />
          <div>还没有导出记录</div>
          <div class="empty-sub">导出压缩结果后会在这里留下统计信息</div>
        </div>
        <div v-else class="list">
          <div v-for="(e, i) in entries" :key="e.time + i" class="row">
            <div class="row-top">
              <span class="time">{{ timeOf(e.time) }}</span>
              <span class="rate">{{ formatRatio(ratioOf(e)) }}</span>
            </div>
            <div class="row-mid">
              <span><b>{{ e.fileCount }}</b> 个文件</span>
              <span v-if="e.failed" class="fail">{{ e.failed }} 个失败</span>
              <span class="dim">{{ formatSize(e.originalTotal) }} → {{ formatSize(e.minifiedTotal) }}</span>
            </div>
            <div class="dest" :title="e.destDir">
              <Icon name="folder" :size="12" />
              <span>{{ e.destDir }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-foot">
        <button class="btn" :disabled="!entries.length" @click="clearAll">
          <Icon name="trash" :size="14" />{{ confirming ? '确认清空？' : '清空历史' }}
        </button>
        <div class="spacer"></div>
        <button class="btn btn-primary" @click="emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.history-modal {
  width: 620px;
}
.empty {
  padding: 34px 0;
  text-align: center;
  color: var(--text-3);
  font-size: 12.5px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.empty-sub {
  font-size: 11px;
  color: var(--text-3);
  opacity: 0.8;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.row {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--panel-2);
}
.row-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.time {
  font-size: 12px;
  color: var(--text-2);
  font-family: var(--mono);
}
.rate {
  font-size: 11px;
  font-weight: 600;
  color: var(--green);
  background: var(--green-dim);
  border-radius: 4px;
  padding: 2px 5px;
  font-family: var(--mono);
}
.row-mid {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 6px;
  font-size: 11.5px;
  color: var(--text-2);
}
.row-mid b {
  color: var(--text);
  font-weight: 600;
}
.dim {
  color: var(--text-3);
  font-family: var(--mono);
}
.fail {
  color: var(--red);
}
.dest {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-3);
  font-family: var(--mono);
}
.dest span {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.spacer {
  flex: 1;
}
</style>
