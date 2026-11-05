<script setup lang="ts">
import Icon from './Icon.vue'

withDefaults(
  defineProps<{
    tone?: 'info' | 'warning' | 'danger'
    icon?: string
    title?: string
  }>(),
  { tone: 'info', icon: 'info', title: '' },
)
</script>

<template>
  <div class="notice-bar" :class="tone" role="status">
    <Icon :name="icon" />
    <div class="notice-copy">
      <strong v-if="title">{{ title }}</strong>
      <slot />
    </div>
    <slot name="action" />
  </div>
</template>
