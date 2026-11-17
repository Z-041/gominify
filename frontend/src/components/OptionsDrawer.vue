<script setup lang="ts">
import Icon from './Icon.vue'
import type { MinifyOptions } from '../types'

defineProps<{ open: boolean; options: MinifyOptions }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'apply', opts: MinifyOptions): void
}>()

const toggle = (o: MinifyOptions, key: 'JSMangle' | 'KeepCopyright') => {
  o[key] = !o[key]
}
</script>

<template>
  <div class="overlay" :class="{ show: open }" @click="emit('close')"></div>
  <aside class="drawer" :class="{ show: open }">
    <div class="drawer-head">
      压缩选项
      <button class="icon-btn" style="margin-left: auto" @click="emit('close')"><Icon name="close" :size="16" /></button>
    </div>

    <div class="drawer-body">
      <div class="group-title">JavaScript</div>
      <div class="opt" @click="toggle(options, 'JSMangle')">
        <div class="opt-text">
          <div class="opt-name">变量名混淆</div>
          <div class="opt-desc">重命名局部变量与参数，进一步减小体积</div>
        </div>
        <div class="switch" :class="{ on: options.JSMangle }"></div>
      </div>
      <div class="opt" @click="toggle(options, 'KeepCopyright')">
        <div class="opt-text">
          <div class="opt-name">保留版权注释</div>
          <div class="opt-desc">保留以 /*! 开头的许可声明注释</div>
        </div>
        <div class="switch" :class="{ on: options.KeepCopyright }"></div>
      </div>

      <div class="group-title">CSS</div>
      <div class="opt" style="display: block">
        <div class="opt-text">
          <div class="opt-name">数值精度</div>
          <div class="opt-desc">小数保留位数，越低体积越小</div>
          <div class="seg-level">
            <span
              v-for="n in 4"
              :key="n"
              :class="{ on: options.CSSPrecision === n - 1 }"
              @click.stop="options.CSSPrecision = n - 1"
              >{{ n - 1 }} 位</span
            >
          </div>
        </div>
      </div>

      <div class="group-title">通用</div>
      <div class="opt static">
        <div class="opt-text">
          <div class="opt-name">单文件上限</div>
          <div class="opt-desc">5 MB，超出会跳过以避免界面卡顿</div>
        </div>
      </div>
    </div>

    <div class="drawer-foot">
      <button class="btn" @click="emit('close')">取消</button>
      <button class="btn btn-primary" @click="emit('apply', options)">应用并重压</button>
    </div>
  </aside>
</template>

<style scoped>
.overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.22s;
  z-index: 20;
}
.overlay.show {
  opacity: 1;
  pointer-events: auto;
}
.drawer {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 340px;
  background: var(--panel);
  border-left: 1px solid var(--border);
  transform: translateX(100%);
  transition: transform 0.26s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 21;
  display: flex;
  flex-direction: column;
  box-shadow: -24px 0 60px rgba(0, 0, 0, 0.4);
}
.drawer.show {
  transform: translateX(0);
}
.drawer-head {
  height: 48px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  font-weight: 600;
}
.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}
.group-title {
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  margin: 18px 0 8px;
}
.group-title:first-child {
  margin-top: 2px;
}
.opt {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 11px 12px;
  border-radius: var(--r-sm);
  cursor: pointer;
}
.opt:hover {
  background: var(--panel-2);
}
.opt.static {
  cursor: default;
}
.opt-text {
  flex: 1;
}
.opt-name {
  font-size: 12.5px;
}
.opt-desc {
  font-size: 11px;
  color: var(--text-3);
  margin-top: 3px;
  line-height: 1.6;
}
.switch {
  width: 34px;
  height: 19px;
  border-radius: 999px;
  background: #2a323c;
  position: relative;
  flex-shrink: 0;
  margin-top: 1px;
  transition: background 0.18s;
}
.switch::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 15px;
  height: 15px;
  border-radius: 50%;
  background: #7a8492;
  transition: all 0.18s;
}
.switch.on {
  background: rgba(78, 143, 247, 0.7);
}
.switch.on::after {
  left: 17px;
  background: #fff;
}
.seg-level {
  display: flex;
  background: var(--panel-2);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
  margin-top: 8px;
  width: 220px;
}
.seg-level span {
  flex: 1;
  text-align: center;
  font-size: 11px;
  color: var(--text-3);
  padding: 4px 0;
  border-radius: 4px;
  cursor: pointer;
}
.seg-level span.on {
  background: var(--panel-3);
  color: var(--text);
}
.drawer-foot {
  flex-shrink: 0;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  display: flex;
  gap: 8px;
}
.drawer-foot .btn {
  flex: 1;
  justify-content: center;
  padding: 9px;
}
</style>