<script setup lang="ts">
const props = withDefaults(defineProps<{ name: string; size?: number }>(), { size: 16 })

const paths: Record<string, string> = {
  bolt: 'M13 2 4 14h7l-1 8 9-12h-7l1-8z',
  tune: 'M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12M20 18h0',
  add: 'M12 5v14M5 12h14',
  upload: 'M12 16V4m0 0L7 9m5-5 5 5M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2',
  download: 'M12 4v12m0 0 5-5m-5 5-5-5M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2',
  copy: 'M9 9h10v10a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2V9zM5 15V5a2 2 0 0 1 2-2h10',
  close: 'M6 6l12 12M18 6 6 18',
  refresh: 'M20 11a8 8 0 0 0-14-5L4 8m0-5v5h5M4 13a8 8 0 0 0 14 5l2-2m0 5v-5h-5',
  folder: 'M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z',
  error: 'M12 8v5M12 16.5v.5M10.3 3.9 2.7 17a2 2 0 0 0 1.7 3h15.2a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z',
  check: 'M4 12l5 5L20 6',
  update: 'M12 3v12m0 0 4-4m-4 4-4-4M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2',
  trash: 'M4 7h16M9 7V5h6v2M6 7l1 13h10l1-13',
  chevron: 'M9 6l6 6-6 6',
  sun: 'M12 4v2m0 12v2M4 12h2m12 0h2M6.3 6.3l1.4 1.4m8.6 8.6 1.4 1.4m0-11.4-1.4 1.4M7.7 16.3l-1.4 1.4M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z',
}
</script>

<template>
  <svg
    :width="props.size"
    :height="props.size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.7"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="paths[props.name] || paths.bolt" />
  </svg>
</template>