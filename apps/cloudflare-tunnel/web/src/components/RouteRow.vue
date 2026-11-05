<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'
import StatusPill from './StatusPill.vue'
import type { DnsUiState } from '../lib/dns'
import { dnsMeta } from '../lib/dns'
import type { Route } from '../types'

const props = defineProps<{
  route: Route
  index: number
  dnsState: DnsUiState
  dragging?: boolean
  dragOver?: boolean
  busy?: boolean
}>()

const emit = defineEmits<{
  edit: [route: Route]
  remove: [route: Route]
  dns: [route: Route]
  move: [route: Route, delta: number]
}>()

const meta = computed(() => dnsMeta(props.dnsState))
const pathLabel = computed(() => {
  const path = props.route.path
  return !path || path === '/' ? '根路径' : `路径 ${path}`
})
</script>

<template>
  <div
    class="route-row"
    :class="{ 'is-dragging': dragging, 'is-drag-over': dragOver }"
    draggable="true"
  >
    <span class="route-index">{{ String(index + 1).padStart(2, '0') }}</span>
    <button
      class="drag-handle"
      type="button"
      aria-label="拖动排序，或用上下方向键调整顺序"
      title="拖动排序"
      @keydown.up.prevent="emit('move', route, -1)"
      @keydown.down.prevent="emit('move', route, 1)"
    >
      <Icon name="grip" />
    </button>
    <div class="route-host">
      <strong>{{ route.hostname }}</strong>
      <small>{{ pathLabel }}</small>
    </div>
    <span class="route-arrow"><Icon name="arrow-right" /></span>
    <div class="route-service">{{ route.service }}</div>
    <StatusPill :label="meta.label" :kind="meta.pill" />
    <div class="route-actions">
      <button
        class="icon-btn"
        type="button"
        :aria-label="`为 ${route.hostname} 创建 CNAME`"
        title="创建/检查 CNAME"
        :disabled="busy"
        @click="emit('dns', route)"
      >
        <span v-if="busy" class="spinner" aria-hidden="true"></span>
        <Icon v-else name="cloud" />
      </button>
      <button class="icon-btn" type="button" :aria-label="`编辑 ${route.hostname}`" title="编辑" @click="emit('edit', route)">
        <Icon name="pencil" />
      </button>
      <button class="icon-btn" type="button" :aria-label="`删除 ${route.hostname}`" title="删除" @click="emit('remove', route)">
        <Icon name="trash" />
      </button>
    </div>
  </div>
</template>
