<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import OverlayLayer from './OverlayLayer.vue'
import { apiErrorMessage } from '../lib/api'
import { closeConfirm, confirmState } from '../lib/dialog'

const checkbox = ref(true)
const overlay = ref<InstanceType<typeof OverlayLayer> | null>(null)

const title = computed(() => confirmState.request?.title ?? '')
const tone = computed(() => confirmState.request?.tone ?? 'primary')

async function submit() {
  const request = confirmState.request
  if (!request || confirmState.pending) return
  if (!request.action) {
    closeConfirm(true, checkbox.value)
    return
  }
  confirmState.pending = true
  confirmState.error = ''
  try {
    await request.action(checkbox.value)
    closeConfirm(true, checkbox.value)
  } catch (err) {
    confirmState.error = apiErrorMessage(err)
  } finally {
    confirmState.pending = false
  }
}

function cancel() {
  if (confirmState.pending) return
  closeConfirm(false, checkbox.value)
}
</script>

<template>
  <OverlayLayer
    v-if="confirmState.open"
    ref="overlay"
    initial-focus="[data-confirm]"
    :dismissible="!confirmState.pending"
    @close="cancel"
    @opened="checkbox = true"
  >
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="confirm-dialog-title">
      <div class="modal-head">
        <div>
          <p class="eyebrow">确认操作</p>
          <h2 id="confirm-dialog-title">{{ title }}</h2>
        </div>
        <button class="icon-btn" type="button" aria-label="关闭" :disabled="confirmState.pending" @click="cancel">
          <Icon name="x" />
        </button>
      </div>
      <div class="modal-body">
        <p v-if="confirmState.request?.message">{{ confirmState.request.message }}</p>
        <label v-if="confirmState.request?.checkbox" class="check-row">
          <span class="switch">
            <input v-model="checkbox" type="checkbox" :disabled="confirmState.pending" />
            <i aria-hidden="true"></i>
          </span>
          <span class="check-copy">
            <strong>{{ confirmState.request.checkbox }}</strong>
            <small>{{ confirmState.request.checkboxHint || '删除后无法恢复；也可以在 Cloudflare 控制台手动处理。' }}</small>
          </span>
        </label>
        <div v-if="confirmState.error" class="modal-error" role="alert">
          <Icon name="alert" />
          <span>{{ confirmState.error }}</span>
        </div>
      </div>
      <div class="modal-foot">
        <button class="btn ghost" type="button" :disabled="confirmState.pending" @click="cancel">
          {{ confirmState.request?.cancelText || '取消' }}
        </button>
        <button
          class="btn"
          :class="tone === 'danger' ? 'danger' : 'primary'"
          type="button"
          data-confirm
          :disabled="confirmState.pending"
          @click="submit"
        >
          <span v-if="confirmState.pending" class="spinner" aria-hidden="true"></span>
          {{ confirmState.pending ? '处理中' : confirmState.request?.confirmText || '确定' }}
        </button>
      </div>
    </div>
  </OverlayLayer>
</template>
