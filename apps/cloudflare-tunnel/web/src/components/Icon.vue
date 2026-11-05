<script setup lang="ts">
import { computed } from 'vue'
import { iconPaths } from '../lib/icons'

const props = defineProps<{ name: string }>()
const path = computed(() => iconPaths[props.name] || iconPaths.info)
</script>

<template>
  <svg class="icon" viewBox="0 0 24 24" aria-hidden="true" v-html="path" />
</template>
