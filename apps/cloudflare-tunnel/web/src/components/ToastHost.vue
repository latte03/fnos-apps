<script setup lang="ts">
import Icon from './Icon.vue'
import { dismissToast, toasts } from '../lib/toast'
</script>

<template>
  <Teleport to="body">
    <div id="toast-root" class="toast-stack" role="status" aria-live="polite">
      <TransitionGroup name="toast">
        <div
          v-for="item in toasts"
          :key="item.id"
          class="toast is-in"
          :class="{ error: item.tone === 'error' }"
        >
          <Icon :name="item.tone === 'error' ? 'alert' : 'check'" />
          <div class="toast-copy">{{ item.message }}</div>
          <button class="toast-close" type="button" aria-label="关闭提示" @click="dismissToast(item.id)">
            <Icon name="x" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
