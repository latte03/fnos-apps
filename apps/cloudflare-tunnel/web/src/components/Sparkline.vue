<script setup lang="ts">
import { computed, useId } from 'vue'

const props = withDefaults(
  defineProps<{ values: number[]; green?: boolean; width?: number; height?: number }>(),
  { green: false, width: 84, height: 28 },
)

const gradientId = `spark-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`

const geometry = computed(() => {
  const values = props.values.filter((value) => Number.isFinite(value))
  if (values.length < 2) return null
  const w = props.width
  const h = props.height
  const min = Math.min(...values)
  const max = Math.max(...values)
  const span = max - min || 1
  const points = values.map((value, index) => {
    const x = (index / (values.length - 1)) * w
    const y = h - 4 - ((value - min) / span) * (h - 8)
    return [x, y] as const
  })
  const line = points.map((point, index) => `${index ? 'L' : 'M'}${point[0].toFixed(1)} ${point[1].toFixed(1)}`).join(' ')
  return { line, area: `${line} L ${w} ${h} L 0 ${h} Z`, w, h }
})
</script>

<template>
  <svg
    v-if="geometry"
    class="metric-spark"
    :class="{ green }"
    :viewBox="`0 0 ${geometry.w} ${geometry.h}`"
    aria-hidden="true"
  >
    <defs>
      <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
        <stop :stop-color="green ? 'var(--teal)' : 'var(--accent)'" stop-opacity=".25" />
        <stop offset="1" :stop-color="green ? 'var(--teal)' : 'var(--accent)'" stop-opacity="0" />
      </linearGradient>
    </defs>
    <path :d="geometry.area" :fill="`url(#${gradientId})`" />
    <path :d="geometry.line" class="line" />
  </svg>
</template>
