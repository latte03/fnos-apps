<script setup lang="ts">
import Icon from './Icon.vue'

withDefaults(defineProps<{ icon?: string; title: string; message?: string; big?: boolean }>(), {
  icon: 'info',
  message: '',
  big: false,
})
</script>

<template>
  <div class="table-empty" :class="{ 'is-big': big }">
    <span v-if="icon" class="empty-icon"><Icon :name="icon" /></span>
    <div class="empty-title">{{ title }}</div>
    <p v-if="message" class="empty-message">{{ message }}</p>
    <div v-if="$slots.default" class="empty-actions">
      <slot />
    </div>
  </div>
</template>
