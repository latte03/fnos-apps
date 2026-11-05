<script setup lang="ts">
import { nextTick, ref } from 'vue'
import Icon from './Icon.vue'
import OverlayLayer from './OverlayLayer.vue'

const props = defineProps<{ submitting: boolean; serverError: string }>()
const emit = defineEmits<{ close: []; submit: [name: string] }>()

const name = ref('')
const nameError = ref('')
const input = ref<HTMLInputElement | null>(null)

function submit() {
  if (props.submitting) return
  if (!name.value.trim()) {
    nameError.value = '请输入凭证名称'
    input.value?.focus()
    return
  }
  emit('submit', name.value.trim())
}

function onOpened() {
  void nextTick(() => input.value?.focus())
}
</script>

<template>
  <OverlayLayer initial-focus="#credential-name" @close="emit('close')" @opened="onOpened">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="credential-title">
      <div class="modal-head">
        <div>
          <p class="eyebrow">API</p>
          <h2 id="credential-title">新建凭证</h2>
        </div>
        <button class="icon-btn" type="button" aria-label="关闭" :disabled="submitting" @click="emit('close')">
          <Icon name="x" />
        </button>
      </div>
      <form class="modal-body" @submit.prevent="submit">
        <div class="form-grid">
          <div class="field" :class="{ 'has-error': nameError }">
            <label for="credential-name">名称</label>
            <input
              id="credential-name"
              ref="input"
              v-model="name"
              class="input"
              placeholder="Home Assistant"
              autocomplete="off"
              :disabled="submitting"
              @input="nameError = ''"
            />
            <span class="hint">用应用或用途命名，方便以后吊销。</span>
            <span class="error">{{ nameError || '请输入凭证名称' }}</span>
          </div>
          <div class="modal-warning">
            <Icon name="shield" />
            凭证以 Bearer Token 调用本机 /api/v1 接口；服务端只保存 SHA-256 哈希，明文只显示一次。
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
          {{ submitting ? '生成中' : '生成凭证' }}
        </button>
      </div>
    </div>
  </OverlayLayer>
</template>
