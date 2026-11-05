<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from '../components/Icon.vue'
import PageHead from '../components/PageHead.vue'
import EmptyState from '../components/EmptyState.vue'
import NoticeBar from '../components/NoticeBar.vue'
import SkeletonBlock from '../components/SkeletonBlock.vue'
import CreateKeyModal from '../components/CreateKeyModal.vue'
import SecretModal from '../components/SecretModal.vue'
import { api } from '../lib/client'
import { apiErrorMessage } from '../lib/api'
import { confirmDialog } from '../lib/dialog'
import { copyText } from '../lib/clipboard'
import { formatDate, formatRelative } from '../lib/format'
import { toast } from '../lib/toast'
import { noteError, app } from '../stores/state'
import type { ApiKey } from '../types'

interface Endpoint {
  method: string
  path: string
  description: string
}

const keys = ref<ApiKey[]>([])
const loading = ref(true)
const loadError = ref('')
const createOpen = ref(false)
const creating = ref(false)
const createError = ref('')
const tokenOpen = ref(false)
const tokenValue = ref('')
const tokenKeyName = ref('')

const origin = computed(() => window.location.origin)
const apiBase = computed(() => `${window.location.origin}/api/v1`)

const endpoints: Endpoint[] = [
  { method: 'GET', path: '/health', description: '健康检查与版本，公开接口' },
  { method: 'GET', path: '/meta', description: '版本、数据目录、监听地址与 metrics 端口' },
  { method: 'GET', path: '/overview', description: '概览聚合：运行状态、计数、metrics、活动' },
  { method: 'GET', path: '/settings', description: '读取账户与连接设置（不含 Token）' },
  { method: 'PUT', path: '/settings', description: '更新设置，可选 apiToken' },
  { method: 'POST', path: '/settings/verify', description: '验证账户与 Token 的权限' },
  { method: 'GET', path: '/account', description: '账户信息与 Zone 数量' },
  { method: 'GET', path: '/tunnels', description: '列出全部隧道' },
  { method: 'POST', path: '/tunnels', description: '创建隧道' },
  { method: 'DELETE', path: '/tunnels/{id}', description: '删除隧道，运行中会先停止' },
  { method: 'GET', path: '/tunnels/{id}/token', description: '读取隧道 Token（仅会话）' },
  { method: 'GET', path: '/tunnels/{id}/routes', description: '读取 ingress 规则与兜底服务' },
  { method: 'PUT', path: '/tunnels/{id}/routes', description: '写入 ingress 规则与兜底服务' },
  { method: 'POST', path: '/dns', description: '创建或更新指向隧道的 CNAME' },
  { method: 'GET', path: '/runtime', description: 'cloudflared 运行状态' },
  { method: 'POST', path: '/runtime/connect', description: '连接指定隧道（202）' },
  { method: 'POST', path: '/runtime/disconnect', description: '断开当前隧道' },
  { method: 'GET', path: '/runtime/metrics', description: '请求、错误、连接数与 RTT' },
  { method: 'GET', path: '/logs', description: '查询日志（limit / afterSeq / level / q）' },
  { method: 'GET', path: '/logs/download', description: '下载日志文本' },
  { method: 'POST', path: '/logs/clear', description: '清空内存与落盘日志' },
  { method: 'GET', path: '/events', description: 'SSE：hello / status / log / activity / metrics' },
  { method: 'GET', path: '/activity', description: '本机活动记录' },
  { method: 'GET', path: '/keys', description: 'API 凭证列表' },
  { method: 'POST', path: '/keys', description: '创建凭证，明文 Token 只返回一次' },
  { method: 'DELETE', path: '/keys/{id}', description: '吊销凭证' },
]

const curlSnippet = computed(
  () =>
    `# 查询隧道状态\ncurl http://<your-nas>:18960/api/v1/runtime \\\n  -H "Authorization: Bearer ct_..."\n\n# 触发连接\ncurl -X POST http://<your-nas>:18960/api/v1/runtime/connect \\\n  -H "Authorization: Bearer ct_..." \\\n  -H "Content-Type: application/json" \\\n  -d '{"tunnelId":"<tunnel-id>"}'`,
)

function methodClass(method: string): string {
  return (
    {
      GET: 'm-get',
      POST: 'm-post',
      PUT: 'm-put',
      DELETE: 'm-delete',
    }[method] || 'm-get'
  )
}

function lastUsed(key: ApiKey): string {
  return key.lastUsedAt ? formatRelative(key.lastUsedAt) : '从未使用'
}

async function load() {
  loading.value = !keys.value.length
  try {
    const result = await api.keys()
    keys.value = result.keys
    loadError.value = ''
  } catch (err) {
    loadError.value = noteError(err)
  } finally {
    loading.value = false
  }
}

function copyBase() {
  void copyText(apiBase.value, 'API 地址已复制')
}

function openCreate() {
  createError.value = ''
  createOpen.value = true
}

async function submitCreate(name: string) {
  creating.value = true
  createError.value = ''
  try {
    const result = await api.createKey(name)
    keys.value = [result.key, ...keys.value]
    createOpen.value = false
    tokenValue.value = result.token
    tokenKeyName.value = result.key.name
    tokenOpen.value = true
    toast('凭证已创建', 'success')
  } catch (err) {
    createError.value = apiErrorMessage(err)
  } finally {
    creating.value = false
  }
}

async function revoke(key: ApiKey) {
  const result = await confirmDialog({
    title: '吊销凭证？',
    message: `吊销后，使用 ${key.name}（${key.prefix}…）的客户端会立即收到 401，无法恢复。`,
    confirmText: '吊销凭证',
    tone: 'danger',
    action: async () => {
      await api.deleteKey(key.id)
      await load()
    },
  })
  if (result.ok) toast('凭证已吊销', 'success')
}

function onRefresh() {
  void load()
}

onMounted(() => {
  void load()
  window.addEventListener('ct:refresh', onRefresh)
})

onBeforeUnmount(() => {
  window.removeEventListener('ct:refresh', onRefresh)
})
</script>

<template>
  <PageHead
    eyebrow="集成"
    title="接入"
    sub="通过本机 HTTP API 把隧道能力接进其他应用。凭证按用途命名，可随时吊销。"
  />

  <div class="banner reveal" style="--i: 1">
    <Icon name="info" />
    <div>
      <strong>接口与 UI 共用同一套 DTO。</strong>
      浏览器使用会话 Cookie；外部客户端使用 <span class="mono">Authorization: Bearer ct_…</span>。MCP server 不在本期范围，但
      /api/v1 与 DTO 已为它预留。
    </div>
  </div>

  <NoticeBar v-if="loadError" tone="danger" icon="alert" title="无法读取凭证">
    {{ loadError }}
    <template #action>
      <button class="btn small ghost" type="button" @click="onRefresh">重试</button>
    </template>
  </NoticeBar>

  <div class="integration-grid">
    <div class="integration-main">
      <section class="card endpoint-card reveal" style="--i: 2">
        <h3><Icon name="braces" />HTTP API</h3>
        <p>给脚本、自动化或自建面板使用。所有调用都在本机完成，不经过第三方。</p>
        <div class="endpoint-row">
          <span class="mono">{{ apiBase }}</span>
          <button class="icon-btn" type="button" aria-label="复制 API 地址" @click="copyBase">
            <Icon name="copy" />
          </button>
        </div>
        <div class="endpoint-meta">
          <Icon name="lock" /> Bearer Token · 只监听局域网 · 监听 {{ app.meta?.listen || '0.0.0.0:18960' }}
        </div>
        <div class="code-block-wrap">
          <pre class="code-block">{{ curlSnippet }}</pre>
          <button class="btn small ghost code-copy" type="button" @click="copyText(curlSnippet, 'curl 示例已复制')">
            <Icon name="copy" />
            复制
          </button>
        </div>
      </section>

      <section class="card endpoint-card reveal" style="--i: 3">
        <h3><Icon name="list" />接口参考</h3>
        <p>与 CONTRACT 一致；除 /health 外都需要会话 Cookie 或 Bearer Token。</p>
        <div class="table-wrap endpoint-table">
          <table class="data-table">
            <thead>
              <tr>
                <th>方法</th>
                <th>路径</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="endpoint in endpoints" :key="`${endpoint.method} ${endpoint.path}`">
                <td><span class="pill method-pill" :class="methodClass(endpoint.method)">{{ endpoint.method }}</span></td>
                <td><span class="host">{{ endpoint.path }}</span></td>
                <td class="endpoint-desc">{{ endpoint.description }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <section class="card credentials reveal" style="--i: 4">
      <div class="card-head">
        <div>
          <h3>API 凭证</h3>
          <p>可调用全部 /api/v1 接口，可随时吊销</p>
        </div>
        <button class="btn small ghost" type="button" @click="openCreate">
          <Icon name="plus" />
          新建
        </button>
      </div>
      <div class="credential-list">
        <div v-if="loading" class="credential-loading">
          <SkeletonBlock v-for="index in 2" :key="index" height="38px" />
        </div>
        <EmptyState
          v-else-if="!keys.length"
          icon="key"
          title="还没有 API 凭证"
          message="为脚本或面板创建一个凭证；明文只在创建时显示一次。"
        />
        <div v-for="key in keys" :key="key.id" class="credential-row">
          <span class="credential-icon"><Icon name="key" /></span>
          <span class="credential-copy">
            <strong>{{ key.name }}</strong>
            <small class="mono">{{ key.prefix }}… · 创建于 {{ formatDate(key.createdAt) }}</small>
          </span>
          <span class="pill neutral credential-used">{{ lastUsed(key) }}</span>
          <button
            class="icon-btn"
            type="button"
            :aria-label="`吊销凭证 ${key.name}`"
            title="吊销凭证"
            @click="revoke(key)"
          >
            <Icon name="trash" />
          </button>
        </div>
      </div>
      <div class="card-foot" style="padding-top: 12px">
        <span class="micro">Token 只在创建时显示一次；服务端以 0600 保存 SHA-256 哈希。</span>
      </div>
    </section>
  </div>

  <CreateKeyModal
    v-if="createOpen"
    :submitting="creating"
    :server-error="createError"
    @close="createOpen = false"
    @submit="submitCreate"
  />

  <SecretModal
    v-if="tokenOpen"
    eyebrow="API 凭证"
    title="已生成"
    :secret="tokenValue"
    :warning="`Token 只显示一次。关闭后只能吊销并重新生成（${tokenKeyName}）。`"
    note="外部客户端在请求头中携带此 Token；浏览器 UI 不需要它。"
    confirm-label="我已保存"
    @close="tokenOpen = false"
  />

  <span class="sr-only" aria-live="polite">{{ origin }}</span>
</template>
