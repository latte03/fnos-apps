<script setup lang="ts">
import { computed, useId } from 'vue'

const props = withDefaults(
  defineProps<{
    live: boolean
    protocol?: string
    edgeLocation?: string
    connections?: number
    latencyMs?: number | null
    hostName?: string
    hostAddress?: string
  }>(),
  {
    protocol: 'quic',
    edgeLocation: '',
    connections: 0,
    latencyMs: null,
    hostName: '本机',
    hostAddress: '',
  },
)

const uid = useId().replace(/[^a-zA-Z0-9_-]/g, '')
const dotsId = `spineDots-${uid}`
const lineId = `spineLine-${uid}`

const badgeLabel = computed(() => {
  const parts: string[] = []
  if (props.latencyMs !== null && props.latencyMs !== undefined) parts.push(`${props.latencyMs} ms`)
  if (props.edgeLocation) parts.push(props.edgeLocation)
  return parts.join(' · ') || '—'
})

const edgeSub = computed(() => {
  const parts: string[] = []
  if (props.edgeLocation) parts.push(props.edgeLocation)
  parts.push(`${props.connections} connections`)
  return parts.join(' · ')
})
</script>

<template>
  <svg
    class="spine"
    :class="{ 'is-idle': !live }"
    viewBox="0 0 560 320"
    fill="none"
    role="img"
    aria-label="NAS 通过 Cloudflare Edge 连接到互联网"
  >
    <defs>
      <pattern :id="dotsId" width="18" height="18" patternUnits="userSpaceOnUse">
        <circle cx="1" cy="1" r="1" fill="var(--line-strong)" />
      </pattern>
      <linearGradient :id="lineId" x1="120" y1="160" x2="440" y2="160" gradientUnits="userSpaceOnUse">
        <stop stop-color="var(--accent)" stop-opacity=".55" />
        <stop offset=".5" stop-color="var(--teal)" stop-opacity=".8" />
        <stop offset="1" stop-color="var(--accent)" stop-opacity=".55" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" width="560" height="320" :fill="`url(#${dotsId})`" opacity=".55" />
    <path d="M126 160 L 434 160" :stroke="`url(#${lineId})`" stroke-width="1.5" stroke-dasharray="5 7" opacity=".8" />
    <path
      d="M126 160 C 190 104, 210 104, 280 104 S 370 104, 434 160"
      stroke="var(--line-strong)"
      stroke-width="1"
      stroke-dasharray="2 8"
      opacity=".55"
    />
    <path
      d="M126 160 C 190 216, 210 216, 280 216 S 370 216, 434 160"
      stroke="var(--line-strong)"
      stroke-width="1"
      stroke-dasharray="2 8"
      opacity=".55"
    />

    <g class="node node-nas" transform="translate(92 160)">
      <circle r="48" class="node-ring" />
      <circle r="34" class="node-core" />
      <g transform="translate(-11 -11) scale(0.92)" class="node-icon">
        <rect x="3" y="4" width="18" height="7" rx="2" />
        <rect x="3" y="13" width="18" height="7" rx="2" />
        <path d="M7 7.5h.01M7 16.5h.01" />
      </g>
    </g>
    <g class="node node-edge" transform="translate(280 160)">
      <circle r="58" class="node-ring" opacity=".45" />
      <circle r="40" class="node-core" />
      <g transform="translate(-11 -11) scale(0.94)" class="node-icon">
        <path d="M17.5 19a4.5 4.5 0 0 0 .4-8.98A7 7 0 0 0 4.2 12.5 4 4 0 0 0 6 20h11.5Z" />
      </g>
      <circle cx="0" cy="0" r="47" fill="none" stroke="var(--accent)" stroke-width="1" stroke-dasharray="2 7" opacity=".3" />
    </g>
    <g class="node node-cloud" transform="translate(468 160)">
      <circle r="48" class="node-ring" />
      <circle r="34" class="node-core" />
      <g transform="translate(-11 -11) scale(0.92)" class="node-icon">
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18" />
        <path d="M12 3a15 15 0 0 1 0 18" />
        <path d="M12 3a15 15 0 0 0 0 18" />
      </g>
    </g>

    <g class="latency-badge" transform="translate(244 62)">
      <rect width="72" height="24" rx="12" />
      <text x="36" y="16">{{ badgeLabel }}</text>
    </g>

    <text x="92" y="226" class="node-label">{{ hostName }}</text>
    <text x="92" y="240" class="node-sub">{{ hostAddress }}</text>
    <text x="280" y="238" class="node-label">Cloudflare Edge</text>
    <text x="280" y="252" class="node-sub">{{ edgeSub }}</text>
    <text x="468" y="226" class="node-label">Internet</text>
    <text x="468" y="240" class="node-sub">{{ protocol.toUpperCase() }} · TLS 1.3</text>

    <circle class="packet" r="3.2">
      <animateMotion dur="3.6s" repeatCount="indefinite" path="M126 160 L 434 160" />
    </circle>
    <circle class="packet" r="2.4" opacity=".7">
      <animateMotion dur="3.6s" begin="1.8s" repeatCount="indefinite" path="M126 160 L 434 160" />
    </circle>
  </svg>
</template>
