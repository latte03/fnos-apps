/** Presentation helpers shared by views and components. */

const pad = (n: number) => String(n).padStart(2, '0')

export function formatClock(value: string | number | Date): string {
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return '--:--:--'
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export interface UptimeParts {
  days: number
  clock: string
  idle: boolean
}

export function uptimeParts(seconds: number | null | undefined): UptimeParts {
  if (seconds === null || seconds === undefined || !Number.isFinite(seconds) || seconds < 0) {
    return { days: 0, clock: '--:--:--', idle: true }
  }
  const total = Math.floor(seconds)
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  return { days, clock: `${pad(hours)}:${pad(minutes)}:${pad(secs)}`, idle: false }
}

export function uptimeSeconds(runtime: { connectedAt: string; uptimeSec: number } | null, now = Date.now()): number | null {
  if (!runtime) return null
  const started = Date.parse(runtime.connectedAt)
  if (!Number.isNaN(started)) {
    const diff = Math.floor((now - started) / 1000)
    return Math.max(runtime.uptimeSec || 0, diff, 0)
  }
  return runtime.uptimeSec
}

export function formatRelative(value: string | number | Date, now = Date.now()): string {
  const t = value instanceof Date ? value.getTime() : new Date(value).getTime()
  if (Number.isNaN(t)) return ''
  const diff = Math.max(0, Math.round((now - t) / 1000))
  if (diff < 45) return '刚刚'
  if (diff < 3600) return `${Math.round(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.round(diff / 3600)} 小时前`
  if (diff < 86400 * 30) return `${Math.round(diff / 86400)} 天前`
  return formatDate(value)
}

export function formatDate(value: string | number | Date): string {
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function formatDateTime(value: string | number | Date): string {
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return `${formatDate(d)} ${formatClock(d)}`
}

export function shortId(id: string): string {
  if (!id) return '—'
  if (id.length <= 14) return id
  return `${id.slice(0, 8)}…${id.slice(-4)}`
}

export function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—'
  return value.toLocaleString('en-US')
}

export function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined || !Number.isFinite(seconds)) return '—'
  const total = Math.max(0, Math.floor(seconds))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分`
  if (minutes > 0) return `${minutes} 分 ${total % 60} 秒`
  return `${total} 秒`
}

export function initials(name: string): string {
  const trimmed = (name || '').trim()
  if (!trimmed) return 'CF'
  const words = trimmed.split(/[\s-]+/).filter(Boolean)
  if (words.length >= 2) return (words[0][0] + words[1][0]).toUpperCase()
  return trimmed.slice(0, 2).toUpperCase()
}

export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

export function truncateMiddle(value: string, head = 14, tail = 6): string {
  if (value.length <= head + tail + 1) return value
  return `${value.slice(0, head)}…${value.slice(-tail)}`
}

export function plural(n: number, unit: string): string {
  return `${n} ${unit}`
}
