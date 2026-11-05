<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import Icon from './components/Icon.vue'
import ToastHost from './components/ToastHost.vue'
import appIcon from './assets/brand/app-icon.png'
import ConfirmDialog from './components/ConfirmDialog.vue'
import NoticeBar from './components/NoticeBar.vue'
import { toggleTheme, resolvedTheme } from './lib/theme'
import { toast } from './lib/toast'
import { shortId } from './lib/format'
import { app, bootstrap, isDemo, runtimeState, startEvents, stopEvents } from './stores/state'

const route = useRoute()
const sidebarOpen = ref(false)
const refreshing = ref(false)
const pageEl = ref<HTMLElement | null>(null)

const navGroups = [
  {
    label: '工作区',
    links: [
      { name: 'overview', title: '概览', icon: 'grid' },
      { name: 'tunnels', title: '隧道', icon: 'waypoints', count: 'tunnels' },
      { name: 'routes', title: '路由', icon: 'branch', count: 'routes' },
    ],
  },
  {
    label: '诊断',
    links: [{ name: 'logs', title: '运行日志', icon: 'terminal', live: true }],
  },
  {
    label: '集成',
    links: [{ name: 'api', title: '接入', icon: 'braces' }],
  },
  {
    label: '配置',
    links: [{ name: 'settings', title: '设置', icon: 'sliders' }],
  },
] as const

const pageTitle = computed(() => (route.meta.title as string) || '概览')
const accountName = computed(() => app.account?.accountName || 'Cloudflare')
const accountIdLabel = computed(() =>
  app.account?.accountId ? shortId(app.account.accountId) : app.settings?.apiTokenSet ? 'Cloudflare' : '未配置',
)
const avatarLabel = computed(() => {
  const name = app.account?.accountName || 'Cloudflare'
  const words = name.trim().split(/\s+/)
  if (words.length >= 2) return (words[0][0] + words[1][0]).toUpperCase()
  return name.slice(0, 2).toUpperCase()
})
const hostLabel = computed(() => window.location.hostname || '本机')
const hostSub = computed(() => `局域网 · ${hostLabel.value}`)
const hostState = computed(() => {
  const state = runtimeState.value
  if (state === 'connected') return 'healthy'
  if (state === 'error') return 'danger'
  if (state === 'connecting' || state === 'reconnecting' || state === 'stopping') return 'warning'
  return ''
})
const backendVersion = computed(() => app.meta?.version || '')
const versionLabel = computed(() => (backendVersion.value ? `v${backendVersion.value}` : 'dev'))
const tunnelCount = computed(() => app.tunnels.length || app.overview?.tunnelCount || 0)
const routeCount = computed(() => app.routeCount)

function countFor(key: string): number {
  if (key === 'tunnels') return tunnelCount.value
  return routeCount.value
}

function closeSidebar() {
  sidebarOpen.value = false
}

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

function onThemeToggle() {
  toggleTheme()
}

function onRefresh() {
  if (refreshing.value) return
  refreshing.value = true
  window.dispatchEvent(new CustomEvent('ct:refresh'))
  toast('状态已刷新', 'success')
  window.setTimeout(() => {
    refreshing.value = false
  }, 700)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && sidebarOpen.value) closeSidebar()
}

function onResize() {
  if (window.innerWidth > 920 && sidebarOpen.value) closeSidebar()
}

function restartPageAnimation() {
  const el = pageEl.value
  if (!el) return
  el.classList.remove('is-entering')
  void el.offsetWidth
  el.classList.add('is-entering')
}

watch(
  () => route.fullPath,
  () => {
    closeSidebar()
    document.title = `${pageTitle.value} · Cloudflare Tunnel`
    void nextTick(() => {
      if (pageEl.value) pageEl.value.scrollTop = 0
      restartPageAnimation()
      pageEl.value?.focus({ preventScroll: true })
    })
  },
  { immediate: true },
)

function reloadPage() {
  window.location.reload()
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onResize)
  void bootstrap()
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onResize)
  stopEvents()
})

function retryBoot() {
  void bootstrap()
  startEvents()
}
</script>

<template>
  <div class="ambient" aria-hidden="true"></div>

  <div class="app" :class="{ 'sidebar-open': sidebarOpen }">
    <aside class="sidebar" id="sidebar">
      <RouterLink class="brand" to="/overview" aria-label="Tunnel 首页" @click="closeSidebar">
        <img class="brand-mark" :src="appIcon" alt="" aria-hidden="true" />
        <span class="brand-copy">
          <strong>Tunnel</strong>
          <small>for fnOS</small>
        </span>
      </RouterLink>

      <nav class="nav" aria-label="主导航">
        <div v-for="group in navGroups" :key="group.label" class="nav-group">
          <p class="nav-label">{{ group.label }}</p>
          <RouterLink
            v-for="link in group.links"
            :key="link.name"
            class="nav-link"
            :class="{ 'is-active': route.name === link.name }"
            :to="{ name: link.name }"
            :aria-current="route.name === link.name ? 'page' : undefined"
            @click="closeSidebar"
          >
            <span class="nav-icon"><Icon :name="link.icon" /></span>
            <span>{{ link.title }}</span>
            <span v-if="'count' in link && link.count" class="nav-count">{{ countFor(link.count) }}</span>
            <span v-else-if="'live' in link && link.live" class="live-dot" aria-hidden="true"></span>
          </RouterLink>
        </div>
      </nav>

      <div class="sidebar-foot">
        <div class="host-card">
          <span class="status-dot" :class="hostState" aria-hidden="true"></span>
          <div class="host-copy">
            <strong>{{ hostLabel }}</strong>
            <small>{{ hostSub }}</small>
          </div>
          <span class="host-signal" aria-hidden="true"><Icon name="wifi" /></span>
        </div>
        <div class="side-row">
          <button
            class="icon-btn"
            id="theme-toggle"
            type="button"
            :aria-label="resolvedTheme === 'dark' ? '切换到浅色主题' : '切换到深色主题'"
            :title="resolvedTheme === 'dark' ? '切换到浅色主题' : '切换到深色主题'"
            @click="onThemeToggle"
          >
            <Icon :name="resolvedTheme === 'dark' ? 'sun' : 'moon'" />
          </button>
          <span class="version-tag" :title="backendVersion ? `后端 ${backendVersion}` : 'Tunnel for fnOS'">{{ versionLabel }}</span>
        </div>
      </div>
    </aside>

    <div class="sidebar-scrim" :hidden="!sidebarOpen" @click="closeSidebar"></div>

    <div class="main">
      <header class="topbar">
        <div class="topbar-left">
          <button
            class="icon-btn mobile-menu"
            type="button"
            :aria-label="sidebarOpen ? '关闭导航' : '打开导航'"
            :aria-expanded="sidebarOpen"
            @click="toggleSidebar"
          >
            <Icon name="menu" />
          </button>
          <div class="crumbs">
            <span>控制台</span>
            <span class="crumb-sep">/</span>
            <strong>{{ pageTitle }}</strong>
          </div>
        </div>
        <div class="topbar-right">
          <span v-if="isDemo" class="prototype-badge demo-badge" title="后端以 -demo 运行，数据存放在临时目录">演示模式</span>
          <button
            class="icon-btn"
            :class="{ 'is-spinning': refreshing }"
            type="button"
            aria-label="刷新状态"
            title="刷新状态"
            @click="onRefresh"
          >
            <Icon name="refresh" />
          </button>
          <button class="account-chip" type="button" aria-label="打开设置" @click="$router.push('/settings')">
            <span class="avatar">{{ avatarLabel }}</span>
            <span class="account-copy">
              <strong>{{ accountName }}</strong>
              <small>{{ accountIdLabel }}</small>
            </span>
            <span class="chevron"><Icon name="chevron-down" /></span>
          </button>
        </div>
      </header>

      <main ref="pageEl" class="page" id="page" tabindex="-1">
        <div v-if="app.sessionExpired" class="app-notice">
          <NoticeBar tone="danger" icon="alert" title="会话已过期">
            服务重启会失效浏览器会话。刷新页面即可建立新的会话，正在进行的操作需要重新提交。
            <template #action>
              <button class="btn small ghost" type="button" @click="reloadPage">重新加载</button>
            </template>
          </NoticeBar>
        </div>
        <div v-else-if="app.sse === 'closed' && !app.booting" class="app-notice">
          <NoticeBar tone="warning" icon="alert" title="实时连接已断开">
            正在自动重连（第 {{ app.sseAttempt }} 次）。状态与日志可能暂时不是最新的。
            <template #action>
              <button class="btn small ghost" type="button" @click="startEvents">立即重试</button>
            </template>
          </NoticeBar>
        </div>

        <div class="page-inner">
          <div v-if="app.bootError" class="card boot-error reveal" style="--i: 0">
            <div class="card-head">
              <div>
                <h3>无法连接到应用服务</h3>
                <p>{{ app.bootError }}</p>
              </div>
            </div>
            <div class="card-foot">
              <button class="btn primary" type="button" @click="retryBoot">
                <Icon name="refresh" />
                重试
              </button>
            </div>
          </div>
          <RouterView v-else />
        </div>
      </main>
    </div>
  </div>

  <ToastHost />
  <ConfirmDialog />
</template>
