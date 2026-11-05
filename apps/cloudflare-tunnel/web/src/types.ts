/**
 * DTOs mirrored from CONTRACT.md §6. Keep names and optionality identical to
 * the wire format; never add fields the server does not send.
 */

export type RuntimeState =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'error'
  | 'stopping'

export type LogLevel = 'debug' | 'info' | 'warn' | 'error'
export type LogLevelFilter = 'all' | LogLevel
export type Protocol = 'auto' | 'quic' | 'http2'
export type EdgeIpVersion = 'auto' | '4' | '6'
export type Tone = 'green' | 'blue' | 'teal' | 'neutral' | 'danger'
export type DnsAction = 'created' | 'updated' | 'unchanged'

export interface Health {
  status: string
  version: string
  demo: boolean
  startedAt: string
  cloudflared: { installed: boolean; version: string }
}

export interface Meta {
  version: string
  demo: boolean
  dataDir: string
  configDir: string
  logDir: string
  listen: string
  metricsPort: number
  cloudflaredVersion: string
}

export interface Settings {
  accountId: string
  apiTokenSet: boolean
  apiTokenPreview: string
  protocol: Protocol
  edgeIpVersion: EdgeIpVersion
  autoConnect: boolean
  autoReconnect: boolean
  maxReconnect: number
  logRetentionDays: number
  maxLogLines: number
}

export interface SettingsUpdate {
  accountId: string
  protocol: Protocol
  edgeIpVersion: EdgeIpVersion
  autoConnect: boolean
  autoReconnect: boolean
  maxReconnect: number
  logRetentionDays: number
  maxLogLines: number
  /** Only present when the user typed a new token. */
  apiToken?: string
}

export interface VerifyRequest {
  accountId?: string
  apiToken?: string
}

export interface VerifyResult {
  ok: boolean
  accountId: string
  tokenStatus: string
  tunnelAccess: boolean
  zoneAccess: boolean
  error: string
}

export interface AccountInfo {
  configured: boolean
  accountId: string
  accountName: string
  zones: number
}

export interface Tunnel {
  id: string
  name: string
  status: string
  createdAt: string
  connections: number
  configSrc: string
  localActive: boolean
}

export interface TunnelList {
  configured: boolean
  tunnels: Tunnel[]
}

export interface TunnelToken {
  token: string
}

export interface Route {
  hostname: string
  path: string
  service: string
}

export interface RoutesResponse {
  routes: Route[]
  catchAllService: string
  rawIngressCount: number
}

export interface RoutesUpdate {
  routes: Route[]
  catchAllService: string
}

export interface TunnelRoute extends Route {
  /** Optional advanced fields exposed by the drawer; mapped to originRequest. */
  noTLSVerify?: boolean
  httpHostHeader?: string
}

export interface DnsResult {
  recordId: string
  zoneId: string
  zoneName: string
  name: string
  content: string
  proxied: boolean
  action: DnsAction
}

export interface Runtime {
  state: RuntimeState
  tunnelId: string
  tunnelName: string
  connectedAt: string
  uptimeSec: number
  protocol: string
  edgeLocation: string
  connections: number
  reconnectAttempt: number
  lastError: string
  cloudflaredVersion: string
}

export interface RuntimeConnectResult {
  status: Runtime
}

export interface RuntimeMetrics {
  requests: number | null
  errors: number | null
  connections: number | null
  rttMs: number | null
  sampledAt: string
}

export interface LogLine {
  seq: number
  ts: string
  level: LogLevel
  message: string
  fields: Record<string, unknown>
}

export interface LogPage {
  lines: LogLine[]
  nextSeq: number
  dropped: number
}

export interface Activity {
  id: string
  ts: string
  kind: string
  title: string
  tone: Tone
}

export interface ActivityList {
  items: Activity[]
}

export interface ApiKey {
  id: string
  name: string
  prefix: string
  createdAt: string
  lastUsedAt: string
}

export interface ApiKeyList {
  keys: ApiKey[]
}

export interface ApiKeyCreated {
  key: ApiKey
  token: string
}

export interface Overview {
  runtime: Runtime
  tunnelCount: number
  healthyCount: number
  routeCount: number
  zoneCount: number
  metrics: {
    requests: number | null
    errors: number | null
    connections: number | null
    rttMs: number | null
  }
  cloudflaredVersion: string
  activity: Activity[]
}

export interface ApiErrorEnvelope {
  error: {
    code: string
    message: string
    details?: unknown
  }
}

/** Events delivered on `GET /api/v1/events`. */
export type StreamEvent =
  | { type: 'hello'; status: Runtime; recent: LogLine[] }
  | { type: 'status'; status: Runtime }
  | { type: 'log'; line: LogLine }
  | { type: 'activity'; activity: Activity }
  | { type: 'metrics'; metrics: RuntimeMetrics }
