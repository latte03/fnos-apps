<script setup lang="ts">
defineProps<{ eyebrow: string; title: string; sub?: string }>()
</script>

<template>
  <section class="page-head reveal" style="--i: 0">
    <div>
      <p class="eyebrow">{{ eyebrow }}</p>
      <h1>{{ title }}</h1>
      <p v-if="sub" class="sub">{{ sub }}</p>
    </div>
    <div v-if="$slots.actions" class="page-actions">
      <slot name="actions" />
    </div>
  </section>
</template>
