<script setup lang="ts">
const model = defineModel<boolean>({ required: true })
defineProps<{ label?: string; disabled?: boolean }>()
</script>

<template>
  <span class="switch">
    <input
      v-model="model"
      type="checkbox"
      :aria-label="label"
      :disabled="disabled"
      role="switch"
      :aria-checked="model"
    />
    <i aria-hidden="true"></i>
  </span>
</template>
