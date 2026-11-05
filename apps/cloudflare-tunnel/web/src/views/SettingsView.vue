<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import Icon from '../components/Icon.vue'
import PageHead from '../components/PageHead.vue'
import SegmentedControl from '../components/SegmentedControl.vue'
import SelectControl from '../components/SelectControl.vue'
import SwitchControl from '../components/SwitchControl.vue'
import NoticeBar from '../components/NoticeBar.vue'
import SkeletonBlock from '../components/SkeletonBlock.vue'
import TokenHelpModal from '../components/TokenHelpModal.vue'
import { api } from '../lib/client'
import { apiErrorMessage } from '../lib/api'
import { confirmDialog } from '../lib/dialog'
import { copyText } from '../lib/clipboard'
import { setReduceMotion, setThemePref, reduceMotion, themePref } from '../lib/theme'
import { toast } from '../lib/toast'
import { app, noteError, refreshSettings } from '../stores/state'
import type { EdgeIpVersion, Protocol, Settings, SettingsUpdate, VerifyResult } from '../types'

const form = reactive({
  accountId: '',
  apiToken: '',
  protocol: 'auto' as Protocol,
  edgeIpVersion: 'auto' as EdgeIpVersion,
  autoConnect: false,
  autoReconnect: true,
  maxReconnect: 10,
  logRetentionDays: 30,
  maxLogLines: 2000,
})

const baseline = ref<Settings | null>(null)
const loading = ref(!app.settings)
const loadError = ref('')
const saving = ref(false)
const saveError = ref('')
const verifying = ref(false)
const verifyResult = ref<VerifyResult | null>(null)
const verifyError = ref('')
const tokenVisible = ref(false)
const tokenHelpOpen = ref(false)
const resetting = ref(false)
const accountTouched = ref(false)

const protocolOptions = [
  { value: 'auto', label: '自动' },
  { value: 'quic', label: 'QUIC' },
  { value: 'http2', label: 'HTTP/2' },
]

const edgeIpOptions = [
  { value: 'auto' as EdgeIpVersion, label: '自动' },
  { value: '4' as EdgeIpVersion, label: '仅 IPv4' },
  { value: '6' as EdgeIpVersion, label: '仅 IPv6' },
]

const themeOptions = [
  { value: 'system', label: '跟随系统' },
  { value: 'light', label: '浅色' },
  { value: 'dark', label: '深色' },
]

const retentionOptions = computed(() => {
  const values = new Set([7, 30, 90, 180, 365, form.logRetentionDays])
  return Array.from(values)
    .filter((value) => Number.isFinite(value) && value > 0)
    .sort((a, b) => a - b)
})

const maxLinesOptions = computed(() => {
  const values = new Set([1000, 2000, 5000, 10000, form.maxLogLines])
  return Array.from(values)
    .filter((value) => Number.isFinite(value) && value > 0)
    .sort((a, b) => a - b)
})

const retentionItems = computed(() =>
  retentionOptions.value.map((days) => ({ value: days, label: `${days} 天` })),
)

const maxLinesItems = computed(() =>
  maxLinesOptions.value.map((lines) => ({ value: lines, label: lines.toLocaleString('en-US') })),
)

const accountError = computed(() => {
  if (!accountTouched.value) return ''
  const value = form.accountId.trim()
  if (!value) return ''
  return /^[0-9a-f]{32}$/i.test(value) ? '' : 'Account ID 需要是 32 位十六进制字符'
})

const reconnectError = computed(() => {
  const value = Number(form.maxReconnect)
  if (!Number.isFinite(value) || value < 0 || value > 50) return '最大重连次数必须在 0–50 之间'
  return ''
})

const dirty = computed(() => {
  const base = baseline.value
  if (!base) return false
  return (
    form.accountId.trim().toLowerCase() !== base.accountId.toLowerCase() ||
    form.apiToken.length > 0 ||
    form.protocol !== base.protocol ||
    form.edgeIpVersion !== base.edgeIpVersion ||
    form.autoConnect !== base.autoConnect ||
    form.autoReconnect !== base.autoReconnect ||
    Number(form.maxReconnect) !== base.maxReconnect ||
    Number(form.logRetentionDays) !== base.logRetentionDays ||
    Number(form.maxLogLines) !== base.maxLogLines
  )
})

const verifyLine = computed(() => {
  if (verifyError.value) return { ok: false, text: verifyError.value }
  if (!verifyResult.value) return null
  const result = verifyResult.value
  if (!result.ok) return { ok: false, text: result.error || '验证未通过' }
  const parts = [`Token ${result.tokenStatus || 'active'}`]
  parts.push(result.tunnelAccess ? 'Tunnel ✓' : 'Tunnel ✗')
  parts.push(result.zoneAccess ? 'Zone ✓' : 'Zone ✗')
  return { ok: result.tunnelAccess && result.zoneAccess, text: `已验证 · ${parts.join(' · ')}` }
})

const tokenHint = computed(() => {
  if (!baseline.value) return '仅保存在本机 config/settings.json（0600）。'
  if (baseline.value.apiTokenSet) return `留空表示保留当前 Token（${baseline.value.apiTokenPreview}）。`
  return '尚未设置 Token；保存后才会写入本机配置。'
})

function hydrate(settings: Settings) {
  form.accountId = settings.accountId
  form.apiToken = ''
  form.protocol = settings.protocol
  form.edgeIpVersion = settings.edgeIpVersion
  form.autoConnect = settings.autoConnect
  form.autoReconnect = settings.autoReconnect
  form.maxReconnect = settings.maxReconnect
  form.logRetentionDays = settings.logRetentionDays
  form.maxLogLines = settings.maxLogLines
  baseline.value = { ...settings }
  accountTouched.value = false
  tokenVisible.value = false
}

async function load() {
  loading.value = !baseline.value
  loadError.value = ''
  if (!app.settings) {
    const error = await refreshSettings()
    if (error) loadError.value = error
  }
  if (app.settings) hydrate(app.settings)
  loading.value = false
}

function buildPayload(): SettingsUpdate {
  const payload: SettingsUpdate = {
    accountId: form.accountId.trim().toLowerCase(),
    protocol: form.protocol,
    edgeIpVersion: form.edgeIpVersion,
    autoConnect: form.autoConnect,
    autoReconnect: form.autoReconnect,
    maxReconnect: Number(form.maxReconnect),
    logRetentionDays: Number(form.logRetentionDays),
    maxLogLines: Number(form.maxLogLines),
  }
  if (form.apiToken) payload.apiToken = form.apiToken
  return payload
}

async function save() {
  if (!dirty.value || saving.value) return
  accountTouched.value = true
  if (accountError.value || reconnectError.value) {
    const message = accountError.value || reconnectError.value
    saveError.value = message
    toast(message, 'error')
    return
  }
  saving.value = true
  saveError.value = ''
  try {
    const updated = await api.updateSettings(buildPayload())
    app.settings = updated
    hydrate(updated)
    toast('设置已保存', 'success')
  } catch (err) {
    saveError.value = apiErrorMessage(err)
    toast(saveError.value, 'error')
  } finally {
    saving.value = false
  }
}

async function verify() {
  if (verifying.value) return
  verifying.value = true
  verifyError.value = ''
  try {
    const result = await api.verify({
      accountId: form.accountId.trim(),
      ...(form.apiToken ? { apiToken: form.apiToken } : {}),
    })
    verifyResult.value = result
    if (result.ok) toast(`账户验证通过${app.account?.zones ? ` · ${app.account.zones} 个 Zone` : ''}`, 'success')
    else toast(result.error || '账户验证未通过', 'error')
  } catch (err) {
    verifyResult.value = null
    verifyError.value = noteError(err)
    toast(verifyError.value, 'error')
  } finally {
    verifying.value = false
  }
}

async function resetConfig() {
  if (resetting.value) return
  resetting.value = true
  const result = await confirmDialog({
    title: '重置本地配置？',
    message:
      '连接与日志设置会恢复默认，主题与动效偏好也会重置。账户与 API Token 会保留，隧道与路由不会删除。',
    confirmText: '重置配置',
    tone: 'danger',
    action: async () => {
      const updated = await api.updateSettings({
        accountId: (baseline.value?.accountId || '').toLowerCase(),
        protocol: 'auto',
        edgeIpVersion: 'auto',
        autoConnect: false,
        autoReconnect: true,
        maxReconnect: 10,
        logRetentionDays: 30,
        maxLogLines: 2000,
      })
      app.settings = updated
      setThemePref('system')
      setReduceMotion(false)
      hydrate(updated)
    },
  })
  resetting.value = false
  if (result.ok) toast('本地配置已重置 · 账户与 Token 保留', 'success')
}

function copyPath(value: string | undefined, label: string) {
  if (!value) return
  void copyText(value, `${label} 已复制`)
}

function onThemeChange(value: string) {
  setThemePref(value as 'system' | 'light' | 'dark')
}

function setProtocol(value: string) {
  form.protocol = value as Protocol
}

function onRefresh() {
  if (dirty.value) {
    toast('有未保存的更改，保存或放弃后再刷新', 'error')
    return
  }
  void load()
}

function onBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

onBeforeRouteLeave(async () => {
  if (!dirty.value) return true
  const result = await confirmDialog({
    title: '放弃未保存的更改？',
    message: '设置还没有保存，离开后修改会丢失。',
    confirmText: '放弃更改',
    tone: 'danger',
  })
  return result.ok
})

onMounted(() => {
  if (app.settings) hydrate(app.settings)
  void load()
  window.addEventListener('ct:refresh', onRefresh)
  window.addEventListener('beforeunload', onBeforeUnload)
})

onBeforeUnmount(() => {
  window.removeEventListener('ct:refresh', onRefresh)
  window.removeEventListener('beforeunload', onBeforeUnload)
})
</script>

<template>
  <PageHead title="设置" sub="账户、连接行为、外观与日志保留。修改后需要保存。">
    <template #actions>
      <span class="save-state" :class="{ 'is-dirty': dirty }">{{ dirty ? '有未保存的更改' : '已保存' }}</span>
      <button class="btn primary" type="button" :disabled="!dirty || saving" @click="save">
        <span v-if="saving" class="spinner" aria-hidden="true"></span>
        {{ saving ? '保存中' : '保存更改' }}
      </button>
    </template>
  </PageHead>

  <NoticeBar v-if="loadError" tone="danger" icon="alert" title="无法读取设置">
    {{ loadError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="onRefresh">重试</button>
    </template>
  </NoticeBar>
  <NoticeBar v-if="saveError" tone="danger" icon="alert" title="保存失败">
    {{ saveError }}
  </NoticeBar>

  <div v-if="loading" class="settings-wrap">
    <section class="card settings-card">
      <div class="card-head"><div><h3>Cloudflare 账户</h3><p>正在读取设置…</p></div></div>
      <div class="settings-loading">
        <SkeletonBlock v-for="index in 4" :key="index" height="36px" />
      </div>
    </section>
  </div>

  <div v-else class="settings-wrap" :class="{ 'has-save-dock': dirty || saving || saveError }">
    <section class="card settings-card reveal" style="--i: 1">
      <div class="card-head">
        <div>
          <h3>Cloudflare 账户</h3>
          <p>使用最小权限 API Token；应用不会保存账户密码。</p>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>Account ID</strong>
          <small>Cloudflare 账户 ID，用于调用 Tunnel API。</small>
        </div>
        <div class="setting-control">
          <div class="field inline-field" :class="{ 'has-error': accountError }">
            <input
              v-model="form.accountId"
              class="input mono"
              style="width: 100%"
              spellcheck="false"
              autocomplete="off"
              aria-label="Account ID"
              placeholder="32 位十六进制账户 ID"
              @blur="accountTouched = true"
            />
            <span class="error">{{ accountError || 'Account ID 需要是 32 位十六进制字符' }}</span>
          </div>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>API Token</strong>
          <small>{{ tokenHint }}</small>
        </div>
        <div class="setting-control">
          <div class="input-wrap">
            <input
              v-model="form.apiToken"
              class="input mono"
              :type="tokenVisible ? 'text' : 'password'"
              :placeholder="baseline?.apiTokenPreview || '粘贴 Cloudflare API Token'"
              spellcheck="false"
              autocomplete="new-password"
              aria-label="API Token"
            />
            <button
              class="icon-btn"
              type="button"
              :aria-label="tokenVisible ? '隐藏 Token' : '显示 Token'"
              @click="tokenVisible = !tokenVisible"
            >
              <Icon :name="tokenVisible ? 'eye-off' : 'eye'" />
            </button>
          </div>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>连接验证</strong>
          <small>验证 Token 是否可访问当前账户与 Zone。</small>
        </div>
        <div class="setting-control verify-control">
          <span v-if="verifyLine" class="verify-line" :class="{ 'is-bad': !verifyLine.ok }">
            <Icon :name="verifyLine.ok ? 'check' : 'alert'" />
            {{ verifyLine.text }}
          </span>
          <span v-else class="micro">尚未验证</span>
          <button class="btn small ghost" type="button" :disabled="verifying" @click="verify">
            <span v-if="verifying" class="spinner" aria-hidden="true"></span>
            <Icon v-else name="shield" />
            {{ verifying ? '验证中' : '重新验证' }}
          </button>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>权限说明</strong>
          <small>查看应用所需的最小权限集。</small>
        </div>
        <div class="setting-control" style="justify-content: flex-start">
          <button class="link" type="button" @click="tokenHelpOpen = true">如何创建最小权限 Token →</button>
        </div>
      </div>
    </section>

    <section class="card settings-card reveal" style="--i: 2">
      <div class="card-head">
        <div>
          <h3>连接</h3>
          <p>cloudflared 与 Cloudflare Edge 的握手方式，保存后下次连接生效。</p>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>连接协议</strong>
          <small>UDP 受限的网络请切换到 HTTP/2；自动由 cloudflared 协商。</small>
        </div>
        <div class="setting-control">
          <SegmentedControl
            :model-value="form.protocol"
            :options="protocolOptions"
            aria-label="连接协议"
            @update:model-value="setProtocol"
          />
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>边缘 IP 版本</strong>
          <small>IPv6 可用时通常延迟更低；不确定时选自动。</small>
        </div>
        <div class="setting-control">
          <SelectControl
            v-model="form.edgeIpVersion"
            :options="edgeIpOptions"
            aria-label="边缘 IP 版本"
            style="min-width: 150px"
          />
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>失败时自动重试</strong>
          <small>cloudflared 异常退出后按指数退避重启（5s → 60s）。</small>
        </div>
        <div class="setting-control">
          <SwitchControl v-model="form.autoReconnect" label="失败时自动重试" />
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>最大重连次数</strong>
          <small>达到次数后保持异常状态，等待手动连接（0–50）。</small>
        </div>
        <div class="setting-control">
          <div class="field inline-field number-field" :class="{ 'has-error': reconnectError }">
            <input
              v-model.number="form.maxReconnect"
              class="input mono"
              type="number"
              min="0"
              max="50"
              step="1"
              aria-label="最大重连次数"
              :disabled="!form.autoReconnect"
            />
            <span class="error">{{ reconnectError || '最大重连次数必须在 0–50 之间' }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="card settings-card reveal" style="--i: 3">
      <div class="card-head">
        <div>
          <h3>启动</h3>
          <p>fnOS 重启或应用重新打开后的行为。</p>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>开机自动连接</strong>
          <small>fnOS 启动后自动恢复上一次的隧道连接。</small>
        </div>
        <div class="setting-control">
          <SwitchControl v-model="form.autoConnect" label="开机自动连接" />
        </div>
      </div>
    </section>

    <section class="card settings-card reveal" style="--i: 4">
      <div class="card-head">
        <div>
          <h3>外观</h3>
          <p>仅保存在浏览器本地，不写入服务端配置。</p>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>主题</strong>
          <small>浅色与深色共用同一套布局和状态色。</small>
        </div>
        <div class="setting-control">
          <SegmentedControl
            :model-value="themePref"
            :options="themeOptions"
            aria-label="主题"
            @update:model-value="onThemeChange"
          />
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>减少动效</strong>
          <small>关闭页面进入、脉冲和打包动画；保留必要的状态过渡。</small>
        </div>
        <div class="setting-control">
          <SwitchControl
            :model-value="reduceMotion"
            label="减少动效"
            @update:model-value="setReduceMotion($event)"
          />
        </div>
      </div>
    </section>

    <section class="card settings-card reveal" style="--i: 5">
      <div class="card-head">
        <div>
          <h3>日志</h3>
          <p>只保留本地日志，不自动上传。</p>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>保留天数</strong>
          <small>超过期限的日志文件会被清理（1–365 天）。</small>
        </div>
        <div class="setting-control">
          <SelectControl
            v-model="form.logRetentionDays"
            :options="retentionItems"
            aria-label="日志保留天数"
            style="min-width: 120px"
          />
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>内存行数</strong>
          <small>日志页面保留的最大行数（200–10,000）。</small>
        </div>
        <div class="setting-control">
          <SelectControl
            v-model="form.maxLogLines"
            :options="maxLinesItems"
            aria-label="内存行数"
            style="min-width: 120px"
          />
        </div>
      </div>
    </section>

    <section class="card settings-card reveal" style="--i: 6">
      <div class="card-head">
        <div>
          <h3>存储位置</h3>
          <p>来自 /meta；持久数据位于应用共享目录，重启不丢失。</p>
        </div>
      </div>
      <div class="setting-row path-row">
        <div class="setting-copy"><strong>数据目录</strong><small>state.json · activity.json</small></div>
        <div class="setting-control path-control">
          <span class="mono path-value">{{ app.meta?.dataDir || '—' }}</span>
          <button class="icon-btn" type="button" aria-label="复制数据目录" @click="copyPath(app.meta?.dataDir, '数据目录')">
            <Icon name="copy" />
          </button>
        </div>
      </div>
      <div class="setting-row path-row">
        <div class="setting-copy"><strong>配置目录</strong><small>settings.json · apikeys.json（0600）</small></div>
        <div class="setting-control path-control">
          <span class="mono path-value">{{ app.meta?.configDir || '—' }}</span>
          <button class="icon-btn" type="button" aria-label="复制配置目录" @click="copyPath(app.meta?.configDir, '配置目录')">
            <Icon name="copy" />
          </button>
        </div>
      </div>
      <div class="setting-row path-row">
        <div class="setting-copy"><strong>日志目录</strong><small>app.log · cloudflared.log</small></div>
        <div class="setting-control path-control">
          <span class="mono path-value">{{ app.meta?.logDir || '—' }}</span>
          <button class="icon-btn" type="button" aria-label="复制日志目录" @click="copyPath(app.meta?.logDir, '日志目录')">
            <Icon name="copy" />
          </button>
        </div>
      </div>
    </section>

    <section class="card settings-card danger-zone reveal" style="--i: 7">
      <div class="card-head">
        <div>
          <h3>危险区</h3>
          <p>这些操作只影响本地配置。</p>
        </div>
      </div>
      <div class="setting-row">
        <div class="setting-copy">
          <strong>重置本地配置</strong>
          <small>恢复默认连接与日志设置；账户与 Token 保留，不删除隧道和路由。</small>
        </div>
        <div class="setting-control" style="justify-content: flex-start">
          <button class="btn danger" type="button" :disabled="resetting" @click="resetConfig">
            <span v-if="resetting" class="spinner" aria-hidden="true"></span>
            <Icon v-else name="refresh" />
            重置配置
          </button>
        </div>
      </div>
    </section>
  </div>

  <Teleport to="body">
    <Transition name="save-dock">
      <div v-if="dirty || saving || saveError" class="settings-save-dock" role="status" aria-live="polite">
        <span class="save-state" :class="saveError ? 'is-error' : 'is-dirty'">
          {{ saveError ? '保存失败' : '有未保存的更改' }}
        </span>
        <button class="btn primary" type="button" :disabled="saving" @click="save">
          <span v-if="saving" class="spinner" aria-hidden="true"></span>
          {{ saving ? '保存中' : saveError ? '重试保存' : '保存更改' }}
        </button>
      </div>
    </Transition>
  </Teleport>

  <TokenHelpModal v-if="tokenHelpOpen" @close="tokenHelpOpen = false" />
</template>
