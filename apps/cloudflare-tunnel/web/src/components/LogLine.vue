<script setup lang="ts">
import { computed } from 'vue'
import { formatClock } from '../lib/format'
import type { LogLine } from '../types'

const props = defineProps<{ line: LogLine; expanded: boolean }>()
const emit = defineEmits<{ toggle: [] }>()

const levelTag: Record<string, string> = {
  debug: 'DBG',
  info: 'INF',
  warn: 'WRN',
  error: 'ERR',
}

const levelClass = computed(() => ({ debug: 'dbg', info: 'inf', warn: 'wrn', error: 'err' })[props.line.level] || 'inf')

const inlineMeta = computed(() => {
  const entries = Object.entries(props.line.fields || {}).slice(0, 4)
  return entries
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .map(([key, value]) => `${key}=${typeof value === 'object' ? JSON.stringify(value) : String(value)}`)
    .join(' ')
})

const detail = computed(() =>
  JSON.stringify(
    {
      seq: props.line.seq,
      ts: props.line.ts,
      level: props.line.level,
      message: props.line.message,
      ...props.line.fields,
    },
    null,
    2,
  ),
)
</script>

<template>
  <div class="log-entry">
    <button class="log-line" type="button" :aria-expanded="expanded" @click="emit('toggle')">
      <span class="log-time">{{ formatClock(line.ts) }}</span>
      <span class="log-level" :class="levelClass">{{ levelTag[line.level] || 'INF' }}</span>
      <span class="log-msg">
        {{ line.message }}<span v-if="inlineMeta" class="meta">{{ inlineMeta }}</span>
      </span>
    </button>
    <pre v-show="expanded" class="log-detail">{{ detail }}</pre>
  </div>
</template>
