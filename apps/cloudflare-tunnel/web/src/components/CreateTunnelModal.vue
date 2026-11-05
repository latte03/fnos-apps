<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import Icon from './Icon.vue'
import OverlayLayer from './OverlayLayer.vue'
import SwitchControl from './SwitchControl.vue'

const props = defineProps<{ submitting: boolean; serverError: string }>()
const emit = defineEmits<{ close: []; submit: [payload: { name: string; autoConnect: boolean }] }>()

const name = ref('')
const autoConnect = ref(true)
const nameError = ref('')
const input = ref<HTMLInputElement | null>(null)

watch(
  () => name.value,
  (value) => {
    const clean = value.toLowerCase().replace(/[^a-z0-9-]/g, '')
    if (clean !== value) name.value = clean
    nameError.value = ''
  },
)

function validate(): boolean {
  const trimmed = name.value.trim()
  if (!/^[a-z0-9][a-z0-9-]{0,62}$/.test(trimmed)) {
    nameError.value = '请输入 1–63 位小写字母、数字或短横线，且以字母或数字开头'
    return false
  }
  return true
}

function submit() {
  if (props.submitting) return
  if (!validate()) {
    input.value?.focus()
    return
  }
  emit('submit', { name: name.value.trim(), autoConnect: autoConnect.value })
}

function onOpened() {
  void nextTick(() => input.value?.focus())
}

function onInput() {
  if (nameError.value) nameError.value = ''
}
</script>

<template>
  <OverlayLayer initial-focus="#tunnel-name" @close="emit('close')" @opened="onOpened">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="new-tunnel-title">
      <div class="modal-head">
        <div>
          <p class="eyebrow">新建</p>
          <h2 id="new-tunnel-title">创建隧道</h2>
        </div>
        <button class="icon-btn" type="button" aria-label="关闭" :disabled="submitting" @click="emit('close')">
          <Icon name="x" />
        </button>
      </div>
      <form class="modal-body" @submit.prevent="submit">
        <div class="form-grid">
          <div class="field" :class="{ 'has-error': nameError }">
            <label for="tunnel-name">隧道名称</label>
            <input
              id="tunnel-name"
              ref="input"
              v-model="name"
              class="input mono"
              name="name"
              placeholder="fnos-home"
              autocomplete="off"
              spellcheck="false"
              :disabled="submitting"
              @input="onInput"
              @blur="validate"
            />
            <span class="hint">只用于识别，不影响域名。支持小写字母、数字和短横线。</span>
            <span class="error">{{ nameError || '请输入 1–63 位小写名称，例如 fnos-home' }}</span>
          </div>
          <label class="check-row">
            <SwitchControl v-model="autoConnect" label="创建后立即连接" />
            <span class="check-copy">
              <strong>创建后立即连接</strong>
              <small>创建成功后会接管当前连接；已有隧道会先断开。</small>
            </span>
          </label>
          <div class="modal-warning">
            <Icon name="info" />
            隧道会通过 Cloudflare API 创建在当前账户下，需要 Account · Cloudflare Tunnel · Edit 权限。
          </div>
          <div v-if="serverError" class="modal-error" role="alert">
            <Icon name="alert" />
            <span>{{ serverError }}</span>
          </div>
        </div>
        <button type="submit" hidden></button>
      </form>
      <div class="modal-foot">
        <button class="btn ghost" type="button" :disabled="submitting" @click="emit('close')">取消</button>
        <button class="btn primary" type="button" :disabled="submitting" @click="submit">
          <span v-if="submitting" class="spinner" aria-hidden="true"></span>
          {{ submitting ? '创建中' : '创建隧道' }}
        </button>
      </div>
    </div>
  </OverlayLayer>
</template>
