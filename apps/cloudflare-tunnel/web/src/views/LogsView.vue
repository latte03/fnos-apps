<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import Icon from '../components/Icon.vue'
import PageHead from '../components/PageHead.vue'
import LogLine from '../components/LogLine.vue'
import SegmentedControl from '../components/SegmentedControl.vue'
import SwitchControl from '../components/SwitchControl.vue'
import StatusPill from '../components/StatusPill.vue'
import NoticeBar from '../components/NoticeBar.vue'
import SkeletonBlock from '../components/SkeletonBlock.vue'
import { api } from '../lib/client'
import { formatDate, formatDuration } from '../lib/format'
import { toast } from '../lib/toast'
import { app, clearLogs, loadLogs, runtimeStatusMeta } from '../stores/state'
import type { LogLevelFilter } from '../types'

const level = ref<LogLevelFilter>('all')
const query = ref('')
const paused = ref(false)
const frozenSeq = ref(0)
const autoScroll = ref(true)
const expanded = reactive(new Set<number>())
const logBody = ref<HTMLDivElement | null>(null)
const loading = ref(!app.logsLoaded)
const loadError = ref('')
const clearing = ref(false)

const levels = [
  { value: 'all', label: '全部' },
  { value: 'info', label: 'INF' },
  { value: 'warn', label: 'WRN' },
  { value: 'error', label: 'ERR' },
  { value: 'debug', label: 'DBG' },
]

const matches = computed(() => {
  const needle = query.value.trim().toLowerCase()
  return app.logs.filter((line) => {
    if (paused.value && line.seq > frozenSeq.value) return false
    if (level.value !== 'all' && line.level !== level.value) return false
    if (!needle) return true
    return `${line.message} ${JSON.stringify(line.fields)}`.toLowerCase().includes(needle)
  })
})

const pendingCount = computed(() => (paused.value ? app.logs.filter((line) => line.seq > frozenSeq.value).length : 0))

const stateMeta = computed(() => runtimeStatusMeta(app.runtime))

const liveKind = computed(() => {
  if (paused.value) return { label: '已暂停', kind: 'warning' as const }
  if (app.sse === 'closed') return { label: '重连中', kind: 'warning' as const }
  return { label: '实时流', kind: 'healthy' as const }
})

const logCount = computed(() => `${matches.value.length} / ${app.logs.length} 行`)

const settings = computed(() => app.settings)
const runtime = computed(() => app.runtime)

let lastAutoScrollAt = 0
let programmatic = false

function onBodyScroll() {
  const body = logBody.value
  if (!body) return
  if (Date.now() - lastAutoScrollAt < 250) return
  const distance = body.scrollHeight - body.scrollTop - body.clientHeight
  if (distance > 28 && autoScroll.value && !programmatic) autoScroll.value = false
}

function scrollToBottom(force = false) {
  const body = logBody.value
  if (!body) return
  if (!autoScroll.value && !force) return
  programmatic = true
  lastAutoScrollAt = Date.now()
  body.scrollTop = body.scrollHeight
  window.requestAnimationFrame(() => {
    programmatic = false
  })
}

function toggleLine(seq: number) {
  if (expanded.has(seq)) expanded.delete(seq)
  else {
    expanded.add(seq)
    autoScroll.value = false
  }
}

function setLevel(value: string) {
  level.value = value as LogLevelFilter
}

function togglePause() {
  paused.value = !paused.value
  if (paused.value) {
    frozenSeq.value = app.logs.length ? app.logs[app.logs.length - 1].seq : 0
  } else {
    frozenSeq.value = 0
    void nextTick(() => scrollToBottom(true))
  }
}

async function doClear() {
  if (clearing.value) return
  clearing.value = true
  const error = await clearLogs()
  clearing.value = false
  if (error) toast(error, 'error')
  else {
    expanded.clear()
    toast('日志已清空', 'success')
  }
}

function download() {
  if (!app.logs.length) {
    toast('暂无可下载的日志', 'error')
    return
  }
  const anchor = document.createElement('a')
  anchor.href = api.logsDownloadUrl()
  anchor.download = `cloudflared-${formatDate(new Date())}.log`
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  toast('日志已下载', 'success')
}

async function load() {
  loading.value = !app.logsLoaded
  loadError.value = (await loadLogs()) || ''
  loading.value = false
  await nextTick()
  scrollToBottom(true)
}

function onRefresh() {
  void load()
}

watch(
  () => matches.value.length,
  async () => {
    await nextTick()
    if (autoScroll.value && !paused.value) scrollToBottom()
  },
)

watch(
  () => app.sse,
  (value, previous) => {
    if (value === 'open' && previous === 'closed') {
      toast('实时连接已恢复', 'success')
      void nextTick(() => scrollToBottom())
    }
  },
)

watch(level, async () => {
  await nextTick()
  if (autoScroll.value) scrollToBottom(true)
})

onMounted(() => {
  void load()
  window.addEventListener('ct:refresh', onRefresh)
})

onBeforeUnmount(() => {
  window.removeEventListener('ct:refresh', onRefresh)
})

function onToggleAuto(value: boolean) {
  autoScroll.value = value
  if (value) scrollToBottom(true)
}
</script>

<template>
  <PageHead eyebrow="诊断" title="运行日志" sub="来自 cloudflared 的实时输出。日志只在本地保留，不上传云端。">
    <template #actions>
      <button class="btn ghost" type="button" @click="download">
        <Icon name="download" />
        下载
      </button>
      <button class="btn danger" type="button" :disabled="clearing" @click="doClear">
        <span v-if="clearing" class="spinner" aria-hidden="true"></span>
        <Icon v-else name="trash" />
        清空
      </button>
    </template>
  </PageHead>

  <NoticeBar v-if="loadError" tone="danger" icon="alert" title="无法读取日志">
    {{ loadError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="onRefresh">重试</button>
    </template>
  </NoticeBar>
  <NoticeBar v-if="app.logDropped > 0" tone="warning" icon="alert" title="日志被截断">
    内存保留上限已触发，已丢弃 {{ app.logDropped }} 行较早的日志。
  </NoticeBar>
  <NoticeBar v-if="app.sse === 'closed'" tone="warning" icon="alert" title="实时流断开">
    正在自动重连（第 {{ app.sseAttempt }} 次），期间不会收到新日志。
  </NoticeBar>
  <NoticeBar v-if="pendingCount" tone="info" icon="pause" title="日志已暂停">
    暂停期间收到 {{ pendingCount }} 条新日志，继续后会补齐显示。
    <template #action>
      <button class="btn small ghost" type="button" @click="togglePause">继续</button>
    </template>
  </NoticeBar>

  <div class="toolbar reveal" style="--i: 1">
    <StatusPill :label="liveKind.label" :kind="liveKind.kind" dot />
    <button class="btn ghost small" type="button" @click="togglePause">
      <Icon :name="paused ? 'play' : 'pause'" />
      {{ paused ? '继续' : '暂停' }}
    </button>
    <SegmentedControl :model-value="level" :options="levels" aria-label="日志级别" @update:model-value="setLevel" />
    <div class="search">
      <Icon name="search" />
      <input v-model="query" type="search" placeholder="搜索日志、域名或连接..." autocomplete="off" aria-label="搜索日志" />
    </div>
    <div class="spacer"></div>
    <div class="switch-row auto-scroll-row">
      自动滚动
      <SwitchControl :model-value="autoScroll" label="自动滚动" @update:model-value="onToggleAuto" />
    </div>
  </div>

  <div class="logs-layout">
    <section class="card log-console reveal" style="--i: 2">
      <div class="log-head">
        <div class="log-head-left">
          <span class="live-dot" :class="{ 'is-paused': paused || app.sse === 'closed' }"></span>
          cloudflared · stdout<span class="log-count">{{ logCount }}</span>
        </div>
        <div class="log-head-right"><span>tail -f</span></div>
      </div>
      <div ref="logBody" class="log-body" @scroll.passive="onBodyScroll">
        <div v-if="loading" class="log-skeleton">
          <SkeletonBlock v-for="index in 8" :key="index" height="12px" width="92%" radius="3px" />
        </div>
        <div v-else-if="!matches.length" class="log-empty">
          {{ app.logs.length ? '没有匹配的日志。调整级别或搜索词。' : '还没有日志。连接隧道后，cloudflared 的输出会实时出现在这里。' }}
        </div>
        <template v-else>
          <LogLine
            v-for="line in matches"
            :key="line.seq"
            :line="line"
            :expanded="expanded.has(line.seq)"
            @toggle="toggleLine(line.seq)"
          />
        </template>
      </div>
    </section>

    <aside class="log-side reveal" style="--i: 3">
      <section class="card info-card">
        <h3>进程</h3>
        <div class="kv"><span>状态</span><strong>{{ stateMeta.label }}</strong></div>
        <div class="kv"><span>运行时长</span><strong>{{ runtime?.state === 'connected' ? formatDuration(runtime.uptimeSec) : '—' }}</strong></div>
        <div class="kv"><span>cloudflared</span><strong class="mono">{{ runtime?.cloudflaredVersion || app.meta?.cloudflaredVersion || '—' }}</strong></div>
        <div class="kv"><span>隧道</span><strong class="mono">{{ runtime?.tunnelName || '—' }}</strong></div>
      </section>
      <section class="card info-card">
        <h3>连接</h3>
        <div class="kv"><span>边缘节点</span><strong class="mono">{{ runtime?.edgeLocation || '—' }}</strong></div>
        <div class="kv"><span>协议</span><strong class="mono">{{ (runtime?.protocol || '—').toUpperCase() }}</strong></div>
        <div class="kv"><span>连接数</span><strong>{{ runtime?.connections ?? '—' }}</strong></div>
        <div class="kv"><span>Metrics</span><strong class="mono">:{{ app.meta?.metricsPort ?? '—' }}</strong></div>
      </section>
      <section class="card info-card">
        <h3>日志级别</h3>
        <div class="legend">
          <span><i class="inf"></i>INF 信息</span>
          <span><i class="wrn"></i>WRN 警告</span>
          <span><i class="err"></i>ERR 错误</span>
          <span><i class="dbg"></i>DBG 调试</span>
        </div>
        <p class="micro" style="margin: 10px 0 0">
          内存保留 {{ settings?.maxLogLines?.toLocaleString('en-US') || '—' }} 行，落盘 {{ settings?.logRetentionDays ?? '—' }} 天。点击任意日志可展开 JSON 详情。
        </p>
      </section>
    </aside>
  </div>
</template>
