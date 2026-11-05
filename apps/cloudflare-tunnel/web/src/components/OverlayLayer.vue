<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { popOverlay, pushOverlay } from '../lib/overlay-stack'

const props = withDefaults(
  defineProps<{
    kind?: 'modal' | 'drawer'
    /** When false the backdrop only blocks clicks (used for progress dialogs). */
    dismissible?: boolean
    initialFocus?: string
  }>(),
  { kind: 'modal', dismissible: true, initialFocus: '' },
)

const emit = defineEmits<{ close: []; opened: [] }>()

const root = ref<HTMLDivElement | null>(null)
const open = ref(false)
let closed = false
let previousFocus: HTMLElement | null = null
let closeTimer = 0

const focusableSelector =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

function focusables(): HTMLElement[] {
  if (!root.value) return []
  return Array.from(root.value.querySelectorAll<HTMLElement>(focusableSelector)).filter(
    (el) => el.offsetParent !== null && el.getAttribute('aria-hidden') !== 'true',
  )
}

function focusFirst() {
  if (!root.value) return
  const preferred = props.initialFocus ? root.value.querySelector<HTMLElement>(props.initialFocus) : null
  const target = preferred || focusables()[0]
  target?.focus()
}

function requestClose() {
  if (closed) return
  if (!props.dismissible) return
  closed = true
  open.value = false
  window.clearTimeout(closeTimer)
  closeTimer = window.setTimeout(() => emit('close'), 300)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    requestClose()
    return
  }
  if (event.key !== 'Tab') return
  const items = focusables()
  if (!items.length) return
  const first = items[0]
  const last = items[items.length - 1]
  const active = document.activeElement as HTMLElement | null
  if (event.shiftKey && (active === first || !root.value?.contains(active))) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && (active === last || !root.value?.contains(active))) {
    event.preventDefault()
    first.focus()
  }
}

onMounted(async () => {
  previousFocus = document.activeElement as HTMLElement | null
  pushOverlay()
  document.addEventListener('keydown', onKeydown, true)
  await nextTick()
  requestAnimationFrame(() => {
    open.value = true
    emit('opened')
  })
  window.setTimeout(focusFirst, 90)
})

onBeforeUnmount(() => {
  window.clearTimeout(closeTimer)
  document.removeEventListener('keydown', onKeydown, true)
  popOverlay()
  if (previousFocus && document.contains(previousFocus)) previousFocus.focus()
})

defineExpose({ requestClose, focusFirst })
</script>

<template>
  <Teleport to="body">
    <div
      ref="root"
      class="layer"
      :class="[kind === 'drawer' ? 'drawer-layer' : '', { open }]"
      @mousedown.self="requestClose"
    >
      <div class="layer-backdrop" @click="requestClose"></div>
      <slot :close="requestClose" />
    </div>
  </Teleport>
</template>
