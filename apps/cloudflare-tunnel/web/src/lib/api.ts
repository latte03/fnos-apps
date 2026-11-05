import type {
  AccountInfo,
  ActivityList,
  ApiKeyCreated,
  ApiKeyList,
  ApiErrorEnvelope,
  DnsResult,
  Health,
  LogLevelFilter,
  LogLine,
  LogPage,
  Meta,
  Overview,
  RoutesResponse,
  RoutesUpdate,
  Runtime,
  RuntimeConnectResult,
  RuntimeMetrics,
  Settings,
  SettingsUpdate,
  Tunnel,
  TunnelList,
  TunnelToken,
  VerifyRequest,
  VerifyResult,
} from '../types'

const API_BASE = '/api/v1'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: unknown

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }

  get kind(): 'unauthorized' | 'setup' | 'upstream' | 'network' | 'server' | 'validation' | 'unknown' {
    if (this.status === 401) return 'unauthorized'
    if (this.status === 412) return 'setup'
    if (this.status === 502 || this.status === 504) return 'upstream'
    if (this.status === 0) return 'network'
    if (this.status === 400 || this.status === 409 || this.status === 404) return 'validation'
    return 'server'
  }
}

export function apiErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.kind === 'network') return '无法连接应用服务，请确认 cloudflare-tunnel 正在运行。'
    if (err.kind === 'upstream') {
      return err.message || 'Cloudflare API 暂时不可用，请稍后重试。'
    }
    if (err.kind === 'unauthorized') return '会话已过期，请刷新页面重新建立会话。'
    if (err.kind === 'setup') return '尚未完成配置：请先填写 Cloudflare 账户与 API Token。'
    return err.message || `请求失败（HTTP ${err.status}）`
  }
  if (err instanceof Error) return err.message
  return '请求失败，请稍后重试。'
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  headers.set('Accept', 'application/json')

  let res: Response
  try {
    res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
      credentials: 'include',
      signal: AbortSignal.timeout(25000),
    })
  } catch (err) {
    const aborted = err instanceof DOMException && err.name === 'TimeoutError'
    throw new ApiError(0, aborted ? 'timeout' : 'network', aborted ? '请求超时，请稍后重试。' : '网络不可达')
  }

  if (res.status === 204) return undefined as T

  const text = await res.text()
  let payload: unknown = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = null
    }
  }

  if (!res.ok) {
    const envelope = payload as ApiErrorEnvelope | null
    const code = envelope?.error?.code || `http_${res.status}`
    const message = envelope?.error?.message || res.statusText || `HTTP ${res.status}`
    throw new ApiError(res.status, code, message, envelope?.error?.details)
  }

  return payload as T
}

export interface StreamHandlers {
  onHello?: (status: Runtime, recent: LogLine[]) => void
  onStatus?: (status: Runtime) => void
  onLog?: (line: LogLine) => void
  onActivity?: (activity: ActivityList['items'][number]) => void
  onMetrics?: (metrics: RuntimeMetrics) => void
  onOpen?: () => void
  onError?: (attempt: number) => void
}

export interface EventStreamHandle {
  close: () => void
}

export interface ApiClient {
  health(): Promise<Health>
  meta(): Promise<Meta>
  overview(): Promise<Overview>
  getSettings(): Promise<Settings>
  updateSettings(body: SettingsUpdate): Promise<Settings>
  verify(body: VerifyRequest): Promise<VerifyResult>
  account(): Promise<AccountInfo>
  tunnels(): Promise<TunnelList>
  createTunnel(name: string): Promise<Tunnel>
  deleteTunnel(id: string): Promise<void>
  tunnelToken(id: string): Promise<TunnelToken>
  routes(tunnelId: string): Promise<RoutesResponse>
  saveRoutes(tunnelId: string, body: RoutesUpdate): Promise<RoutesResponse>
  createDns(tunnelId: string, hostname: string): Promise<DnsResult>
  runtime(): Promise<Runtime>
  connect(tunnelId: string): Promise<RuntimeConnectResult>
  disconnect(): Promise<RuntimeConnectResult>
  metrics(): Promise<RuntimeMetrics>
  logs(params: { limit?: number; afterSeq?: number; level?: LogLevelFilter; q?: string }): Promise<LogPage>
  clearLogs(): Promise<void>
  logsDownloadUrl(): string
  activity(): Promise<ActivityList>
  keys(): Promise<ApiKeyList>
  createKey(name: string): Promise<ApiKeyCreated>
  deleteKey(id: string): Promise<void>
  openEvents(handlers: StreamHandlers): EventStreamHandle
}

const enc = encodeURIComponent

export const realApi: ApiClient = {
  health: () => request<Health>('/health'),
  meta: () => request<Meta>('/meta'),
  overview: () => request<Overview>('/overview'),
  getSettings: () => request<Settings>('/settings'),
  updateSettings: (body) => request<Settings>('/settings', { method: 'PUT', body: JSON.stringify(body) }),
  verify: (body) => request<VerifyResult>('/settings/verify', { method: 'POST', body: JSON.stringify(body) }),
  account: () => request<AccountInfo>('/account'),
  tunnels: () => request<TunnelList>('/tunnels'),
  createTunnel: (name) => request<Tunnel>('/tunnels', { method: 'POST', body: JSON.stringify({ name }) }),
  deleteTunnel: (id) => request<void>(`/tunnels/${enc(id)}`, { method: 'DELETE' }),
  tunnelToken: (id) => request<TunnelToken>(`/tunnels/${enc(id)}/token`),
  routes: (tunnelId) => request<RoutesResponse>(`/tunnels/${enc(tunnelId)}/routes`),
  saveRoutes: (tunnelId, body) =>
    request<RoutesResponse>(`/tunnels/${enc(tunnelId)}/routes`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }),
  createDns: (tunnelId, hostname) =>
    request<DnsResult>('/dns', { method: 'POST', body: JSON.stringify({ tunnelId, hostname }) }),
  runtime: () => request<Runtime>('/runtime'),
  connect: (tunnelId) =>
    request<RuntimeConnectResult>('/runtime/connect', {
      method: 'POST',
      body: JSON.stringify({ tunnelId }),
    }),
  disconnect: () => request<RuntimeConnectResult>('/runtime/disconnect', { method: 'POST' }),
  metrics: () => request<RuntimeMetrics>('/runtime/metrics'),
  logs: ({ limit = 500, afterSeq = 0, level = 'all', q = '' } = {}) => {
    const params = new URLSearchParams({ limit: String(limit), afterSeq: String(afterSeq), level, q })
    return request<LogPage>(`/logs?${params.toString()}`)
  },
  clearLogs: () => request<void>('/logs/clear', { method: 'POST' }),
  logsDownloadUrl: () => `${API_BASE}/logs/download`,
  activity: () => request<ActivityList>('/activity'),
  keys: () => request<ApiKeyList>('/keys'),
  createKey: (name) => request<ApiKeyCreated>('/keys', { method: 'POST', body: JSON.stringify({ name }) }),
  deleteKey: (id) => request<void>(`/keys/${enc(id)}`, { method: 'DELETE' }),
  openEvents: (handlers) => openEventSource(handlers),
}

/** One EventSource for the whole app, exponential backoff, closed on demand. */
function openEventSource(handlers: StreamHandlers): EventStreamHandle {
  let source: EventSource | null = null
  let retryTimer: number | null = null
  let attempt = 0
  let closed = false

  const dispatch = (type: string, data: string) => {
    let payload: unknown
    try {
      payload = JSON.parse(data)
    } catch {
      return
    }
    try {
      switch (type) {
        case 'hello': {
          const body = payload as { status: Runtime; recent?: LogLine[] }
          handlers.onHello?.(body.status, body.recent || [])
          break
        }
        case 'status':
          handlers.onStatus?.(payload as Runtime)
          break
        case 'log':
          handlers.onLog?.(payload as LogLine)
          break
        case 'activity':
          handlers.onActivity?.(payload as ActivityList['items'][number])
          break
        case 'metrics':
          handlers.onMetrics?.(payload as RuntimeMetrics)
          break
        default:
          break
      }
    } catch (err) {
      console.error('[events] handler failed', err)
    }
  }

  const connect = () => {
    if (closed) return
    source = new EventSource(`${API_BASE}/events`, { withCredentials: true })
    const named = ['hello', 'status', 'log', 'activity', 'metrics'] as const
    for (const type of named) {
      source.addEventListener(type, (event) => {
        dispatch(type, (event as MessageEvent<string>).data)
      })
    }
    source.onopen = () => {
      attempt = 0
      handlers.onOpen?.()
    }
    source.onerror = () => {
      if (closed) return
      source?.close()
      source = null
      attempt += 1
      handlers.onError?.(attempt)
      const delay = Math.min(15000, 1000 * 2 ** Math.min(attempt - 1, 4))
      if (retryTimer !== null) window.clearTimeout(retryTimer)
      retryTimer = window.setTimeout(connect, delay)
    }
  }

  connect()

  return {
    close: () => {
      closed = true
      if (retryTimer !== null) window.clearTimeout(retryTimer)
      source?.close()
      source = null
    },
  }
}
