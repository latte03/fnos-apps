<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { formatClock } from '../lib/format'
import type { RuntimeMetrics } from '../types'

const props = defineProps<{ samples: RuntimeMetrics[] }>()

const wrap = ref<HTMLElement | null>(null)
const width = ref(640)
const height = 186
const left = 46
const right = 14
const top = 12
const bottom = 24

let observer: ResizeObserver | null = null

onMounted(() => {
  if (!wrap.value) return
  observer = new ResizeObserver((entries) => {
    width.value = Math.max(280, Math.round(entries[0].contentRect.width))
  })
  observer.observe(wrap.value)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
})

function niceMax(value: number): number {
  if (value <= 4) return 4
  const exponent = 10 ** Math.floor(Math.log10(value))
  const ratio = value / exponent
  const step = ratio <= 1 ? 1 : ratio <= 2 ? 2 : ratio <= 5 ? 5 : 10
  return step * exponent
}

function tickLabel(value: number): string {
  if (value >= 10000) return `${Math.round(value / 1000)}k`
  if (value >= 1000) return `${(value / 1000).toFixed(value >= 2000 ? 0 : 1)}k`
  return String(Math.round(value))
}

const geometry = computed(() => {
  const samples = props.samples
  if (samples.length < 2) return null

  const points: { at: number; value: number; errors: number }[] = []
  for (let index = 1; index < samples.length; index += 1) {
    const previous = samples[index - 1]
    const current = samples[index]
    if (previous.requests === null || current.requests === null) continue
    const at = Date.parse(current.sampledAt)
    if (Number.isNaN(at)) continue
    const errors =
      previous.errors !== null && current.errors !== null
        ? Math.max(0, current.errors - previous.errors)
        : 0
    points.push({ at, value: Math.max(0, current.requests - previous.requests), errors })
  }
  if (points.length < 2) return null

  const plotWidth = width.value - left - right
  const plotHeight = height - top - bottom
  const firstAt = points[0].at
  const lastAt = points[points.length - 1].at
  const span = lastAt - firstAt
  const xOf = (at: number, index: number) =>
    span > 0 ? left + ((at - firstAt) / span) * plotWidth : left + (index / (points.length - 1)) * plotWidth
  const peak = Math.max(...points.map((point) => point.value))
  const peakErrors = Math.max(...points.map((point) => point.errors))
  const max = niceMax(Math.max(peak, 1))
  const yOf = (value: number) => top + plotHeight - (value / max) * plotHeight

  const coords = points.map((point, index) => ({
    x: xOf(point.at, index),
    y: yOf(point.value),
    errorY: yOf(point.errors),
  }))
  const line = coords.map((point, index) => `${index ? 'L' : 'M'}${point.x.toFixed(1)} ${point.y.toFixed(1)}`).join(' ')
  const area = `${line} L ${coords[coords.length - 1].x.toFixed(1)} ${top + plotHeight} L ${coords[0].x.toFixed(1)} ${top + plotHeight} Z`
  const errorLine = peakErrors > 0
    ? coords.map((point, index) => `${index ? 'L' : 'M'}${point.x.toFixed(1)} ${point.errorY.toFixed(1)}`).join(' ')
    : ''

  const yTicks = [0, max / 2, max].map((value) => ({
    value,
    y: yOf(value),
    label: tickLabel(value),
  }))

  const xTicks = [
    { x: left, at: firstAt, anchor: 'start' as const },
    { x: left + plotWidth / 2, at: (firstAt + lastAt) / 2, anchor: 'middle' as const },
    { x: left + plotWidth, at: lastAt, anchor: 'end' as const },
  ]
    .filter((tick) => width.value >= 460 || tick.anchor !== 'middle')
    .map((tick) => ({ x: tick.x, label: formatClock(tick.at), anchor: tick.anchor }))

  return {
    line,
    area,
    errorLine,
    hasErrors: peakErrors > 0,
    yTicks,
    xTicks,
    last: coords[coords.length - 1],
    peak,
    peakErrors,
  }
})

const ariaLabel = computed(() => {
  if (!geometry.value) return '请求趋势，等待更多采样'
  return `请求趋势，峰值为每采样 ${geometry.value.peak} 次请求`
})
</script>

<template>
  <div ref="wrap" class="traffic-plot">
    <svg
      v-if="geometry"
      class="traffic-svg"
      :width="width"
      :height="height"
      :viewBox="`0 0 ${width} ${height}`"
      role="img"
      :aria-label="ariaLabel"
    >
      <g class="chart-grid">
        <line v-for="tick in geometry.yTicks" :key="`grid-${tick.value}`" :x1="left" :x2="width - right" :y1="tick.y" :y2="tick.y" />
      </g>
      <g class="chart-axis">
        <text v-for="tick in geometry.yTicks" :key="`y-${tick.value}`" :x="left - 8" :y="tick.y + 3.5" text-anchor="end">
          {{ tick.label }}
        </text>
        <text v-for="tick in geometry.xTicks" :key="`x-${tick.anchor}`" :x="tick.x" :y="height - 6" :text-anchor="tick.anchor">
          {{ tick.label }}
        </text>
      </g>
      <path class="chart-area" :d="geometry.area" />
      <path v-if="geometry.errorLine" class="chart-line is-error" :d="geometry.errorLine" />
      <path class="chart-line" :d="geometry.line" />
      <circle class="chart-dot" :cx="geometry.last.x" :cy="geometry.last.y" r="3" />
    </svg>
    <p v-else class="traffic-empty">等待更多 metrics 样本…</p>
    <div class="chart-legend">
      <span class="legend-key requests"><i></i>请求 / 采样</span>
      <span v-if="geometry?.hasErrors" class="legend-key errors"><i></i>错误 / 采样</span>
      <span class="chart-meta">{{ Math.max(0, samples.length - 1) }} 个采样间隔</span>
    </div>
  </div>
</template>
