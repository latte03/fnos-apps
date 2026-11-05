<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import PageHead from '../components/PageHead.vue'
import RouteRow from '../components/RouteRow.vue'
import RouteDrawer from '../components/RouteDrawer.vue'
import EmptyState from '../components/EmptyState.vue'
import NoticeBar from '../components/NoticeBar.vue'
import SkeletonBlock from '../components/SkeletonBlock.vue'
import StatusPill from '../components/StatusPill.vue'
import { api } from '../lib/client'
import { apiErrorMessage } from '../lib/api'
import { cloneRoutes, getCachedRoutes, routeCacheKey, setCachedRoutes, type CachedRoutes } from '../lib/route-cache'
import { confirmDialog } from '../lib/dialog'
import type { DnsUiState } from '../lib/dns'
import { shortId } from '../lib/format'
import { copyText } from '../lib/clipboard'
import { toast } from '../lib/toast'
import { app, noteError, setRouteCount, statusMetaFor, tunnelDisplayState } from '../stores/state'
import type { DnsResult, Route } from '../types'

interface YamlSegment {
  t: 'cmt' | 'key' | 'val' | 'dash' | 'plain'
  v: string
}

const route = useRoute()
const router = useRouter()

const selectedId = ref('')
const saved = ref<Route[]>([])
const draft = ref<Route[]>([])
const catchAll = ref('http_status:404')
const savedCatchAll = ref('http_status:404')
const rawIngressCount = ref(0)
const loading = ref(false)
const loaded = ref(false)
const loadError = ref('')
const saving = ref(false)
const saveError = ref('')
const pickerOpen = ref(false)

const dns = reactive<Record<string, DnsUiState>>({})
const dnsDetail = reactive<Record<string, DnsResult>>({})
const autoDnsFlags = reactive<Record<string, boolean>>({})
const dnsBusy = ref('')
const dnsError = ref('')

const drawer = ref<{ mode: 'add' | 'edit'; initial: Route | null; originalHost: string } | null>(null)

const draggingId = ref('')
const overId = ref('')

const selectedTunnel = computed(() => app.tunnels.find((tunnel) => tunnel.id === selectedId.value) || null)
const liveTunnelId = computed(() => (app.runtime?.state !== 'disconnected' ? app.runtime?.tunnelId || '' : ''))
const tunnelState = computed(() => (selectedTunnel.value ? tunnelDisplayState(selectedTunnel.value) : 'idle'))
const stateMeta = computed(() => statusMetaFor(tunnelState.value === 'connected' ? 'connected' : tunnelState.value))

const dirty = computed(
  () =>
    JSON.stringify(draft.value.map(wireRoute)) !== JSON.stringify(saved.value.map(wireRoute)) ||
    catchAll.value !== savedCatchAll.value,
)

const syncState = computed<'saved' | 'saving' | 'dirty' | 'error'>(() => {
  if (saveError.value) return 'error'
  if (saving.value) return 'saving'
  if (dirty.value) return 'dirty'
  return 'saved'
})

const syncLabel = computed(() => {
  switch (syncState.value) {
    case 'saving':
      return '保存中'
    case 'dirty':
      return '待保存'
    case 'error':
      return '保存失败'
    default:
      return '已保存'
  }
})

let saveTimer: ReturnType<typeof setTimeout> | null = null
let saveQueued = false
let revision = 0
let pendingToast = ''

const tunnelNeedsRun = computed(() => Boolean(selectedTunnel.value) && liveTunnelId.value !== selectedId.value)

const hostnames = computed(() => draft.value.map((item) => item.hostname))

const lastZone = computed(() => {
  const zones = new Set(Object.values(dnsDetail).map((detail) => detail.zoneName).filter(Boolean))
  return zones.size ? Array.from(zones).join(', ') : ''
})

const dnsCounts = computed(() => {
  const entries = Object.values(dns)
  return {
    created: entries.filter((value) => value === 'created').length,
    updated: entries.filter((value) => value === 'updated').length,
    unchanged: entries.filter((value) => value === 'unchanged').length,
    pending: entries.filter((value) => value === 'pending').length,
    failed: entries.filter((value) => value === 'failed').length,
    unknown: draft.value.filter((item) => !dns[item.hostname]).length,
  }
})

const yaml = computed<YamlSegment[][]>(() => {
  const lines: YamlSegment[][] = []
  lines.push([{ t: 'cmt', v: '# 由 Tunnel 路由表生成' }])
  lines.push([{ t: 'key', v: 'tunnel' }, { t: 'plain', v: ': ' }, { t: 'val', v: selectedId.value || '—' }])
  lines.push([{ t: 'key', v: 'ingress' }, { t: 'plain', v: ':' }])
  draft.value.forEach((item) => {
    lines.push([
      { t: 'dash', v: '  - ' },
      { t: 'key', v: 'hostname' },
      { t: 'plain', v: ': ' },
      { t: 'val', v: item.hostname },
    ])
    if (item.path && item.path !== '/') {
      lines.push([{ t: 'plain', v: '    ' }, { t: 'key', v: 'path' }, { t: 'plain', v: ': ' }, { t: 'val', v: item.path }])
    }
    lines.push([
      { t: 'plain', v: '    ' },
      { t: 'key', v: 'service' },
      { t: 'plain', v: ': ' },
      { t: 'val', v: item.service },
    ])
  })
  lines.push([
    { t: 'dash', v: '  - ' },
    { t: 'key', v: 'service' },
    { t: 'plain', v: ': ' },
    { t: 'val', v: catchAll.value || 'http_status:404' },
  ])
  return lines
})

const yamlText = computed(() => yaml.value.map((line) => line.map((segment) => segment.v).join('')).join('\n'))

function wireRoute(item: Route): Route {
  return { hostname: item.hostname, path: item.path || '/', service: item.service }
}

function resetDnsState() {
  for (const key of Object.keys(dns)) delete dns[key]
  for (const key of Object.keys(dnsDetail)) delete dnsDetail[key]
  for (const key of Object.keys(autoDnsFlags)) delete autoDnsFlags[key]
  dnsError.value = ''
}

function routesCacheKey(tunnelId: string) {
  return routeCacheKey(app.settings?.accountId, tunnelId)
}

function applyRoutesData(tunnelId: string, data: CachedRoutes) {
  saved.value = cloneRoutes(data.routes)
  draft.value = cloneRoutes(data.routes)
  catchAll.value = data.catchAllService || 'http_status:404'
  savedCatchAll.value = catchAll.value
  rawIngressCount.value = data.rawIngressCount
  loaded.value = true
  setCachedRoutes(routesCacheKey(tunnelId), {
    routes: data.routes,
    catchAllService: catchAll.value,
    rawIngressCount: data.rawIngressCount,
  })
  if (tunnelId === liveTunnelId.value) setRouteCount(data.routes.length)
}

async function loadRoutes(tunnelId = selectedId.value, force = false) {
  if (!tunnelId) return
  if (!force) {
    const cached = getCachedRoutes(routesCacheKey(tunnelId))
    if (cached) {
      selectedId.value = tunnelId
      loading.value = false
      loadError.value = ''
      applyRoutesData(tunnelId, cached)
      return
    }
  }
  loading.value = true
  loadError.value = ''
  saveError.value = ''
  try {
    const result = await api.routes(tunnelId)
    if (selectedId.value !== tunnelId) return
    selectedId.value = tunnelId
    applyRoutesData(tunnelId, {
      routes: result.routes,
      catchAllService: result.catchAllService,
      rawIngressCount: result.rawIngressCount,
    })
  } catch (err) {
    loadError.value = noteError(err)
  } finally {
    loading.value = false
  }
}

async function selectTunnel(id: string) {
  pickerOpen.value = false
  if (id === selectedId.value) return
  if (saving.value) {
    toast('正在保存，请稍候再切换隧道', 'error')
    return
  }
  if (dirty.value) {
    await flushSave()
    if (dirty.value) {
      toast('当前隧道仍有未保存的修改，请先重试保存', 'error')
      return
    }
  }
  resetDnsState()
  selectedId.value = id
  loaded.value = false
  draft.value = []
  saved.value = []
  await loadRoutes(id)
}

function initialTunnelId(): string {
  const fromQuery = typeof route.query.tunnel === 'string' ? route.query.tunnel : ''
  if (fromQuery && app.tunnels.some((tunnel) => tunnel.id === fromQuery)) return fromQuery
  const live = liveTunnelId.value
  if (live) return live
  return app.tunnels[0]?.id || ''
}

function openAdd() {
  if (!selectedId.value) {
    toast('请先选择或创建一条隧道', 'error')
    void router.push('/tunnels')
    return
  }
  drawer.value = { mode: 'add', initial: null, originalHost: '' }
}

function openEdit(item: Route) {
  drawer.value = { mode: 'edit', initial: { ...item }, originalHost: item.hostname }
}

function applyDrawer(payload: { hostname: string; path: string; service: string; autoDns: boolean }) {
  const current = drawer.value
  if (!current) return
  const next: Route = { hostname: payload.hostname, path: payload.path, service: payload.service }
  if (current.mode === 'add') {
    draft.value = [...draft.value, next]
    autoDnsFlags[payload.hostname] = payload.autoDns
  } else {
    draft.value = draft.value.map((item) => (item.hostname === current.originalHost ? next : item))
    if (current.originalHost !== payload.hostname) {
      const flag = autoDnsFlags[current.originalHost]
      delete autoDnsFlags[current.originalHost]
      delete dns[current.originalHost]
      if (flag !== undefined) autoDnsFlags[payload.hostname] = flag
    }
    if (payload.autoDns) autoDnsFlags[payload.hostname] = true
  }
  drawer.value = null
  revision += 1
  pendingToast = current.mode === 'add' ? '路由已添加' : '路由已更新'
  void flushSave()
}

async function removeRoute(item: Route) {
  const result = await confirmDialog({
    title: '删除路由？',
    message: `删除 ${item.hostname} 的转发规则。DNS 记录会保留在 Cloudflare，需要时可在控制台手动清理。`,
    confirmText: '删除路由',
    tone: 'danger',
  })
  if (!result.ok) return
  draft.value = draft.value.filter((entry) => entry.hostname !== item.hostname)
  delete autoDnsFlags[item.hostname]
  delete dns[item.hostname]
  revision += 1
  pendingToast = '路由已删除'
  void flushSave()
}

function moveRoute(item: Route, delta: number) {
  const list = [...draft.value]
  const from = list.findIndex((entry) => entry.hostname === item.hostname)
  const to = from + delta
  if (from < 0 || to < 0 || to >= list.length) return
  const [moved] = list.splice(from, 1)
  list.splice(to, 0, moved)
  draft.value = list
  revision += 1
  scheduleSave()
}

function onDragStart(event: DragEvent, item: Route) {
  draggingId.value = item.hostname
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', item.hostname)
  }
}

function onDragOver(event: DragEvent, item: Route) {
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
  if (item.hostname !== draggingId.value) overId.value = item.hostname
}

function onDrop(event: DragEvent, item: Route) {
  event.preventDefault()
  const source = draggingId.value
  overId.value = ''
  draggingId.value = ''
  if (!source || source === item.hostname) return
  const list = [...draft.value]
  const from = list.findIndex((entry) => entry.hostname === source)
  const to = list.findIndex((entry) => entry.hostname === item.hostname)
  if (from < 0 || to < 0) return
  const [moved] = list.splice(from, 1)
  list.splice(to, 0, moved)
  draft.value = list
  revision += 1
  scheduleSave()
}

function onDragEnd() {
  draggingId.value = ''
  overId.value = ''
}

async function createDnsFor(item: Route) {
  if (!selectedId.value || dnsBusy.value) return
  dnsBusy.value = item.hostname
  dns[item.hostname] = 'pending'
  dnsError.value = ''
  try {
    const result = await api.createDns(selectedId.value, item.hostname)
    dns[item.hostname] = result.action
    dnsDetail[item.hostname] = result
    const labels: Record<string, string> = {
      created: `CNAME ${item.hostname} 已创建 · ${result.zoneName}`,
      updated: `CNAME ${item.hostname} 已更新`,
      unchanged: `CNAME ${item.hostname} 已存在，无需变更`,
    }
    toast(labels[result.action] || 'DNS 记录已处理', 'success')
  } catch (err) {
    dns[item.hostname] = 'failed'
    dnsError.value = apiErrorMessage(err)
    toast(noteError(err), 'error')
  } finally {
    dnsBusy.value = ''
  }
}

async function runPendingDns(tunnelId: string, hosts: string[]) {
  for (const host of hosts) {
    const item = draft.value.find((entry) => entry.hostname === host)
    if (!item) continue
    dnsBusy.value = host
    dns[host] = 'pending'
    try {
      const result = await api.createDns(tunnelId, host)
      dns[host] = result.action
      dnsDetail[host] = result
    } catch (err) {
      dns[host] = 'failed'
      dnsError.value = apiErrorMessage(err)
    } finally {
      dnsBusy.value = ''
    }
  }
  const failed = Object.values(dns).filter((value) => value === 'failed').length
  const created = Object.values(dns).filter((value) => value === 'created' || value === 'updated').length
  if (failed) toast(`${failed} 条 CNAME 创建失败，可在行内重试`, 'error')
  else if (created) toast(`${created} 条 CNAME 已就绪`, 'success')
}

function scheduleSave() {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    saveTimer = null
    void flushSave()
  }, 450)
}

async function flushSave() {
  if (!selectedId.value) return
  if (saving.value) {
    saveQueued = true
    return
  }
  if (!dirty.value) {
    saveError.value = ''
    return
  }
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }
  const tunnelId = selectedId.value
  const requestRevision = revision
  const message = pendingToast
  pendingToast = ''
  const payloadRoutes = draft.value.map(wireRoute)
  const payloadCatchAll = catchAll.value.trim() || 'http_status:404'
  const pendingHosts = payloadRoutes
    .filter((item) => autoDnsFlags[item.hostname])
    .map((item) => item.hostname)
    .filter((host) => !['created', 'updated', 'unchanged'].includes(dns[host] || ''))

  saving.value = true
  saveError.value = ''
  try {
    const result = await api.saveRoutes(tunnelId, {
      routes: payloadRoutes,
      catchAllService: payloadCatchAll,
    })
    if (selectedId.value !== tunnelId) return
    saved.value = cloneRoutes(result.routes)
    savedCatchAll.value = result.catchAllService || payloadCatchAll
    rawIngressCount.value = result.rawIngressCount
    if (revision === requestRevision) {
      draft.value = cloneRoutes(result.routes)
      catchAll.value = result.catchAllService || payloadCatchAll
    }
    setCachedRoutes(routesCacheKey(tunnelId), {
      routes: result.routes,
      catchAllService: result.catchAllService || payloadCatchAll,
      rawIngressCount: result.rawIngressCount,
    })
    if (tunnelId === liveTunnelId.value) setRouteCount(result.routes.length)
    for (const host of pendingHosts) delete autoDnsFlags[host]
    if (message) toast(message, 'success')
    if (pendingHosts.length) void runPendingDns(tunnelId, pendingHosts)
  } catch (err) {
    saveError.value = apiErrorMessage(err)
    toast(saveError.value, 'error')
  } finally {
    saving.value = false
    if (saveQueued) {
      saveQueued = false
      void flushSave()
    } else if (dirty.value && !saveError.value) {
      void flushSave()
    }
  }
}

function retrySave() {
  saveError.value = ''
  void flushSave()
}

function onCatchAllInput() {
  revision += 1
  scheduleSave()
}

function copyYaml() {
  void copyText(yamlText.value, 'config.yml 已复制')
}

async function onRefresh() {
  if (!selectedId.value || saving.value) {
    if (saving.value) toast('正在保存，请稍候再刷新', 'error')
    return
  }
  if (dirty.value) {
    await flushSave()
    if (dirty.value) return
  }
  resetDnsState()
  await loadRoutes(selectedId.value, true)
  if (!loadError.value) toast('路由数据已刷新', 'success')
}

function onDocumentClick(event: MouseEvent) {
  if (!pickerOpen.value) return
  const target = event.target as HTMLElement | null
  if (target && target.closest('.picker')) return
  pickerOpen.value = false
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && pickerOpen.value) pickerOpen.value = false
}

function onBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

let stopTunnelWatch: (() => void) | null = null

stopTunnelWatch = watch(
  () => app.tunnels.length,
  (count) => {
    if (!count) return
    if (!selectedId.value) {
      selectedId.value = initialTunnelId()
      if (selectedId.value) void loadRoutes()
    }
  },
)

onMounted(async () => {
  selectedId.value = initialTunnelId()
  if (selectedId.value) await loadRoutes()
  document.addEventListener('click', onDocumentClick)
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('beforeunload', onBeforeUnload)
  window.addEventListener('ct:refresh', onRefresh)

  if (typeof route.query.host === 'string') {
    const target = draft.value.find((item) => item.hostname === route.query.host)
    if (target) openEdit(target)
    void router.replace({ name: 'routes' })
  } else if (route.query.add === '1') {
    openAdd()
    void router.replace({ name: 'routes' })
  }
})

onBeforeUnmount(() => {
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
    if (dirty.value && !saving.value) void flushSave()
  }
  document.removeEventListener('click', onDocumentClick)
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('beforeunload', onBeforeUnload)
  window.removeEventListener('ct:refresh', onRefresh)
  stopTunnelWatch?.()
})
</script>

<template>
  <PageHead
    title="路由 · Ingress 规则"
    sub="一个域名对应一个内网服务。规则按顺序匹配，首条命中后不再继续；最后一条为兜底。"
  >
    <template #actions>
      <span class="save-state" :class="`is-${syncState}`" role="status">
        <span v-if="syncState === 'saving'" class="spinner" aria-hidden="true"></span>
        <Icon v-else :name="syncState === 'error' ? 'alert' : 'check'" />
        {{ syncLabel }}
      </span>
      <button class="btn ghost" type="button" :disabled="loading || saving" @click="onRefresh">
        <span v-if="loading" class="spinner" aria-hidden="true"></span>
        <Icon v-else name="refresh" />
        {{ loading ? '刷新中' : '刷新' }}
      </button>
      <button class="btn primary" type="button" @click="openAdd">
        <Icon name="plus" />
        添加路由
      </button>
    </template>
  </PageHead>

  <NoticeBar v-if="app.needsSetup" tone="warning" icon="alert" title="尚未完成配置">
    先填写 Account ID 与 API Token，才能读取与写入 ingress。
    <template #action>
      <RouterLink class="btn small ghost" to="/settings">去设置</RouterLink>
    </template>
  </NoticeBar>
  <NoticeBar v-if="loadError" tone="danger" icon="alert" title="无法读取路由">
    {{ loadError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="onRefresh">重试</button>
    </template>
  </NoticeBar>
  <NoticeBar v-if="saveError" tone="danger" icon="alert" title="自动保存失败">
    {{ saveError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="retrySave">重试</button>
    </template>
  </NoticeBar>

  <section v-if="!app.tunnels.length && !app.booting" class="card reveal" style="--i: 1">
    <EmptyState icon="waypoints" title="还没有隧道" message="路由属于某一条隧道。先创建并连接一条隧道。" big>
      <RouterLink class="btn primary" to="/tunnels">
        <Icon name="waypoints" />
        去隧道
      </RouterLink>
    </EmptyState>
  </section>

  <template v-else>
    <div class="toolbar reveal" style="--i: 1">
      <div class="picker">
        <button
          class="account-chip picker-trigger"
          type="button"
          aria-haspopup="listbox"
          :aria-expanded="pickerOpen"
          aria-label="选择隧道"
          :title="selectedTunnel?.name || '选择隧道'"
          @click="pickerOpen = !pickerOpen"
        >
          <span class="status-dot" :class="stateMeta.dot" aria-hidden="true"></span>
          <span class="account-copy">
            <strong>{{ selectedTunnel?.name || '选择隧道' }}</strong>
            <small class="mono">{{ selectedId ? shortId(selectedId) : '—' }}</small>
          </span>
          <span class="chevron"><Icon name="chevron-down" /></span>
        </button>
        <Transition name="select-pop">
          <div v-if="pickerOpen" class="picker-menu" role="listbox" aria-label="隧道列表">
            <button
              v-for="tunnel in app.tunnels"
              :key="tunnel.id"
              class="picker-item"
              type="button"
              role="option"
              :aria-selected="tunnel.id === selectedId"
              :class="{ 'is-selected': tunnel.id === selectedId }"
              @click="selectTunnel(tunnel.id)"
            >
              <span
                class="status-dot"
                :class="statusMetaFor(tunnelDisplayState(tunnel)).dot"
                aria-hidden="true"
              ></span>
              <span class="picker-copy">
                <strong>{{ tunnel.name }}</strong>
                <small class="mono">{{ shortId(tunnel.id) }}</small>
              </span>
              <span v-if="tunnel.id === liveTunnelId" class="pill info">运行中</span>
              <Icon v-if="tunnel.id === selectedId" name="check" />
            </button>
          </div>
        </Transition>
      </div>
      <StatusPill :label="`${draft.length} 条规则`" kind="info" />
      <span class="micro">自动 DNS 已开启</span>
      <div class="spacer"></div>
      <span class="micro">拖动手柄可调整匹配顺序</span>
    </div>

    <NoticeBar v-if="tunnelNeedsRun && selectedTunnel" tone="info" icon="info" title="隧道未运行">
      该隧道当前没有活动连接；路由会写入 Cloudflare 配置，在下次连接时生效。
    </NoticeBar>

    <div class="routes-layout">
      <section class="card route-list reveal" style="--i: 2">
        <div v-if="loading && !loaded" class="route-skeleton">
          <div v-for="index in 4" :key="index" class="route-row">
            <SkeletonBlock height="12px" width="16px" />
            <SkeletonBlock height="16px" width="16px" />
            <SkeletonBlock height="28px" width="80%" />
            <SkeletonBlock height="10px" width="18px" />
            <SkeletonBlock height="24px" width="70%" />
            <SkeletonBlock height="20px" width="48px" />
          </div>
        </div>
        <template v-else>
          <EmptyState
            v-if="!draft.length"
            icon="branch"
            title="还没有路由规则"
            message="添加一条 域名 → 内网服务 的映射，保存后立即写入隧道配置。"
          >
            <button class="btn primary" type="button" @click="openAdd">
              <Icon name="plus" />
              添加路由
            </button>
          </EmptyState>
          <RouteRow
            v-for="(item, index) in draft"
            :key="item.hostname"
            :route="item"
            :index="index"
            :dns-state="dns[item.hostname] || 'unknown'"
            :dragging="draggingId === item.hostname"
            :drag-over="overId === item.hostname"
            :busy="dnsBusy === item.hostname"
            @edit="openEdit"
            @remove="removeRoute"
            @dns="createDnsFor"
            @move="moveRoute"
            @dragstart="onDragStart($event, item)"
            @dragover="onDragOver($event, item)"
            @dragleave="overId = ''"
            @drop="onDrop($event, item)"
            @dragend="onDragEnd"
          />
          <div class="route-row is-catchall">
            <span class="route-index">—</span>
            <span class="drag-handle is-locked" aria-hidden="true"><Icon name="lock" /></span>
            <div class="route-host">
              <strong>*</strong>
              <small>兜底规则，始终最后</small>
            </div>
            <span class="route-arrow"><Icon name="arrow-right" /></span>
            <input
              v-model="catchAll"
              class="input mono catchall-input"
              aria-label="兜底服务"
              spellcheck="false"
              placeholder="http_status:404"
              @input="onCatchAllInput"
            />
            <StatusPill label="固定" kind="neutral" />
            <div class="route-actions"></div>
          </div>
        </template>
        <div class="route-note">
          <Icon name="info" />
          规则按顺序匹配，首条命中后不再继续；最后一条为兜底。
        </div>
        <div class="dns-summary">
          <span v-if="dnsCounts.created" class="pill healthy">{{ dnsCounts.created }} 已创建</span>
          <span v-if="dnsCounts.updated" class="pill info">{{ dnsCounts.updated }} 已更新</span>
          <span v-if="dnsCounts.unchanged" class="pill info">{{ dnsCounts.unchanged }} 已存在</span>
          <span v-if="dnsCounts.pending" class="pill warning">{{ dnsCounts.pending }} 创建中</span>
          <span v-if="dnsCounts.failed" class="pill danger">{{ dnsCounts.failed }} 失败</span>
          <span v-if="dnsCounts.unknown" class="pill neutral">{{ dnsCounts.unknown }} 未检查</span>
          <span class="micro">CNAME 指向当前隧道的 cfargotunnel.com</span>
          <span v-if="lastZone" class="micro">Zone {{ lastZone }}</span>
          <span v-if="dnsError" class="micro dns-error">{{ dnsError }}</span>
        </div>
      </section>

      <aside class="card yaml-card reveal" style="--i: 3">
        <div class="yaml-head">
          <div class="file"><Icon name="code" />config.yml</div>
          <button class="icon-btn" type="button" aria-label="复制 YAML" title="复制 YAML" @click="copyYaml">
            <Icon name="copy" />
          </button>
        </div>
        <div class="yaml-code" id="yaml-code">
          <div v-for="(line, index) in yaml" :key="index" class="yaml-line">
            <span
              v-for="(segment, segIndex) in line"
              :key="segIndex"
              :class="{
                'y-key': segment.t === 'key',
                'y-val': segment.t === 'val',
                'y-cmt': segment.t === 'cmt',
                'y-dash': segment.t === 'dash',
              }"
              >{{ segment.v }}</span
            >
          </div>
        </div>
        <div class="yaml-foot">
          <span>由路由表实时生成 · {{ rawIngressCount }} 条 ingress</span>
          <span>只读预览</span>
        </div>
      </aside>
    </div>
  </template>

  <RouteDrawer
    v-if="drawer"
    :mode="drawer.mode"
    :initial="drawer.initial"
    :tunnel-id="selectedId"
    :tunnel-name="selectedTunnel?.name || ''"
    :existing-hostnames="hostnames"
    :auto-dns-default="drawer.mode === 'add'"
    :submitting="false"
    server-error=""
    @close="drawer = null"
    @save="applyDrawer"
  />
</template>
