<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    kind?: 'healthy' | 'warning' | 'danger' | 'neutral' | 'info'
    dot?: boolean
    spinner?: boolean
  }>(),
  { kind: 'neutral', dot: false, spinner: false },
)
</script>

<template>
  <span class="pill" :class="kind">
    <span v-if="spinner" class="spinner pill-spinner" aria-hidden="true"></span>
    <span v-else-if="dot" class="dot" aria-hidden="true"></span>
    {{ label }}
  </span>
</template>
