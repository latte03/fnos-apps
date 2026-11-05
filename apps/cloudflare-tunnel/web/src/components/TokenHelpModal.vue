<script setup lang="ts">
import Icon from './Icon.vue'
import OverlayLayer from './OverlayLayer.vue'

const emit = defineEmits<{ close: [] }>()

const permissions: Array<[string, string, string]> = [
  ['账户 · Cloudflare Tunnel', 'Read + Edit', '创建 / 连接 / 删除隧道，读写 ingress'],
  ['区域 · DNS', 'Read + Edit', '创建与更新 CNAME 记录'],
  ['区域 · Zone', 'Read', '列出域名与 Zone'],
]
</script>

<template>
  <OverlayLayer @close="emit('close')">
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="token-help-title">
      <div class="modal-head">
        <div>
          <p class="eyebrow">Cloudflare</p>
          <h2 id="token-help-title">最小权限 Token</h2>
        </div>
        <button class="icon-btn" type="button" aria-label="关闭" @click="emit('close')">
          <Icon name="x" />
        </button>
      </div>
      <div class="modal-body">
        <p>在 Cloudflare Dashboard → My Profile → API Tokens → Create Custom Token 中创建：</p>
        <div class="form-grid" style="gap: 8px">
          <div v-for="[scope, perm, why] in permissions" :key="scope" class="check-row" style="align-items: center">
            <span class="credential-icon"><Icon name="shield" /></span>
            <span class="check-copy">
              <strong>{{ scope }}</strong>
              <small>{{ why }}</small>
            </span>
            <span class="pill info" style="margin-left: auto">{{ perm }}</span>
          </div>
        </div>
        <p style="margin-top: 14px">不需要 Global API Key，也不要把 Token 写进 URL 或日志。</p>
      </div>
      <div class="modal-foot">
        <button class="btn primary" type="button" @click="emit('close')">知道了</button>
      </div>
    </div>
  </OverlayLayer>
</template>
