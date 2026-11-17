<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { applyUpdate, checkUpdate, onUpdateProgress, setMockProgressHandler } from '../composables/useBackend'
import type { UpdateInfo, UpdateProgress } from '../types'
import { formatSize } from '../highlight'

const props = defineProps<{ open: boolean; version: string }>()
const emit = defineEmits<{ (e: 'close'): void }>()

type State = 'idle' | 'checking' | 'available' | 'latest' | 'error' | 'downloading' | 'done'
const state = ref<State>('idle')
const info = ref<UpdateInfo | null>(null)
const progress = ref(0)
const message = ref('')

const updateProgress = (p: UpdateProgress) => {
  progress.value = p.percent
}

onMounted(() => {
  onUpdateProgress(updateProgress)
  setMockProgressHandler(updateProgress)
})

watch(
  () => props.open,
  (v) => {
    if (v) {
      state.value = 'idle'
      info.value = null
      progress.value = 0
      message.value = ''
      void check()
    }
  },
)

async function check() {
  state.value = 'checking'
  message.value = ''
  try {
    info.value = await checkUpdate()
    state.value = info.value.available ? 'available' : 'latest'
  } catch (e: any) {
    state.value = 'error'
    message.value = String(e?.message ?? e ?? '检查更新失败')
  }
}

async function startUpdate() {
  state.value = 'downloading'
  progress.value = 0
  try {
    await applyUpdate()
    state.value = 'done'
  } catch (e: any) {
    state.value = 'error'
    message.value = String(e?.message ?? e ?? '更新失败')
  }
}
</script>

<template>
  <div v-if="open" class="modal-mask" @click.self="emit('close')">
    <div class="modal">
      <div class="modal-head">
        <div class="head-icon"><Icon name="update" :size="17" /></div>
        <div>
          <div class="modal-title">软件更新</div>
          <div class="modal-sub">离线安装包，无需重新下载整个应用</div>
        </div>
        <button class="icon-btn" style="margin-left: auto" @click="emit('close')"><Icon name="close" :size="16" /></button>
      </div>

      <div class="modal-body">
        <div class="ver-row">
          <span>当前版本</span>
          <b>v{{ version }}</b>
        </div>

        <div v-if="state === 'checking'" class="status">正在检查更新…</div>

        <template v-else-if="state === 'available' && info">
          <div class="ver-row">
            <span>最新版本</span>
            <b class="new">v{{ info.latest }}</b>
          </div>
          <div class="notes">{{ info.notes || '包含稳定性改进。' }}</div>
          <div class="meta">更新包 {{ formatSize(info.size) }} · 下载后校验 SHA-256 与数字签名</div>
        </template>

        <div v-else-if="state === 'latest'" class="status ok"><Icon name="check" :size="15" /> 已是最新版本</div>

        <template v-else-if="state === 'downloading'">
          <div class="status">正在下载并校验更新包…</div>
          <div class="progress"><div class="bar" :style="{ width: progress + '%' }"></div></div>
          <div class="meta">{{ progress }}%</div>
        </template>

        <div v-else-if="state === 'done'" class="status ok"><Icon name="check" :size="15" /> 更新完成，应用即将重启</div>

        <div v-else-if="state === 'error'" class="status err">
          <Icon name="error" :size="15" /> {{ message }}
          <button class="btn retry" @click="check">重试</button>
        </div>
      </div>

      <div class="modal-foot">
        <button class="btn" @click="emit('close')">{{ state === 'downloading' ? '后台继续' : '关闭' }}</button>
        <button class="btn btn-primary" :disabled="state !== 'available'" @click="startUpdate">
          <Icon name="download" :size="15" />{{ state === 'downloading' ? '更新中…' : '下载并安装' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ver-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12.5px;
  color: var(--text-2);
  padding: 6px 0;
}
.ver-row b {
  color: var(--text);
  font-family: var(--mono);
  font-weight: 600;
}
.ver-row b.new {
  color: var(--green);
}
.notes {
  margin-top: 10px;
  padding: 10px 12px;
  background: var(--panel-2);
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.7;
}
.meta {
  margin-top: 8px;
  font-size: 11px;
  color: var(--text-3);
}
.status {
  margin-top: 14px;
  font-size: 12.5px;
  color: var(--text-2);
  display: flex;
  align-items: center;
  gap: 6px;
}
.status.ok {
  color: var(--green);
}
.status.err {
  color: var(--red);
  flex-wrap: wrap;
}
.retry {
  margin-left: auto;
  padding: 4px 10px;
}
.progress {
  margin-top: 12px;
  height: 6px;
  background: var(--panel-3);
  border-radius: 99px;
  overflow: hidden;
}
.bar {
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--green));
  transition: width 0.2s;
}
</style>