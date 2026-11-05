import type { Route } from '../types'

export interface CachedRoutes {
  routes: Route[]
  catchAllService: string
  rawIngressCount: number
}

const cache = new Map<string, CachedRoutes>()

export function routeCacheKey(accountId: string | undefined, tunnelId: string) {
  return `${accountId || ''}:${tunnelId}`
}

export function cloneRoutes(items: Route[]): Route[] {
  return items.map((item) => ({ hostname: item.hostname, path: item.path || '/', service: item.service }))
}

export function getCachedRoutes(key: string): CachedRoutes | undefined {
  const hit = cache.get(key)
  if (!hit) return undefined
  return { routes: cloneRoutes(hit.routes), catchAllService: hit.catchAllService, rawIngressCount: hit.rawIngressCount }
}

export function setCachedRoutes(key: string, data: CachedRoutes) {
  cache.set(key, {
    routes: cloneRoutes(data.routes),
    catchAllService: data.catchAllService,
    rawIngressCount: data.rawIngressCount,
  })
}
