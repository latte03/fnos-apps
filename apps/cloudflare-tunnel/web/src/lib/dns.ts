import type { DnsAction } from '../types'

/**
 * DNS feedback is client-side state: the contract exposes the write call but no
 * read endpoint, so the pill starts as "未检查" until we create/refresh a CNAME.
 */
export type DnsUiState = 'unknown' | 'pending' | DnsAction | 'failed'

export interface DnsUiMeta {
  label: string
  pill: 'healthy' | 'warning' | 'danger' | 'neutral' | 'info'
}

export function dnsMeta(state: DnsUiState): DnsUiMeta {
  switch (state) {
    case 'created':
      return { label: '已创建', pill: 'healthy' }
    case 'updated':
      return { label: '已更新', pill: 'info' }
    case 'unchanged':
      return { label: '已存在', pill: 'info' }
    case 'pending':
      return { label: '创建中', pill: 'warning' }
    case 'failed':
      return { label: '失败', pill: 'danger' }
    default:
      return { label: '未检查', pill: 'neutral' }
  }
}
