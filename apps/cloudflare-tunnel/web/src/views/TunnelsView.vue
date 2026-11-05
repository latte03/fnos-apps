<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import PageHead from '../components/PageHead.vue'
import SegmentedControl from '../components/SegmentedControl.vue'
import TunnelRow from '../components/TunnelRow.vue'
import CreateTunnelModal from '../components/CreateTunnelModal.vue'
import SecretModal from '../components/SecretModal.vue'
import EmptyState from '../components/EmptyState.vue'
import NoticeBar from '../components/NoticeBar.vue'
import SkeletonBlock from '../components/SkeletonBlock.vue'
import { api } from '../lib/client'
import { apiErrorMessage } from '../lib/api'
import { confirmDialog } from '../lib/dialog'
import { shortId } from '../lib/format'
import { toast } from '../lib/toast'
import {
  app,
  connectTunnel,
  createTunnel,
  deleteTunnel,
  disconnectTunnel,
  noteError,
  refreshTunnels,
  tunnelDisplayState,
  type TunnelDisplayState,
} from '../stores/state'
import type { Tunnel } from '../types'

const router = useRouter()

const filter = ref<'all' | 'connected' | 'idle' | 'error'>('all')
const query = ref('')
const loadError = ref('')
const loading = ref(!app.tunnels.length)

const createOpen = ref(false)
const creating = ref(false)
const createError = ref('')

const tokenOpen = ref(false)
const tokenValue = ref('')
const tokenTunnel = ref<Tunnel | null>(null)
const tokenLoadingId = ref('')

const filters = [
  { value: 'all', label: '全部' },
  { value: 'connected', label: '已连接' },
  { value: 'idle', label: '未连接' },
  { value: 'error', label: '异常' },
]

const states = computed(() => {
  const map = new Map<string, TunnelDisplayState>()
  for (const tunnel of app.tunnels) map.set(tunnel.id, tunnelDisplayState(tunnel))
  return map
})

const connectedCount = computed(
  () => app.tunnels.filter((t) => ['connected', 'connecting'].includes(states.value.get(t.id) || '')).length,
)

const filtered = computed(() => {
  const needle = query.value.trim().toLowerCase()
  return app.tunnels.filter((tunnel) => {
    const state = states.value.get(tunnel.id) || 'idle'
    if (filter.value === 'connected' && !['connected', 'connecting'].includes(state)) return false
    if (filter.value === 'idle' && state !== 'idle') return false
    if (filter.value === 'error' && state !== 'error') return false
    if (!needle) return true
    return tunnel.name.toLowerCase().includes(needle) || tunnel.id.toLowerCase().includes(needle)
  })
})

const activeRuntime = computed(() => app.runtime)
const liveTunnel = computed(() =>
  activeRuntime.value && activeRuntime.value.tunnelId && activeRuntime.value.state !== 'disconnected'
    ? activeRuntime.value
    : null,
)

function setFilter(value: string) {
  filter.value = value as 'all' | 'connected' | 'idle' | 'error'
}

function isBusy(tunnel: Tunnel): boolean {
  if (app.connectingId === tunnel.id) return true
  if (app.disconnecting && liveTunnel.value?.tunnelId === tunnel.id) return true
  return false
}

async function load() {
  loading.value = !app.tunnels.length
  loadError.value = (await refreshTunnels()) || ''
  loading.value = false
}

async function onConnect(tunnel: Tunnel) {
  const live = liveTunnel.value
  if (live && live.tunnelId !== tunnel.id && (live.state === 'connected' || live.state === 'connecting')) {
    const current = app.tunnels.find((item) => item.id === live.tunnelId)
    const result = await confirmDialog({
      title: '切换隧道？',
      message: `当前隧道 ${current?.name || live.tunnelName} 将立即断开，${tunnel.name} 会接管连接。`,
      confirmText: '切换连接',
      action: async () => {
        await connectTunnel(tunnel.id, tunnel.name)
      },
    })
    if (result.ok) toast(`正在切换到 ${tunnel.name}`, 'success')
    return
  }
  try {
    await connectTunnel(tunnel.id, tunnel.name)
  } catch (err) {
    toast(noteError(err), 'error')
  }
}

async function onDisconnect(tunnel: Tunnel) {
  const result = await confirmDialog({
    title: '断开隧道？',
    message: '断开后，所有域名转发规则将暂时失效；隧道本身和路由配置会保留。',
    confirmText: '断开连接',
    tone: 'danger',
    action: async () => {
      await disconnectTunnel()
    },
  })
  if (result.ok) toast(`隧道 ${tunnel.name} 已断开`, 'success')
}

async function onDelete(tunnel: Tunnel) {
  const isLive = liveTunnel.value?.tunnelId === tunnel.id
  const result = await confirmDialog({
    title: '删除隧道？',
    message: isLive
      ? `隧道 ${tunnel.name} 正在使用中。删除会先断开连接，并从 Cloudflare 删除该隧道；DNS 记录不会自动删除。`
      : `从 Cloudflare 删除隧道 ${tunnel.name}。该操作不可撤销，DNS 记录不会自动删除。`,
    confirmText: '删除隧道',
    tone: 'danger',
    action: async () => {
      await deleteTunnel(tunnel)
    },
  })
  if (result.ok) toast('隧道已删除', 'success')
}

function onRoutes(tunnel: Tunnel) {
  const state = states.value.get(tunnel.id)
  if (state === 'connected' || state === 'connecting') {
    void router.push({ name: 'routes', query: { tunnel: tunnel.id } })
    return
  }
  toast('请先连接该隧道，再编辑路由', 'error')
}

async function onToken(tunnel: Tunnel) {
  tokenLoadingId.value = tunnel.id
  try {
    const result = await api.tunnelToken(tunnel.id)
    tokenValue.value = result.token
    tokenTunnel.value = tunnel
    tokenOpen.value = true
  } catch (err) {
    toast(noteError(err), 'error')
  } finally {
    tokenLoadingId.value = ''
  }
}

function openCreate() {
  createError.value = ''
  createOpen.value = true
}

async function submitCreate(payload: { name: string; autoConnect: boolean }) {
  creating.value = true
  createError.value = ''
  try {
    const tunnel = await createTunnel(payload.name)
    createOpen.value = false
    toast(`隧道 ${payload.name} 已创建`, 'success')
    if (payload.autoConnect && tunnel) {
      try {
        await connectTunnel(tunnel.id, tunnel.name)
      } catch (err) {
        toast(noteError(err), 'error')
      }
    }
  } catch (err) {
    createError.value = apiErrorMessage(err)
  } finally {
    creating.value = false
  }
}

function onRefresh() {
  void load()
}

onMounted(() => {
  void load()
  window.addEventListener('ct:refresh', onRefresh)
})

onBeforeUnmount(() => {
  window.removeEventListener('ct:refresh', onRefresh)
})
</script>

<template>
  <PageHead
    eyebrow="网络"
    title="隧道"
    sub="创建、连接和切换 Cloudflare Tunnel。同一时间只保留一条活动连接，切换时会先断开当前隧道。"
  >
    <template #actions>
      <button class="btn primary" type="button" @click="openCreate">
        <Icon name="plus" />
        新建隧道
      </button>
    </template>
  </PageHead>

  <NoticeBar v-if="app.needsSetup" tone="warning" icon="alert" title="尚未完成配置">
    先填写 Account ID 与 API Token，然后才能列出 Cloudflare 上的隧道。
    <template #action>
      <RouterLink class="btn small ghost" to="/settings">去设置</RouterLink>
    </template>
  </NoticeBar>
  <NoticeBar v-if="loadError" tone="danger" icon="alert" title="无法读取隧道">
    {{ loadError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="onRefresh">重试</button>
    </template>
  </NoticeBar>

  <div class="toolbar reveal" style="--i: 1">
    <SegmentedControl
      :model-value="filter"
      :options="filters"
      aria-label="筛选隧道"
      @update:model-value="setFilter"
    />
    <div class="search">
      <Icon name="search" />
      <input v-model="query" type="search" placeholder="搜索隧道名称或 ID" autocomplete="off" aria-label="搜索隧道" />
    </div>
    <div class="spacer"></div>
    <span class="micro">{{ app.tunnels.length }} 条隧道 · {{ connectedCount }} 条活动连接</span>
  </div>

  <section class="card tunnel-list reveal" style="--i: 2">
    <div v-if="loading" class="tunnel-skeleton">
      <div v-for="index in 3" :key="index" class="tunnel-row">
        <SkeletonBlock height="34px" width="70%" />
        <SkeletonBlock height="14px" width="60%" />
        <SkeletonBlock height="14px" width="60%" />
        <SkeletonBlock height="22px" width="60%" />
        <SkeletonBlock height="30px" width="140px" />
      </div>
    </div>
    <EmptyState
      v-else-if="!filtered.length"
      icon="waypoints"
      :title="app.tunnels.length ? '没有匹配的隧道' : '还没有隧道'"
      :message="app.tunnels.length ? '换一个筛选条件，或创建一条新隧道。' : '创建第一条隧道，把 NAS 服务接到公网。'"
    >
      <button class="btn primary" type="button" @click="openCreate">
        <Icon name="plus" />
        新建隧道
      </button>
    </EmptyState>
    <template v-else>
      <TunnelRow
        v-for="tunnel in filtered"
        :key="tunnel.id"
        :tunnel="tunnel"
        :state="states.get(tunnel.id) || 'idle'"
        :active="liveTunnel?.tunnelId === tunnel.id"
        :busy="isBusy(tunnel)"
        :last-error="tunnel.status === 'error' ? 'Cloudflare 报告该隧道状态异常' : ''"
        @connect="onConnect"
        @disconnect="onDisconnect"
        @remove="onDelete"
        @routes="onRoutes"
        @token="onToken"
      />
    </template>
  </section>

  <CreateTunnelModal
    v-if="createOpen"
    :submitting="creating"
    :server-error="createError"
    @close="createOpen = false"
    @submit="submitCreate"
  />

  <SecretModal
    v-if="tokenOpen"
    eyebrow="隧道凭证"
    :title="`${tokenTunnel?.name || ''} Token`"
    :secret="tokenValue"
    warning="Token 等同于隧道运行凭据：只显示一次，关闭后从内存清除。"
    :note="`隧道 ${shortId(tokenTunnel?.id || '')} · 通过环境变量 TUNNEL_TOKEN 传给 cloudflared。`"
    @close="tokenOpen = false"
  />

  <span v-if="tokenLoadingId" class="sr-only" aria-live="polite">正在读取隧道 Token</span>
</template>
