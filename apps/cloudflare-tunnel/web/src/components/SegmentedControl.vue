<script setup lang="ts">
export interface SegmentedOption {
  value: string
  label: string
}

withDefaults(defineProps<{ options: SegmentedOption[]; ariaLabel?: string; disabled?: boolean }>(), {
  ariaLabel: '选项',
  disabled: false,
})

const model = defineModel<string>({ required: true })
</script>

<template>
  <div class="segmented" role="group" :aria-label="ariaLabel">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :class="{ 'is-active': model === option.value }"
      :aria-pressed="model === option.value"
      :disabled="disabled"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>
