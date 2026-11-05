<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import PageHead from '../components/PageHead.vue'
import StatusPill from '../components/StatusPill.vue'
import TrafficChart from '../components/TrafficChart.vue'
import SpineSvg from '../components/SpineSvg.vue'
import SkeletonBlock from '../components/SkeletonBlock.vue'
import EmptyState from '../components/EmptyState.vue'
import NoticeBar from '../components/NoticeBar.vue'
import { api } from '../lib/client'
import { apiErrorMessage } from '../lib/api'
import { getCachedRoutes, routeCacheKey, setCachedRoutes } from '../lib/route-cache'
import { formatNumber, formatRelative, shortId, uptimeParts, uptimeSeconds } from '../lib/format'
import { toast } from '../lib/toast'
import { activityIcon, app, connectTunnel, disconnectTunnel, noteError, refreshActivity, refreshOverview, runtimeStatusMeta, setRouteCount } from '../stores/state'
import type { Route } from '../types'

const router = useRouter()

const now = ref(Date.now())
const routes = ref<Route[]>([])
const routesFor = ref('')
const routesError = ref('')
const routesLoading = ref(false)
const loading = ref(!app.overview)
const pageError = ref('')
let routesRequest = 0

const rt = computed(() => app.runtime)
const metrics = computed(() => app.metrics)
const state = computed(() => rt.value?.state ?? 'disconnected')
const connected = computed(() => state.value === 'connected')
const connecting = computed(
  () => state.value === 'connecting' || state.value === 'reconnecting' || state.value === 'stopping',
)
const errored = computed(() => state.value === 'error')
const meta = computed(() => runtimeStatusMeta(rt.value))
const setupMode = computed(() => app.needsSetup || (!!app.overview && app.overview.tunnelCount === 0 && !app.booting))

const uptime = computed(() => uptimeParts(connected.value ? uptimeSeconds(rt.value, now.value) : null))

const latencyText = computed(() => {
  const rtt = metrics.value?.rttMs
  return rtt === null || rtt === undefined ? null : rtt
})

const heroMicro = computed(() => {
  if (connected.value) {
    return [rt.value?.protocol?.toUpperCase(), rt.value?.edgeLocation, latencyText.value !== null ? `${latencyText.value} ms` : '']
      .filter(Boolean)
      .join(' · ')
  }
  if (connecting.value) return '正在握手'
  return '—'
})

const heroDesc = computed(() => {
  if (!rt.value?.tunnelId) return '还没有连接任何隧道。选择一条隧道开始，把 NAS 上的服务带到公网。'
  if (connected.value) {
    const connections = rt.value.connections || 0
    const edge = rt.value.edgeLocation ? `，流量正在经过 ${rt.value.edgeLocation} 节点` : ''
    return `Cloudflare Edge 已建立 ${connections} 条连接${edge}。`
  }
  if (connecting.value) return '正在与 Cloudflare Edge 建立连接，通常需要 1–3 秒。'
  if (errored.value) return `上次错误：${rt.value.lastError || '连接中断'}。检查日志或重新连接。`
  return '当前隧道未连接。连接后，所有路由规则才会开始转发。'
})

const routeCount = computed(() => (routesFor.value && routesFor.value === rt.value?.tunnelId ? routes.value.length : app.overview?.routeCount ?? 0))

const successRate = computed(() => {
  const requests = metrics.value?.requests
  const errors = metrics.value?.errors
  if (requests === null || requests === undefined || requests <= 0) return null
  return Math.max(0, Math.min(100, ((requests - (errors ?? 0)) / requests) * 100))
})

const bars = computed(() => {
  const samples = app.metricSamples
  const rates: number[] = []
  for (let index = 1; index < samples.length; index += 1) {
    const previous = samples[index - 1]
    const current = samples[index]
    if (previous.requests === null || current.requests === null) continue
    const requests = current.requests - previous.requests
    if (requests <= 0) continue
    const errors =
      previous.errors !== null && current.errors !== null ? Math.max(0, current.errors - previous.errors) : 0
    rates.push(Math.max(0, Math.min(100, ((requests - errors) / requests) * 100)))
  }
  if (!rates.length) return []
  const slots = Array.from({ length: 14 }, () => 0)
  const recent = rates.slice(-14)
  const offset = 14 - recent.length
  recent.forEach((rate, index) => {
    slots[offset + index] = Math.max(8, Math.round(rate))
  })
  return slots
})

const sampleWindow = computed(() => {
  const samples = app.metricSamples
  if (samples.length < 2) return 0
  const first = Date.parse(samples[0].sampledAt)
  const last = Date.parse(samples[samples.length - 1].sampledAt)
  if (Number.isNaN(first) || Number.isNaN(last)) return 0
  return Math.max(1, Math.round((last - first) / 1000))
})

const activityItems = computed(() =>
  app.activity.slice(0, 4).map((item) => ({
    ...item,
    icon: activityIcon(item.kind),
    time: formatRelative(item.ts, now.value),
  })),
)

const nasAddress = computed(() => window.location.hostname || '本机')
const hostName = computed(() => window.location.hostname || '本机')

function applyRoutes(tunnelId: string, list: Route[]) {
  routes.value = list
  routesFor.value = tunnelId
  routesError.value = ''
  if (connected.value) setRouteCount(list.length)
}

async function loadRoutes(force = false) {
  const tunnelId = rt.value?.tunnelId
  if (!tunnelId) {
    routesRequest += 1
    routesLoading.value = false
    routes.value = []
    routesFor.value = ''
    routesError.value = ''
    return
  }
  const key = routeCacheKey(app.settings?.accountId, tunnelId)
  if (!force) {
    const cached = getCachedRoutes(key)
    if (cached) {
      routesRequest += 1
      routesLoading.value = false
      applyRoutes(tunnelId, cached.routes)
      return
    }
  }
  const requestId = ++routesRequest
  routesLoading.value = true
  routesError.value = ''
  try {
    const result = await api.routes(tunnelId)
    if (requestId !== routesRequest || rt.value?.tunnelId !== tunnelId) return
    setCachedRoutes(key, {
      routes: result.routes,
      catchAllService: result.catchAllService,
      rawIngressCount: result.rawIngressCount,
    })
    applyRoutes(tunnelId, result.routes)
  } catch (err) {
    if (requestId !== routesRequest) return
    routesError.value = apiErrorMessage(err)
  } finally {
    if (requestId === routesRequest) routesLoading.value = false
  }
}

async function load(forceRoutes = false) {
  loading.value = !app.overview
  const routesTask = loadRoutes(forceRoutes)
  void refreshActivity()
  pageError.value = (await refreshOverview()) || ''
  await routesTask
  loading.value = false
}

function onRefresh() {
  void load(true)
}

async function refreshRoutes() {
  if (!rt.value?.tunnelId) return
  await loadRoutes(true)
  if (!routesError.value) toast('路由数据已刷新', 'success')
}

async function onConnect() {
  const tunnelId = rt.value?.tunnelId
  if (tunnelId) {
    try {
      await connectTunnel(tunnelId, rt.value?.tunnelName || '')
    } catch (err) {
      toast(noteError(err), 'error')
    }
  } else {
    void router.push('/tunnels')
  }
}

async function onDisconnect() {
  try {
    await disconnectTunnel()
  } catch (err) {
    toast(noteError(err), 'error')
  }
}

function addRoute() {
  if (!rt.value?.tunnelId) {
    toast('请先连接一个隧道，再添加路由', 'error')
    void router.push('/tunnels')
    return
  }
  void router.push({ name: 'routes', query: { add: '1' } })
}

function openRoutes() {
  void router.push({ name: 'routes' })
}

function editRoute(route: Route) {
  void router.push({ name: 'routes', query: { host: route.hostname } })
}

let ticker = 0

onMounted(() => {
  void load()
  ticker = window.setInterval(() => {
    now.value = Date.now()
  }, 1000)
  window.addEventListener('ct:refresh', onRefresh)
})

onBeforeUnmount(() => {
  window.clearInterval(ticker)
  window.removeEventListener('ct:refresh', onRefresh)
})

watch(
  () => rt.value?.tunnelId,
  () => {
    void loadRoutes()
  },
)

watch(
  () => state.value,
  (value) => {
    if (value === 'connected') void refreshOverview()
  },
)
</script>

<template>
  <PageHead
    title="概览"
    sub="一条隧道，把 NAS 上的服务安全地带到公网。状态、路由与日志都在一个地方。"
  >
    <template #actions>
      <button class="btn ghost" type="button" @click="onRefresh">
        <Icon name="refresh" />
        刷新
      </button>
      <button class="btn" :class="connected ? 'primary' : 'ghost'" type="button" @click="addRoute">
        <Icon name="plus" />
        添加路由
      </button>
    </template>
  </PageHead>

  <NoticeBar v-if="pageError" tone="warning" icon="alert" title="部分数据刷新失败">
    {{ pageError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="onRefresh">重试</button>
    </template>
  </NoticeBar>

  <template v-if="setupMode">
    <section class="card setup-card reveal" style="--i: 1">
      <div class="card-head">
        <div>
          <h3>{{ app.needsSetup ? '先完成 Cloudflare 配置' : '还没有隧道' }}</h3>
          <p>把账户凭据交给应用，创建第一条隧道，再把域名映射到内网服务。</p>
        </div>
      </div>
      <ol class="setup-steps">
        <li>
          <span class="setup-index">1</span>
          <div>
            <strong>填写 Account ID 与 API Token</strong>
            <small>Token 需要 Account · Cloudflare Tunnel · Edit 与 Zone · DNS · Edit。</small>
          </div>
          <RouterLink class="btn small ghost" to="/settings">去设置</RouterLink>
        </li>
        <li>
          <span class="setup-index">2</span>
          <div>
            <strong>创建或选择一条隧道</strong>
            <small>同一时间只运行一条；切换时先断开当前隧道。</small>
          </div>
          <RouterLink class="btn small ghost" to="/tunnels">去隧道</RouterLink>
        </li>
        <li>
          <span class="setup-index">3</span>
          <div>
            <strong>添加域名路由</strong>
            <small>保存后自动创建 CNAME，公网域名即可指向 NAS。</small>
          </div>
          <RouterLink class="btn small ghost" to="/routes">去路由</RouterLink>
        </li>
      </ol>
    </section>
  </template>

  <template v-else>
    <div v-if="loading && !app.overview" class="overview-grid">
      <div class="card hero"><SkeletonBlock height="26px" width="42%" /><SkeletonBlock height="70px" width="66%" /><SkeletonBlock height="16px" width="52%" /><SkeletonBlock height="40px" width="30%" /></div>
      <div class="card"><SkeletonBlock height="18px" width="50%" /><SkeletonBlock height="150px" /></div>
    </div>

    <template v-else>
      <div class="overview-grid">
        <section class="card hero reveal" style="--i: 1">
          <div class="hero-top">
            <div class="hero-status">
              <StatusPill
                :label="meta.label"
                :kind="meta.pill"
                :dot="connected || errored"
                :spinner="connecting"
              />
              <span class="micro">{{ heroMicro }}</span>
            </div>
            <button class="icon-btn" type="button" aria-label="管理隧道" title="管理隧道" @click="$router.push('/tunnels')">
              <Icon name="more" />
            </button>
          </div>
          <div class="hero-main">
            <div class="hero-copy">
              <p class="eyebrow">当前隧道</p>
              <h2>
                {{ rt?.tunnelName || '未选择隧道' }}
                <span v-if="rt?.tunnelId" class="mono">{{ shortId(rt.tunnelId) }}</span>
              </h2>
              <p class="hero-desc">{{ heroDesc }}</p>
              <span class="uptime-label">运行时长</span>
              <span class="uptime" :class="{ 'is-idle': uptime.idle }">
                <template v-if="!uptime.idle">{{ uptime.days }}<span>天</span> {{ uptime.clock }}</template>
                <template v-else>—</template>
              </span>
              <div class="hero-actions">
                <button v-if="connected" class="btn ghost danger-text" type="button" @click="onDisconnect">
                  断开连接
                </button>
                <button v-else-if="connecting" class="btn primary" type="button" disabled>
                  <span class="spinner" aria-hidden="true"></span>
                  连接中
                </button>
                <button v-else-if="errored" class="btn primary" type="button" @click="onConnect">
                  <Icon name="refresh" />
                  重新连接
                </button>
                <button v-else class="btn primary" type="button" @click="onConnect">
                  <Icon name="plug" />
                  选择隧道
                </button>
                <button v-if="rt?.tunnelId" class="btn subtle" type="button" @click="openRoutes">
                  <Icon name="branch" />
                  路由 {{ routeCount }}
                </button>
              </div>
            </div>
            <div class="spine-wrap">
              <SpineSvg
                :live="connected"
                :protocol="rt?.protocol || 'quic'"
                :edge-location="rt?.edgeLocation || ''"
                :connections="rt?.connections || 0"
                :latency-ms="latencyText"
                :host-name="hostName"
                :host-address="nasAddress"
              />
            </div>
          </div>
        </section>

        <aside class="card quick reveal" style="--i: 2">
          <div class="card-head">
            <div>
              <h3>快捷操作</h3>
              <p>常用动作，少点两下</p>
            </div>
          </div>
          <button class="quick-item" type="button" @click="$router.push('/tunnels')">
            <span class="q-icon"><Icon name="waypoints" /></span>
            <span class="q-copy"><strong>切换隧道</strong><small>{{ app.tunnels.length }} 条隧道可用</small></span>
            <span class="q-chevron"><Icon name="chevron-right" /></span>
          </button>
          <button class="quick-item" type="button" @click="addRoute">
            <span class="q-icon"><Icon name="branch" /></span>
            <span class="q-copy"><strong>新增路由</strong><small>域名 → 内网服务</small></span>
            <span class="q-chevron"><Icon name="chevron-right" /></span>
          </button>
          <div class="quick-health">
            <div class="qh-top">
              <span>请求成功率</span>
              <strong>{{ successRate === null ? '—' : successRate.toFixed(2) }}<small v-if="successRate !== null">%</small></strong>
            </div>
            <div v-if="bars.length" class="qh-bars" aria-hidden="true">
              <span v-for="(height, index) in bars" :key="index" :style="{ '--h': `${height}%` }"></span>
            </div>
            <div v-else class="qh-empty">等待 metrics 样本…</div>
            <div class="qh-foot">
              <span>本次会话</span>
              <span>{{ app.metricSamples.length }} 个样本</span>
            </div>
          </div>
          <div class="mini-note">
            <span class="pulse-dot"></span>
            {{ app.health?.cloudflared?.installed === false ? 'cloudflared 未安装' : `cloudflared ${app.health?.cloudflared?.version || ''}` }}
            <span class="mono">{{ latencyText !== null ? `${latencyText} ms` : '—' }}</span>
          </div>
        </aside>
      </div>

      <div class="stats-grid">
        <section class="card traffic-card reveal" style="--i: 3">
          <div class="traffic-head">
            <div>
              <h3>请求趋势</h3>
              <p>每 5 秒采样{{ sampleWindow ? ` · 最近 ${sampleWindow} 秒` : '' }}</p>
            </div>
            <div class="traffic-summary">
              <strong>{{ formatNumber(metrics?.requests) }}</strong>
              <small>累计请求</small>
            </div>
          </div>
          <TrafficChart :samples="app.metricSamples" />
        </section>

        <aside class="card stat-card reveal" style="--i: 4">
          <div class="card-head">
            <div>
              <h3>运行数据</h3>
              <p>来自 cloudflared metrics</p>
            </div>
          </div>
          <div class="stat-list">
            <div class="stat-row">
              <span class="stat-label">
                活跃路由
                <small>{{ rt?.tunnelId ? '当前隧道' : '无活动隧道' }}</small>
              </span>
              <strong class="stat-value">{{ formatNumber(routeCount) }}</strong>
            </div>
            <div class="stat-row">
              <span class="stat-label">
                平均延迟
                <small>{{ latencyText !== null ? 'Edge RTT' : '无连接' }}</small>
              </span>
              <strong class="stat-value" :class="{ 'is-null': latencyText === null }">
                {{ latencyText !== null ? latencyText : '—' }}<small v-if="latencyText !== null">ms</small>
              </strong>
            </div>
            <div class="stat-row">
              <span class="stat-label">
                错误请求
                <small>{{ metrics?.connections ? `${metrics.connections} 条连接` : '无活动连接' }}</small>
              </span>
              <strong class="stat-value" :class="{ 'is-null': metrics?.errors === null }">
                {{ formatNumber(metrics?.errors) }}
              </strong>
            </div>
          </div>
        </aside>
      </div>

      <div class="lower-grid">
        <section class="card reveal" style="--i: 7">
          <div class="card-head">
            <div>
              <h3>路由</h3>
              <p>{{ routeCount }} 条规则 · 首条匹配生效</p>
            </div>
            <div class="card-head-tools">
              <button
                class="icon-btn"
                :class="{ 'is-spinning': routesLoading }"
                type="button"
                aria-label="刷新路由"
                title="刷新路由"
                :disabled="routesLoading || !rt?.tunnelId"
                @click="refreshRoutes"
              >
                <Icon name="refresh" />
              </button>
              <button class="link" type="button" @click="openRoutes">全部路由 →</button>
            </div>
          </div>
          <div
            v-if="routesLoading && routesFor !== rt?.tunnelId"
            class="route-table-skeleton"
            role="status"
            aria-label="正在加载路由"
          >
            <div v-for="index in 3" :key="index" class="route-skeleton-row" aria-hidden="true">
              <SkeletonBlock height="14px" width="72%" />
              <SkeletonBlock height="13px" width="84%" />
              <SkeletonBlock height="12px" width="28px" />
              <SkeletonBlock height="22px" width="22px" />
            </div>
          </div>
          <div v-else-if="routesError" class="table-empty">{{ routesError }}</div>
          <div v-else-if="!routes.length" class="table-empty">
            当前隧道还没有路由。添加一条域名 → 内网服务的映射。
          </div>
          <div v-else class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>域名</th>
                  <th>内网服务</th>
                  <th>路径</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="route in routes.slice(0, 3)" :key="route.hostname">
                  <td><span class="host">{{ route.hostname }}</span></td>
                  <td><span class="service">{{ route.service }}</span></td>
                  <td><span class="micro">{{ route.path || '/' }}</span></td>
                  <td class="cell-actions">
                    <button class="icon-btn" type="button" aria-label="编辑路由" @click="editRoute(route)">
                      <Icon name="arrow-up-right" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="card activity reveal" style="--i: 8">
          <div class="card-head">
            <div>
              <h3>最近活动</h3>
              <p>本机事件，非云端审计</p>
            </div>
          </div>
          <EmptyState v-if="!activityItems.length" icon="activity" title="暂无活动" message="连接隧道或保存路由后，这里会记录本机事件。" />
          <ol v-else class="timeline">
            <li v-for="item in activityItems" :key="item.id">
              <span class="t-icon" :class="item.tone === 'neutral' ? '' : item.tone"><Icon :name="item.icon" /></span>
              <span class="t-copy">
                <strong>{{ item.title }}</strong>
                <span>{{ item.time }}</span>
              </span>
            </li>
          </ol>
        </section>
      </div>
    </template>
  </template>
</template>
