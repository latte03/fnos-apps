<script setup lang="ts">
withDefaults(defineProps<{ height?: string; width?: string; radius?: string }>(), {
  height: '16px',
  width: '100%',
  radius: '6px',
})
</script>

<template>
  <span class="skeleton" :style="{ height, width, borderRadius: radius }" aria-hidden="true" />
</template>
