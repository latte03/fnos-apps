import { computed, reactive } from 'vue'
import { api } from '../lib/client'
import { ApiError, apiErrorMessage, type EventStreamHandle } from '../lib/api'
import { toast } from '../lib/toast'
import type {
  AccountInfo,
  Activity,
  Health,
  LogLine,
  Meta,
  Overview,
  Runtime,
  RuntimeMetrics,
  RuntimeState,
  Settings,
  Tunnel,
} from '../types'

export type SseState = 'connecting' | 'open' | 'closed'

export interface AppState {
  booting: boolean
  bootError: string
  health: Health | null
  meta: Meta | null
  settings: Settings | null
  account: AccountInfo | null
  tunnels: Tunnel[]
  overview: Overview | null
  runtime: Runtime | null
  metrics: RuntimeMetrics | null
  metricSamples: RuntimeMetrics[]
  activity: Activity[]
  logs: LogLine[]
  logDropped: number
  logsLoaded: boolean
  loadingOverview: boolean
  loadingTunnels: boolean
  loadingLogs: boolean
  connectingId: string
  disconnecting: boolean
  sse: SseState
  sseAttempt: number
  sessionExpired: boolean
  needsSetup: boolean
  routeCount: number
}

export const app = reactive<AppState>({
  booting: true,
  bootError: '',
  health: null,
  meta: null,
  settings: null,
  account: null,
  tunnels: [],
  overview: null,
  runtime: null,
  metrics: null,
  metricSamples: [],
  activity: [],
  logs: [],
  logDropped: 0,
  logsLoaded: false,
  loadingOverview: false,
  loadingTunnels: false,
  loadingLogs: false,
  connectingId: '',
  disconnecting: false,
  sse: 'connecting',
  sseAttempt: 0,
  sessionExpired: false,
  needsSetup: false,
  routeCount: 0,
})

export const isDemo = computed(() => app.health?.demo === true)
export const cloudflaredInstalled = computed(() => app.health?.cloudflared?.installed !== false)

export const activeTunnel = computed<Tunnel | null>(() => {
  const id = app.runtime?.tunnelId
  if (!id) return null
  return app.tunnels.find((tunnel) => tunnel.id === id) || null
})

export const runtimeState = computed<RuntimeState>(() => app.runtime?.state ?? 'disconnected')

export const isLive = computed(
  () => runtimeState.value === 'connected' || runtimeState.value === 'connecting' || runtimeState.value === 'reconnecting',
)

export function noteError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 401) app.sessionExpired = true
    if (err.status === 412) app.needsSetup = true
  }
  return apiErrorMessage(err)
}

let stream: EventStreamHandle | null = null
let pendingConnectName = ''
let pendingDisconnect = false

function pushLog(line: LogLine) {
  const last = app.logs[app.logs.length - 1]
  if (last && line.seq <= last.seq) return
  if (last && line.seq > last.seq + 1) app.logDropped += line.seq - last.seq - 1
  app.logs.push(line)
  const cap = app.settings?.maxLogLines ?? 2000
  if (app.logs.length > cap) {
    app.logs.splice(0, app.logs.length - cap)
    app.logDropped += 1
  }
}

export function applyRuntime(runtime: Runtime) {
  const previous = app.runtime?.state
  app.runtime = runtime
  const live = runtime.state !== 'disconnected'
  for (const tunnel of app.tunnels) {
    tunnel.localActive = live && tunnel.id === runtime.tunnelId
    tunnel.connections = tunnel.localActive ? runtime.connections : 0
  }
  if (runtime.state !== 'disconnected' && app.needsSetup) app.needsSetup = false
  if (runtime.state === 'connected' && previous !== 'connected' && pendingConnectName) {
    toast(`隧道 ${runtime.tunnelName || pendingConnectName} 已连接`, 'success')
    pendingConnectName = ''
  }
  if (runtime.state === 'disconnected' && pendingDisconnect) {
    toast('已断开连接', 'success')
    pendingDisconnect = false
  }
  if (runtime.state === 'error' && runtime.lastError && previous !== 'error') {
    toast(`隧道异常：${runtime.lastError}`, 'error')
  }
}

export function setRouteCount(count: number) {
  app.routeCount = count
}

function mergeRecent(lines: LogLine[]) {
  for (const line of [...lines].sort((a, b) => a.seq - b.seq)) {
    if (!app.logs.some((existing) => existing.seq === line.seq)) app.logs.push(line)
  }
  app.logs.sort((a, b) => a.seq - b.seq)
}

export function startEvents() {
  if (stream) return
  app.sse = 'connecting'
  stream = api.openEvents({
    onOpen: () => {
      app.sse = 'open'
      app.sseAttempt = 0
    },
    onError: (attempt) => {
      app.sse = 'closed'
      app.sseAttempt = attempt
    },
    onHello: (status, recent) => {
      applyRuntime(status)
      mergeRecent(recent)
      app.sse = 'open'
    },
    onStatus: (status) => applyRuntime(status),
    onLog: (line) => pushLog(line),
    onActivity: (activity) => {
      app.activity = [activity, ...app.activity.filter((item) => item.id !== activity.id)].slice(0, 60)
    },
    onMetrics: (metrics) => {
      app.metrics = metrics
      app.metricSamples = [...app.metricSamples, metrics].slice(-14)
    },
  })
}

export function stopEvents() {
  stream?.close()
  stream = null
}

export async function bootstrap(): Promise<void> {
  app.booting = true
  app.bootError = ''
  try {
    app.health = await api.health()
  } catch (err) {
    app.bootError = noteError(err)
    app.booting = false
    return
  }

  const [meta, settings, tunnels, overview, account, runtime, metrics] = await Promise.allSettled([
    api.meta(),
    api.getSettings(),
    api.tunnels(),
    api.overview(),
    api.account(),
    api.runtime(),
    api.metrics(),
  ])

  if (meta.status === 'fulfilled') app.meta = meta.value
  if (settings.status === 'fulfilled') app.settings = settings.value
  if (tunnels.status === 'fulfilled') app.tunnels = tunnels.value.tunnels
  if (overview.status === 'fulfilled') {
    app.overview = overview.value
    app.activity = overview.value.activity
    app.routeCount = overview.value.routeCount
  }
  if (account.status === 'fulfilled') app.account = account.value
  if (runtime.status === 'fulfilled') app.runtime = runtime.value
  if (metrics.status === 'fulfilled') {
    app.metrics = metrics.value
    app.metricSamples = [metrics.value]
  }

  const firstRejection = [meta, settings, tunnels, overview, account, runtime, metrics].find(
    (result) => result.status === 'rejected',
  )
  if (firstRejection && firstRejection.status === 'rejected') noteError(firstRejection.reason)

  app.booting = false
  startEvents()
}

export async function refreshTunnels(): Promise<string | null> {
  app.loadingTunnels = true
  try {
    const result = await api.tunnels()
    app.tunnels = result.tunnels
    return null
  } catch (err) {
    return noteError(err)
  } finally {
    app.loadingTunnels = false
  }
}

export async function refreshOverview(): Promise<string | null> {
  app.loadingOverview = true
  try {
    const result = await api.overview()
    app.overview = result
    app.routeCount = result.routeCount
    if (!app.activity.length) app.activity = result.activity
    if (app.runtime) applyRuntime(result.runtime)
    else app.runtime = result.runtime
    return null
  } catch (err) {
    return noteError(err)
  } finally {
    app.loadingOverview = false
  }
}

export async function refreshRuntime(): Promise<string | null> {
  try {
    applyRuntime(await api.runtime())
    return null
  } catch (err) {
    return noteError(err)
  }
}

export async function refreshActivity(): Promise<void> {
  try {
    const result = await api.activity()
    app.activity = result.items
  } catch {
    /* the live stream keeps activity fresh; a failed catch-up is not fatal */
  }
}

export async function refreshSettings(): Promise<string | null> {
  try {
    app.settings = await api.getSettings()
    return null
  } catch (err) {
    return noteError(err)
  }
}

export async function loadLogs(limit = 500): Promise<string | null> {
  app.loadingLogs = true
  try {
    const page = await api.logs({ limit })
    app.logs = page.lines
    app.logDropped = page.dropped
    app.logsLoaded = true
    return null
  } catch (err) {
    return noteError(err)
  } finally {
    app.loadingLogs = false
  }
}

export async function clearLogs(): Promise<string | null> {
  try {
    await api.clearLogs()
    app.logs = []
    app.logDropped = 0
    return null
  } catch (err) {
    return noteError(err)
  }
}

export async function connectTunnel(id: string, name: string): Promise<void> {
  if (app.connectingId) throw new ApiError(409, 'busy', '已有连接操作正在进行。')
  app.connectingId = id
  pendingConnectName = name
  try {
    const result = await api.connect(id)
    applyRuntime(result.status)
  } catch (err) {
    pendingConnectName = ''
    throw err
  } finally {
    app.connectingId = ''
  }
}

export async function disconnectTunnel(): Promise<void> {
  if (app.disconnecting) throw new ApiError(409, 'busy', '已有断开操作正在进行。')
  app.disconnecting = true
  pendingDisconnect = true
  try {
    const result = await api.disconnect()
    applyRuntime(result.status)
  } catch (err) {
    pendingDisconnect = false
    throw err
  } finally {
    app.disconnecting = false
  }
}

export async function createTunnel(name: string): Promise<Tunnel | null> {
  app.loadingTunnels = true
  try {
    const tunnel = await api.createTunnel(name)
    await refreshTunnels()
    return tunnel
  } finally {
    app.loadingTunnels = false
  }
}

export async function deleteTunnel(tunnel: Tunnel): Promise<void> {
  await api.deleteTunnel(tunnel.id)
  await refreshTunnels()
  await refreshOverview()
}

export type TunnelDisplayState = 'connected' | 'connecting' | 'error' | 'idle'

export function tunnelDisplayState(tunnel: Tunnel): TunnelDisplayState {
  const runtime = app.runtime
  if (runtime && runtime.tunnelId === tunnel.id && runtime.state !== 'disconnected') {
    if (runtime.state === 'connected') return 'connected'
    if (runtime.state === 'error') return 'error'
    return 'connecting'
  }
  if (tunnel.status === 'error' || tunnel.status === 'down' || tunnel.status === 'degraded') return 'error'
  return 'idle'
}

export interface StatusMeta {
  label: string
  pill: 'healthy' | 'warning' | 'danger' | 'neutral' | 'info'
  dot: 'healthy' | 'warning' | 'danger' | ''
  micro: string
}

export function statusMetaFor(state: 'connected' | 'connecting' | 'error' | 'idle' | 'disconnected'): StatusMeta {
  switch (state) {
    case 'connected':
      return { label: '已连接', pill: 'healthy', dot: 'healthy', micro: '已连接 · 当前' }
    case 'connecting':
      return { label: '连接中', pill: 'warning', dot: 'warning', micro: '正在建立连接' }
    case 'error':
      return { label: '异常', pill: 'danger', dot: 'danger', micro: '连接中断' }
    default:
      return { label: '未连接', pill: 'neutral', dot: '', micro: '等待连接' }
  }
}

export function runtimeStatusMeta(runtime: Runtime | null): StatusMeta {
  const state = runtime?.state ?? 'disconnected'
  if (state === 'connected') return statusMetaFor('connected')
  if (state === 'error') return statusMetaFor('error')
  if (state === 'connecting' || state === 'reconnecting' || state === 'stopping') return statusMetaFor('connecting')
  return statusMetaFor('disconnected')
}

export function activityIcon(kind: string): string {
  switch (kind) {
    case 'connect':
      return 'link'
    case 'disconnect':
      return 'plug'
    case 'route':
      return 'branch'
    case 'dns':
      return 'cloud'
    case 'settings':
      return 'sliders'
    case 'create':
      return 'plus'
    case 'delete':
      return 'trash'
    case 'key':
      return 'key'
    default:
      return 'activity'
  }
}
