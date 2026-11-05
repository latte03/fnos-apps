/**
 * Mock adapter for `pnpm dev` (VITE_USE_MOCK=1).
 *
 * Shapes are byte-for-byte the CONTRACT DTOs; secrets are fake and never reach
 * production builds. Every mutation the real server offers is simulated,
 * including SSE traffic, so the UI is fully clickable without a backend.
 */
import { ApiError, type ApiClient, type EventStreamHandle, type StreamHandlers } from './api'
import type {
  AccountInfo,
  Activity,
  ApiKey,
  ApiKeyCreated,
  ApiKeyList,
  DnsResult,
  Health,
  LogLevel,
  LogLine,
  LogPage,
  Meta,
  Overview,
  Route,
  RoutesResponse,
  Runtime,
  RuntimeMetrics,
  Settings,
  Tunnel,
  TunnelList,
} from '../types'

const VERSION = '2026.10.7-3'
const CLOUDFLARED = '2026.10.7'
const STARTED_AT = new Date(Date.now() - 1000 * 60 * 60 * 27).toISOString()

const iso = (offsetMs = 0) => new Date(Date.now() + offsetMs).toISOString()

const randomHex = (length: number) => {
  const bytes = new Uint8Array(Math.ceil(length / 2))
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('').slice(0, length)
}

const TUNNEL_MAIN = '8f3c2a1e-4d8b-4f1a-9b2c-7e5d6a1b3c90'
const TUNNEL_LAB = '2b7d9c44-6f0e-4c2a-8d1b-5a3e7f9c2d41'
const TUNNEL_BACKUP = 'c91e5f70-8a2b-4d6c-b3e9-1f4a6c8d0e72'

function delay(ms = 260): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

const tunnels: Tunnel[] = [
  { id: TUNNEL_MAIN, name: 'fnos-main', status: 'healthy', createdAt: iso(-1000 * 60 * 60 * 24 * 48), connections: 2, configSrc: 'cloudflare', localActive: true },
  { id: TUNNEL_LAB, name: 'home-lab', status: 'inactive', createdAt: iso(-1000 * 60 * 60 * 24 * 34), connections: 0, configSrc: 'cloudflare', localActive: false },
  { id: TUNNEL_BACKUP, name: 'edge-backup', status: 'error', createdAt: iso(-1000 * 60 * 60 * 24 * 15), connections: 0, configSrc: 'cloudflare', localActive: false },
]

const routesByTunnel: Record<string, Route[]> = {
  [TUNNEL_MAIN]: [
    { hostname: 'nas.example.com', path: '/', service: 'http://127.0.0.1:5666' },
    { hostname: 'ha.example.com', path: '/', service: 'http://192.168.0.42:8123' },
    { hostname: 'ai.example.com', path: '/', service: 'http://127.0.0.1:3000' },
    { hostname: 'pa.example.com', path: '/', service: 'http://127.0.0.1:19091' },
  ],
  [TUNNEL_LAB]: [
    { hostname: 'home.example.com', path: '/', service: 'http://192.168.0.42:8123' },
    { hostname: 'files.example.com', path: '/', service: 'http://192.168.0.42:8080' },
  ],
  [TUNNEL_BACKUP]: [{ hostname: 'backup.example.com', path: '/', service: 'http://127.0.0.1:9000' }],
}

let runtime: Runtime = {
  state: 'connected',
  tunnelId: TUNNEL_MAIN,
  tunnelName: 'fnos-main',
  connectedAt: iso(-1000 * (3 * 86400 + 17 * 3600 + 8 * 60 + 21)),
  uptimeSec: 3 * 86400 + 17 * 3600 + 8 * 60 + 21,
  protocol: 'quic',
  edgeLocation: 'HKG',
  connections: 2,
  reconnectAttempt: 0,
  lastError: '',
  cloudflaredVersion: CLOUDFLARED,
}

let settings: Settings = {
  accountId: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
  apiTokenSet: true,
  apiTokenPreview: '••••1d8a',
  protocol: 'quic',
  edgeIpVersion: 'auto',
  autoConnect: true,
  autoReconnect: true,
  maxReconnect: 10,
  logRetentionDays: 30,
  maxLogLines: 2000,
}

let fakeToken = 'cf_demo_' + randomHex(24)

let metrics: RuntimeMetrics = {
  requests: 12400,
  errors: 3,
  connections: 2,
  rttMs: 38,
  sampledAt: iso(),
}

let seq = 0

function makeLine(level: LogLevel, message: string, fields: Record<string, unknown>, offsetMs = 0): LogLine {
  seq += 1
  return { seq, ts: iso(offsetMs), level, message, fields }
}

const logPool: Array<{ level: LogLevel; message: string; fields: Record<string, unknown> }> = [
  { level: 'info', message: 'Registered tunnel connection', fields: { connIndex: 0, location: 'HKG', protocol: 'quic' } },
  { level: 'info', message: 'Request completed', fields: { method: 'GET', host: 'nas.example.com', status: 200, duration: '42ms' } },
  { level: 'info', message: 'Request completed', fields: { method: 'GET', host: 'ha.example.com', status: 200, duration: '61ms' } },
  { level: 'info', message: 'Updated to new configuration', fields: { config_version: 7 } },
  { level: 'warn', message: 'Connection terminated', fields: { connIndex: 1 } },
  { level: 'info', message: 'Retrying connection in 1s', fields: { connIndex: 1 } },
  { level: 'error', message: 'Unable to reach the origin service', fields: { host: 'ai.example.com', error: 'connection refused' } },
  { level: 'info', message: 'Heartbeat acknowledged', fields: { connIndex: 0 } },
  { level: 'debug', message: 'Metrics scrape completed', fields: { addr: '127.0.0.1:20241' } },
]

let logs: LogLine[] = []
for (const seed of [
  { level: 'info' as LogLevel, message: 'Registered tunnel connection', fields: { connIndex: 0, location: 'HKG', protocol: 'quic' }, ago: 26 },
  { level: 'info' as LogLevel, message: 'Updated to new configuration', fields: { config_version: 7 }, ago: 23 },
  { level: 'info' as LogLevel, message: 'Request completed', fields: { method: 'GET', host: 'nas.example.com', status: 200, duration: '42ms' }, ago: 19 },
  { level: 'warn' as LogLevel, message: 'Connection terminated', fields: { connIndex: 1 }, ago: 15 },
  { level: 'info' as LogLevel, message: 'Retrying connection in 1s', fields: { connIndex: 1 }, ago: 13 },
  { level: 'info' as LogLevel, message: 'Registered tunnel connection', fields: { connIndex: 1, location: 'HKG', protocol: 'quic' }, ago: 11 },
  { level: 'error' as LogLevel, message: 'Unable to reach the origin service', fields: { host: 'ai.example.com', error: 'connection refused' }, ago: 8 },
  { level: 'info' as LogLevel, message: 'Request completed', fields: { method: 'GET', host: 'ha.example.com', status: 200, duration: '61ms' }, ago: 5 },
  { level: 'debug' as LogLevel, message: 'Metrics server listening', fields: { addr: '127.0.0.1:20241' }, ago: 3 },
  { level: 'info' as LogLevel, message: 'Heartbeat acknowledged', fields: { connIndex: 0 }, ago: 1 },
]) {
  logs.push(makeLine(seed.level, seed.message, seed.fields, -seed.ago * 1000))
}

let dropped = 0

let activities: Activity[] = [
  { id: 'a4', ts: iso(-1000 * 60 * 2), kind: 'connect', title: '隧道 fnos-main 已连接', tone: 'green' },
  { id: 'a3', ts: iso(-1000 * 60 * 18), kind: 'route', title: '路由 ai.example.com 已更新', tone: 'blue' },
  { id: 'a2', ts: iso(-1000 * 60 * 61), kind: 'dns', title: 'CNAME pa.example.com 已创建', tone: 'teal' },
  { id: 'a1', ts: iso(-1000 * 60 * 180), kind: 'settings', title: '协议切换为 QUIC', tone: 'neutral' },
]

let activitySeq = activities.length

let keys: ApiKey[] = [
  { id: 'k1', name: 'Home Assistant', prefix: 'ct_ab12', createdAt: iso(-1000 * 60 * 60 * 24 * 12), lastUsedAt: iso(-1000 * 60 * 60 * 26) },
  { id: 'k2', name: 'n8n 自动化', prefix: 'ct_7f3c', createdAt: iso(-1000 * 60 * 60 * 24 * 4), lastUsedAt: '' },
]

const dnsKnown = new Map<string, DnsResult>()

const statusListeners = new Set<(rt: Runtime) => void>()
const logListeners = new Set<(line: LogLine) => void>()
const metricListeners = new Set<(m: RuntimeMetrics) => void>()
const activityListeners = new Set<(a: Activity) => void>()

let intervalsStarted = false

function runtimeView(): Runtime {
  const uptimeSec =
    runtime.state === 'connected' && runtime.connectedAt
      ? Math.max(0, Math.floor((Date.now() - Date.parse(runtime.connectedAt)) / 1000))
      : runtime.uptimeSec
  return { ...runtime, uptimeSec }
}

function emitStatus() {
  const view = runtimeView()
  for (const listener of statusListeners) listener(view)
}

function emitActivity(activity: Activity) {
  for (const listener of activityListeners) listener(activity)
}

function pushActivity(kind: string, title: string, tone: Activity['tone']) {
  activitySeq += 1
  const item: Activity = { id: `a${activitySeq}`, ts: iso(), kind, title, tone }
  activities = [item, ...activities].slice(0, 100)
  emitActivity(item)
}

function pushLog(line: LogLine) {
  logs.push(line)
  if (logs.length > 2000) {
    logs.shift()
    dropped += 1
  }
  for (const listener of logListeners) listener(line)
}

function syncTunnels() {
  const live = runtime.state !== 'disconnected'
  for (const tunnel of tunnels) {
    tunnel.localActive = live && tunnel.id === runtime.tunnelId
    tunnel.connections = tunnel.localActive ? runtime.connections : 0
  }
}

function startIntervals() {
  if (intervalsStarted) return
  intervalsStarted = true

  window.setInterval(() => {
    if (runtime.state === 'disconnected') return
    const seed = logPool[Math.floor(Math.random() * logPool.length)]
    const fields = { ...seed.fields }
    const message = seed.message
    pushLog(makeLine(seed.level, message, fields))
  }, 1800)

  window.setInterval(() => {
    if (runtime.state === 'disconnected') {
      metrics = { requests: metrics.requests, errors: metrics.errors, connections: 0, rttMs: null, sampledAt: iso() }
    } else {
      metrics = {
        requests: (metrics.requests ?? 0) + Math.floor(20 + Math.random() * 120),
        errors: (metrics.errors ?? 0) + (Math.random() < 0.12 ? 1 : 0),
        connections: runtime.connections,
        rttMs: Math.round(30 + Math.random() * 16),
        sampledAt: iso(),
      }
    }
    for (const listener of metricListeners) listener({ ...metrics })
  }, 5000)
}

function validateHostname(hostname: string) {
  return /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$/i.test(hostname)
}

function validateService(service: string) {
  return /^(?:https?|tcp|ssh|rdp|smb):\/\/\S+|^unix:\S+/.test(service)
}

function validateAccountId(accountId: string) {
  return accountId === '' || /^[0-9a-f]{32}$/i.test(accountId)
}

function routesResponse(tunnelId: string): RoutesResponse {
  const routes = (routesByTunnel[tunnelId] || []).map((route) => ({ ...route }))
  return { routes, catchAllService: 'http_status:404', rawIngressCount: routes.length + 1 }
}

function scheduleConnected(tunnelId: string) {
  window.setTimeout(() => {
    if (runtime.tunnelId !== tunnelId) return
    const tunnel = tunnels.find((item) => item.id === tunnelId)
    runtime = {
      ...runtime,
      state: 'connected',
      connectedAt: iso(),
      uptimeSec: 0,
      connections: 2,
      edgeLocation: 'HKG',
      protocol: settings.protocol === 'auto' ? 'quic' : settings.protocol,
      reconnectAttempt: 0,
      lastError: '',
    }
    if (tunnel) {
      tunnel.status = 'healthy'
      pushActivity('connect', `隧道 ${tunnel.name} 已连接`, 'green')
    }
    syncTunnels()
    emitStatus()
    for (const listener of metricListeners) listener({ ...metrics, connections: 2, sampledAt: iso() })
  }, 1200)
}

function stopRuntime() {
  const tunnel = tunnels.find((item) => item.id === runtime.tunnelId)
  runtime = {
    ...runtime,
    state: 'disconnected',
    tunnelId: '',
    tunnelName: '',
    connectedAt: '',
    uptimeSec: 0,
    connections: 0,
    edgeLocation: '',
    reconnectAttempt: 0,
  }
  if (tunnel) pushActivity('disconnect', `隧道 ${tunnel.name} 已断开`, 'danger')
  syncTunnels()
  emitStatus()
}

export function createMockApi(): ApiClient {
  return {
    async health(): Promise<Health> {
      await delay(120)
      return {
        status: 'ok',
        version: VERSION,
        demo: true,
        startedAt: STARTED_AT,
        cloudflared: { installed: true, version: CLOUDFLARED },
      }
    },

    async meta(): Promise<Meta> {
      await delay(140)
      return {
        version: VERSION,
        demo: true,
        dataDir: '/var/apps/cloudflare-tunnel/shares/cloudflare-tunnel/data',
        configDir: '/var/apps/cloudflare-tunnel/shares/cloudflare-tunnel/config',
        logDir: '/usr/local/apps/@appdata/cloudflare-tunnel/logs',
        listen: '0.0.0.0:18960',
        metricsPort: 20241,
        cloudflaredVersion: CLOUDFLARED,
      }
    },

    async overview(): Promise<Overview> {
      await delay(180)
      syncTunnels()
      const active = runtime.tunnelId ? routesByTunnel[runtime.tunnelId] || [] : []
      return {
        runtime: runtimeView(),
        tunnelCount: tunnels.length,
        healthyCount: tunnels.filter((t) => t.status === 'healthy').length,
        routeCount: active.length,
        zoneCount: 3,
        metrics: {
          requests: metrics.requests,
          errors: metrics.errors,
          connections: metrics.connections,
          rttMs: metrics.rttMs,
        },
        cloudflaredVersion: CLOUDFLARED,
        activity: activities.slice(0, 6).map((item) => ({ ...item })),
      }
    },

    async getSettings(): Promise<Settings> {
      await delay(140)
      return { ...settings }
    },

    async updateSettings(body): Promise<Settings> {
      await delay(420)
      if (!validateAccountId(body.accountId)) {
        throw new ApiError(400, 'validation_failed', 'Account ID 必须是 32 位十六进制字符。')
      }
      if (body.maxReconnect < 0 || body.maxReconnect > 50) {
        throw new ApiError(400, 'validation_failed', '最大重连次数必须在 0–50 之间。')
      }
      if (body.logRetentionDays < 1 || body.logRetentionDays > 365) {
        throw new ApiError(400, 'validation_failed', '日志保留天数必须在 1–365 之间。')
      }
      if (body.maxLogLines < 200 || body.maxLogLines > 10000) {
        throw new ApiError(400, 'validation_failed', '日志行数必须在 200–10000 之间。')
      }
      settings = {
        ...settings,
        accountId: body.accountId,
        protocol: body.protocol,
        edgeIpVersion: body.edgeIpVersion,
        autoConnect: body.autoConnect,
        autoReconnect: body.autoReconnect,
        maxReconnect: body.maxReconnect,
        logRetentionDays: body.logRetentionDays,
        maxLogLines: body.maxLogLines,
      }
      if (body.apiToken) {
        fakeToken = body.apiToken
        settings.apiTokenSet = true
        settings.apiTokenPreview = `••••${body.apiToken.slice(-4)}`
      }
      pushActivity('settings', '系统设置已更新', 'blue')
      return { ...settings }
    },

    async verify(body) {
      await delay(620)
      const accountId = (body.accountId ?? settings.accountId).trim()
      const token = body.apiToken || (settings.apiTokenSet ? fakeToken : '')
      if (!/^[0-9a-f]{32}$/i.test(accountId)) {
        return {
          ok: false,
          accountId,
          tokenStatus: 'unknown',
          tunnelAccess: false,
          zoneAccess: false,
          error: 'Account ID 需要是 32 位十六进制字符。',
        }
      }
      if (!token) {
        return {
          ok: false,
          accountId,
          tokenStatus: 'missing',
          tunnelAccess: false,
          zoneAccess: false,
          error: '还没有填写 API Token。',
        }
      }
      if (token.startsWith('bad')) {
        return {
          ok: false,
          accountId,
          tokenStatus: 'invalid',
          tunnelAccess: false,
          zoneAccess: false,
          error: 'Cloudflare 返回：Invalid API Token。',
        }
      }
      return {
        ok: true,
        accountId,
        tokenStatus: 'active',
        tunnelAccess: true,
        zoneAccess: true,
        error: '',
      }
    },

    async account(): Promise<AccountInfo> {
      await delay(160)
      return {
        configured: settings.accountId !== '',
        accountId: settings.accountId,
        accountName: 'Home Lab',
        zones: 3,
      }
    },

    async tunnels(): Promise<TunnelList> {
      await delay(180)
      syncTunnels()
      return { configured: settings.accountId !== '', tunnels: tunnels.map((tunnel) => ({ ...tunnel })) }
    },

    async createTunnel(name: string): Promise<Tunnel> {
      await delay(520)
      if (!/^[a-z0-9][a-z0-9-]{0,62}$/.test(name)) {
        throw new ApiError(400, 'validation_failed', '隧道名称必须匹配 ^[a-z0-9][a-z0-9-]{0,62}$。')
      }
      if (tunnels.some((tunnel) => tunnel.name === name)) {
        throw new ApiError(409, 'conflict', '已存在同名隧道。')
      }
      const tunnel: Tunnel = {
        id: crypto.randomUUID(),
        name,
        status: 'inactive',
        createdAt: iso(),
        connections: 0,
        configSrc: 'cloudflare',
        localActive: false,
      }
      tunnels.push(tunnel)
      routesByTunnel[tunnel.id] = []
      pushActivity('create', `隧道 ${name} 已创建`, 'blue')
      return { ...tunnel }
    },

    async deleteTunnel(id: string): Promise<void> {
      await delay(460)
      const index = tunnels.findIndex((tunnel) => tunnel.id === id)
      if (index < 0) throw new ApiError(404, 'not_found', '隧道不存在。')
      if (runtime.tunnelId === id && runtime.state !== 'disconnected') stopRuntime()
      const [removed] = tunnels.splice(index, 1)
      delete routesByTunnel[id]
      pushActivity('delete', `隧道 ${removed.name} 已删除`, 'danger')
    },

    async tunnelToken(id: string) {
      await delay(360)
      const tunnel = tunnels.find((item) => item.id === id)
      if (!tunnel) throw new ApiError(404, 'not_found', '隧道不存在。')
      return { token: `eyJhIjoi${randomHex(72)}` }
    },

    async routes(tunnelId: string): Promise<RoutesResponse> {
      await delay(220)
      if (!routesByTunnel[tunnelId]) throw new ApiError(404, 'not_found', '隧道不存在。')
      return routesResponse(tunnelId)
    },

    async saveRoutes(tunnelId, body): Promise<RoutesResponse> {
      await delay(620)
      if (!routesByTunnel[tunnelId]) throw new ApiError(404, 'not_found', '隧道不存在。')
      if (!body.catchAllService.trim()) {
        throw new ApiError(400, 'validation_failed', '兜底服务不能为空。')
      }
      const seen = new Set<string>()
      for (const route of body.routes) {
        if (!validateHostname(route.hostname)) {
          throw new ApiError(400, 'validation_failed', `hostname is invalid: ${route.hostname}`)
        }
        if (seen.has(route.hostname)) {
          throw new ApiError(400, 'validation_failed', `duplicate hostname: ${route.hostname}`)
        }
        seen.add(route.hostname)
        if (!validateService(route.service)) {
          throw new ApiError(400, 'validation_failed', `service is invalid: ${route.service}`)
        }
        if (route.path && !route.path.startsWith('/')) {
          throw new ApiError(400, 'validation_failed', 'path must start with /')
        }
      }
      routesByTunnel[tunnelId] = body.routes.map((route) => ({
        hostname: route.hostname.trim().toLowerCase(),
        path: route.path || '/',
        service: route.service.trim(),
      }))
      pushActivity('route', `Ingress 规则已保存（${body.routes.length} 条）`, 'blue')
      return { ...routesResponse(tunnelId), catchAllService: body.catchAllService }
    },

    async createDns(tunnelId, hostname): Promise<DnsResult> {
      await delay(540)
      const tunnel = tunnels.find((item) => item.id === tunnelId)
      if (!tunnel) throw new ApiError(404, 'not_found', '隧道不存在。')
      if (hostname.endsWith('.fail')) {
        throw new ApiError(502, 'upstream_error', 'Cloudflare 返回 403：Authentication error (code 10000)')
      }
      const existing = dnsKnown.get(hostname)
      const result: DnsResult = {
        recordId: randomHex(32),
        zoneId: randomHex(32),
        zoneName: hostname.split('.').slice(-2).join('.'),
        name: hostname,
        content: `${tunnelId}.cfargotunnel.com`,
        proxied: true,
        action: existing ? 'unchanged' : 'created',
      }
      dnsKnown.set(hostname, result)
      if (!existing) pushActivity('dns', `CNAME ${hostname} 已创建`, 'teal')
      return result
    },

    async runtime(): Promise<Runtime> {
      await delay(120)
      return runtimeView()
    },

    async connect(tunnelId: string) {
      await delay(240)
      const tunnel = tunnels.find((item) => item.id === tunnelId)
      if (!tunnel) throw new ApiError(404, 'not_found', '隧道不存在。')
      if (runtime.state === 'connected' && runtime.tunnelId === tunnelId) {
        return { status: runtimeView() }
      }
      if (runtime.tunnelId && runtime.tunnelId !== tunnelId) {
        const previous = tunnels.find((item) => item.id === runtime.tunnelId)
        if (previous) previous.status = 'inactive'
      }
      runtime = {
        ...runtime,
        state: 'connecting',
        tunnelId,
        tunnelName: tunnel.name,
        connectedAt: '',
        uptimeSec: 0,
        connections: 0,
        edgeLocation: '',
        lastError: '',
        protocol: settings.protocol === 'auto' ? 'quic' : settings.protocol,
      }
      syncTunnels()
      emitStatus()
      scheduleConnected(tunnelId)
      return { status: runtimeView() }
    },

    async disconnect() {
      await delay(200)
      if (runtime.state === 'disconnected') return { status: runtimeView() }
      runtime = { ...runtime, state: 'stopping' }
      emitStatus()
      window.setTimeout(() => {
        if (runtime.state === 'stopping') stopRuntime()
      }, 450)
      return { status: runtimeView() }
    },

    async metrics(): Promise<RuntimeMetrics> {
      await delay(100)
      return { ...metrics }
    },

    async logs({ limit = 500, afterSeq = 0, level = 'all', q = '' } = {}): Promise<LogPage> {
      await delay(200)
      const needle = q.trim().toLowerCase()
      const filtered = logs.filter((line) => {
        if (line.seq <= afterSeq) return false
        if (level !== 'all' && line.level !== level) return false
        if (!needle) return true
        return `${line.message} ${JSON.stringify(line.fields)}`.toLowerCase().includes(needle)
      })
      const page = filtered.slice(-limit)
      return {
        lines: page.map((line) => ({ ...line, fields: { ...line.fields } })),
        nextSeq: logs.length ? logs[logs.length - 1].seq : afterSeq,
        dropped,
      }
    },

    async clearLogs(): Promise<void> {
      await delay(200)
      logs = []
      dropped = 0
    },

    logsDownloadUrl(): string {
      const text = logs
        .map((line) => `${line.ts} ${line.level.toUpperCase()} ${line.message} ${JSON.stringify(line.fields)}`)
        .join('\n')
      return `data:text/plain;charset=utf-8,${encodeURIComponent(text || 'no logs')}`
    },

    async activity() {
      await delay(160)
      return { items: activities.slice(0, 100).map((item) => ({ ...item })) }
    },

    async keys(): Promise<ApiKeyList> {
      await delay(200)
      return { keys: keys.map((key) => ({ ...key })) }
    },

    async createKey(name: string): Promise<ApiKeyCreated> {
      await delay(480)
      if (!name.trim()) throw new ApiError(400, 'validation_failed', '名称不能为空。')
      const token = `ct_${randomHex(64)}`
      const key: ApiKey = {
        id: crypto.randomUUID(),
        name: name.trim(),
        prefix: token.slice(0, 7),
        createdAt: iso(),
        lastUsedAt: '',
      }
      keys = [key, ...keys]
      pushActivity('key', `API 凭证 ${key.name} 已创建`, 'blue')
      return { key: { ...key }, token }
    },

    async deleteKey(id: string): Promise<void> {
      await delay(360)
      const key = keys.find((item) => item.id === id)
      if (!key) throw new ApiError(404, 'not_found', '凭证不存在。')
      keys = keys.filter((item) => item.id !== id)
      pushActivity('key', `API 凭证 ${key.name} 已吊销`, 'danger')
    },

    openEvents(handlers: StreamHandlers): EventStreamHandle {
      startIntervals()

      const onStatus = (rt: Runtime) => handlers.onStatus?.(rt)
      const onLog = (line: LogLine) => handlers.onLog?.(line)
      const onMetrics = (m: RuntimeMetrics) => handlers.onMetrics?.(m)
      const onActivity = (a: Activity) => handlers.onActivity?.(a)

      statusListeners.add(onStatus)
      logListeners.add(onLog)
      metricListeners.add(onMetrics)
      activityListeners.add(onActivity)

      window.setTimeout(() => {
        handlers.onOpen?.()
        handlers.onHello?.(runtimeView(), logs.slice(-50).map((line) => ({ ...line })))
      }, 80)

      // Fill the overview sparkline history quickly in demo mode.
      const warmup = [26, 17, 12, 8, 5, 3, 1].map((step) =>
        window.setTimeout(() => {
          onMetrics({ ...metrics, rttMs: 30 + step, sampledAt: iso() })
        }, step * 130),
      )

      return {
        close: () => {
          statusListeners.delete(onStatus)
          logListeners.delete(onLog)
          metricListeners.delete(onMetrics)
          activityListeners.delete(onActivity)
          for (const timer of warmup) window.clearTimeout(timer)
        },
      }
    },
  }
}
