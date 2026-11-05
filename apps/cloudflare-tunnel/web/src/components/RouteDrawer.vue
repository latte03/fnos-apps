<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import Icon from './Icon.vue'
import OverlayLayer from './OverlayLayer.vue'
import SwitchControl from './SwitchControl.vue'
import { shortId } from '../lib/format'
import type { Route } from '../types'

const props = defineProps<{
  mode: 'add' | 'edit'
  initial: Route | null
  tunnelId: string
  tunnelName: string
  existingHostnames: string[]
  autoDnsDefault: boolean
  submitting: boolean
  serverError: string
}>()

const emit = defineEmits<{
  close: []
  save: [payload: { hostname: string; path: string; service: string; autoDns: boolean }]
}>()

const hostname = ref(props.initial?.hostname ?? '')
const service = ref(props.initial?.service ?? '')
const path = ref(props.initial?.path && props.initial.path !== '/' ? props.initial.path : '')
const autoDns = ref(props.mode === 'add' ? props.autoDnsDefault : false)
const hostError = ref('')
const serviceError = ref('')
const pathError = ref('')
const advancedOpen = ref(Boolean(props.initial?.path && props.initial.path !== '/'))

const hostInput = ref<HTMLInputElement | null>(null)

const title = computed(() => (props.mode === 'edit' ? '编辑路由' : '添加路由'))
const eyebrow = computed(() => (props.mode === 'edit' ? '编辑' : '新增'))

watch(hostname, () => {
  hostError.value = ''
})
watch(service, () => {
  serviceError.value = ''
})
watch(path, () => {
  pathError.value = ''
})

function validate(): boolean {
  const host = hostname.value.trim().toLowerCase()
  let ok = true
  if (!/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$/.test(host)) {
    hostError.value = '请输入有效域名，例如 app.example.com'
    ok = false
  } else if (
    props.existingHostnames.some(
      (existing) => existing === host && existing !== (props.initial?.hostname ?? '').toLowerCase(),
    )
  ) {
    hostError.value = '该域名已存在一条规则'
    ok = false
  }
  if (!/^(?:https?|tcp|ssh|rdp|smb):\/\/\S+|^unix:\S+/.test(service.value.trim())) {
    serviceError.value = '请输入有效的服务地址，例如 http://127.0.0.1:5666'
    ok = false
  }
  const pathValue = path.value.trim()
  if (pathValue && !pathValue.startsWith('/')) {
    pathError.value = '路径必须以 / 开头'
    ok = false
  }
  if (!ok) {
    void nextTick(() => {
      const first = document.querySelector<HTMLInputElement>('.drawer .field.has-error .input')
      ;(first || hostInput.value)?.focus()
    })
  }
  return ok
}

function submit() {
  if (props.submitting) return
  if (!validate()) return
  emit('save', {
    hostname: hostname.value.trim().toLowerCase(),
    service: service.value.trim(),
    path: path.value.trim() || '/',
    autoDns: autoDns.value,
  })
}

function onOpened() {
  void nextTick(() => hostInput.value?.focus())
}
</script>

<template>
  <OverlayLayer kind="drawer" initial-focus="#route-host" @close="emit('close')" @opened="onOpened">
    <aside class="drawer" role="dialog" aria-modal="true" aria-labelledby="route-drawer-title">
      <div class="drawer-head">
        <div>
          <p class="eyebrow">{{ eyebrow }}</p>
          <h2 id="route-drawer-title">{{ title }}</h2>
          <p>{{ tunnelName }} · 保存后写入 tunnel config</p>
        </div>
        <button class="icon-btn" type="button" aria-label="关闭" :disabled="submitting" @click="emit('close')">
          <Icon name="x" />
        </button>
      </div>
      <form class="drawer-body" novalidate @submit.prevent="submit">
        <div class="form-grid">
          <div class="field" :class="{ 'has-error': hostError }">
            <label for="route-host">域名</label>
            <input
              id="route-host"
              ref="hostInput"
              v-model="hostname"
              class="input mono"
              name="hostname"
              placeholder="app.example.com"
              autocomplete="off"
              spellcheck="false"
              :disabled="submitting"
            />
            <span class="hint">需要已托管在 Cloudflare 的域名；保存后可自动创建 CNAME。</span>
            <span class="error">{{ hostError || '请输入有效域名，例如 app.example.com' }}</span>
          </div>
          <div class="field" :class="{ 'has-error': serviceError }">
            <label for="route-service">内网服务</label>
            <input
              id="route-service"
              v-model="service"
              class="input mono"
              name="service"
              placeholder="http://127.0.0.1:5666"
              autocomplete="off"
              spellcheck="false"
              :disabled="submitting"
            />
            <span class="hint">支持 http://、https://、tcp://、ssh://、rdp://、smb:// 和 unix:。</span>
            <span class="error">{{ serviceError || '请输入有效的服务地址' }}</span>
          </div>
          <label class="check-row">
            <SwitchControl v-model="autoDns" label="自动创建 CNAME" />
            <span class="check-copy">
              <strong>自动创建 CNAME</strong>
              <small>指向 {{ shortId(tunnelId) }}.cfargotunnel.com；需要 Zone · DNS · Edit 权限。</small>
            </span>
          </label>
          <details class="advanced" :open="advancedOpen" @toggle="advancedOpen = ($event.target as HTMLDetailsElement).open">
            <summary>高级选项</summary>
            <div class="form-grid">
              <div class="field" :class="{ 'has-error': pathError }">
                <label for="route-path">路径前缀（可选）</label>
                <input
                  id="route-path"
                  v-model="path"
                  class="input mono"
                  name="path"
                  placeholder="/api"
                  autocomplete="off"
                  spellcheck="false"
                  :disabled="submitting"
                />
                <span class="hint">留空表示匹配整个域名；必须以 / 开头。</span>
                <span class="error">{{ pathError || '路径必须以 / 开头' }}</span>
              </div>
            </div>
          </details>
          <div class="modal-warning">
            <Icon name="info" />
            规则会按顺序写入当前隧道的 ingress；兜底 404 由服务端自动追加。
          </div>
          <div v-if="serverError" class="modal-error" role="alert">
            <Icon name="alert" />
            <span>{{ serverError }}</span>
          </div>
        </div>
        <button type="submit" hidden></button>
      </form>
      <div class="drawer-foot">
        <span class="note">首条命中生效<br />兜底 404 自动保留</span>
        <div class="actions">
          <button class="btn ghost" type="button" :disabled="submitting" @click="emit('close')">取消</button>
          <button class="btn primary" type="button" :disabled="submitting" @click="submit">
            <span v-if="submitting" class="spinner" aria-hidden="true"></span>
            {{ submitting ? '保存中' : mode === 'edit' ? '保存更改' : '保存并应用' }}
          </button>
        </div>
      </div>
    </aside>
  </OverlayLayer>
</template>
