<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import OverlayLayer from './OverlayLayer.vue'
import { copyText } from '../lib/clipboard'

const props = withDefaults(
  defineProps<{
    eyebrow?: string
    title: string
    secret: string
    warning?: string
    note?: string
    confirmLabel?: string
  }>(),
  {
    eyebrow: '凭证',
    warning: 'Token 只显示一次。关闭后只能吊销并重新生成。',
    note: '',
    confirmLabel: '我已保存',
  },
)

const emit = defineEmits<{ close: [] }>()

const revealed = ref(false)
const masked = computed(() => '•'.repeat(Math.min(props.secret.length, 48)))

function copy() {
  void copyText(props.secret, 'Token 已复制')
}
</script>

<template>
  <OverlayLayer initial-focus="[data-reveal]" @close="emit('close')">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="secret-modal-title">
      <div class="modal-head">
        <div>
          <p class="eyebrow">{{ eyebrow }}</p>
          <h2 id="secret-modal-title">{{ title }}</h2>
        </div>
        <button class="icon-btn" type="button" aria-label="关闭" @click="emit('close')">
          <Icon name="x" />
        </button>
      </div>
      <div class="modal-body">
        <div class="modal-warning">
          <Icon name="shield" />
          {{ warning }}
        </div>
        <div class="endpoint-row secret-row">
          <span class="mono secret-value">{{ revealed ? secret : masked }}</span>
          <button class="btn small ghost" type="button" data-reveal @click="revealed = !revealed">
            <Icon :name="revealed ? 'eye-off' : 'eye'" />
            {{ revealed ? '隐藏' : '显示' }}
          </button>
          <button class="btn small primary" type="button" @click="copy">
            <Icon name="copy" />
            复制
          </button>
        </div>
        <p v-if="note" class="secret-note">{{ note }}</p>
        <p class="secret-note">
          客户端在请求头中携带：<span class="mono">Authorization: Bearer &lt;token&gt;</span>
        </p>
      </div>
      <div class="modal-foot">
        <button class="btn primary" type="button" @click="emit('close')">{{ confirmLabel }}</button>
      </div>
    </div>
  </OverlayLayer>
</template>
