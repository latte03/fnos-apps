<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'
import StatusPill from './StatusPill.vue'
import { formatDate, shortId } from '../lib/format'
import { statusMetaFor, type TunnelDisplayState } from '../stores/state'
import type { Tunnel } from '../types'

const props = defineProps<{
  tunnel: Tunnel
  state: TunnelDisplayState
  active: boolean
  busy: boolean
  lastError?: string
}>()

const emit = defineEmits<{
  connect: [tunnel: Tunnel]
  disconnect: [tunnel: Tunnel]
  remove: [tunnel: Tunnel]
  routes: [tunnel: Tunnel]
  token: [tunnel: Tunnel]
}>()

const meta = computed(() => statusMetaFor(props.state))
const createdLabel = computed(() => formatDate(props.tunnel.createdAt))
const micro = computed(() => {
  if (props.state === 'connected') return '已连接 · 当前'
  if (props.state === 'error') return props.lastError || '连接中断'
  return meta.value.micro
})
</script>

<template>
  <div class="tunnel-row" :class="{ 'is-active': active }" :data-id="tunnel.id">
    <div class="tunnel-ident">
      <span class="status-dot" :class="meta.dot" aria-hidden="true"></span>
      <div class="t-copy">
        <strong>{{ tunnel.name }}</strong>
        <span class="mono">{{ shortId(tunnel.id) }}</span>
      </div>
    </div>
    <div class="tunnel-meta"><strong>{{ tunnel.connections }}</strong>条连接</div>
    <div class="tunnel-meta"><strong>{{ createdLabel }}</strong>创建</div>
    <div class="tunnel-status">
      <StatusPill :label="meta.label" :kind="meta.pill" :dot="meta.pill === 'healthy' || meta.pill === 'danger'" />
      <span class="micro">{{ micro }}</span>
    </div>
    <div class="row-actions">
      <button
        v-if="state === 'connected'"
        class="btn small ghost"
        type="button"
        :disabled="busy"
        @click="emit('disconnect', tunnel)"
      >
        <span v-if="busy" class="spinner" aria-hidden="true"></span>
        断开
      </button>
      <button v-else-if="state === 'connecting'" class="btn small ghost" type="button" disabled>
        <span class="spinner" aria-hidden="true"></span>
        连接中
      </button>
      <button
        v-else
        class="btn small ghost"
        type="button"
        :disabled="busy"
        @click="emit('connect', tunnel)"
      >
        <span v-if="busy" class="spinner" aria-hidden="true"></span>
        连接
      </button>
      <button class="btn small ghost" type="button" @click="emit('routes', tunnel)">路由</button>
      <button
        class="icon-btn"
        type="button"
        :aria-label="`查看 ${tunnel.name} 的 Token`"
        title="查看 Token"
        @click="emit('token', tunnel)"
      >
        <Icon name="key" />
      </button>
      <button
        class="icon-btn"
        type="button"
        :aria-label="`删除隧道 ${tunnel.name}`"
        title="删除隧道"
        @click="emit('remove', tunnel)"
      >
        <Icon name="trash" />
      </button>
    </div>
  </div>
</template>
