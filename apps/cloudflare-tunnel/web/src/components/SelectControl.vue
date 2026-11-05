<script setup lang="ts" generic="T extends string | number">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import Icon from './Icon.vue'

interface SelectOption<V extends string | number> {
  value: V
  label: string
}

const props = withDefaults(
  defineProps<{
    options: SelectOption<T>[]
    ariaLabel?: string
    disabled?: boolean
  }>(),
  { ariaLabel: '选项', disabled: false },
)

const model = defineModel<T>({ required: true })

const uid = useId()
const listId = `${uid}-list`
const open = ref(false)
const activeIndex = ref(-1)
const trigger = ref<HTMLButtonElement | null>(null)
const menu = ref<HTMLElement | null>(null)
const menuStyle = ref<Record<string, string>>({})

const selectedIndex = computed(() => props.options.findIndex((option) => option.value === model.value))
const selectedLabel = computed(() => props.options[selectedIndex.value]?.label ?? '')

function optionId(index: number) {
  return `${listId}-${index}`
}

function placeMenu() {
  const el = trigger.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const width = Math.max(rect.width, 168)
  const height = Math.min(menu.value?.offsetHeight || props.options.length * 36 + 8, 320)
  const below = window.innerHeight - rect.bottom
  const openUp = below < height + 12 && rect.top > below
  const top = openUp ? Math.max(12, rect.top - height - 6) : rect.bottom + 6
  const left = Math.max(12, Math.min(rect.left, window.innerWidth - width - 12))
  menuStyle.value = {
    top: `${top}px`,
    left: `${left}px`,
    width: `${width}px`,
    maxHeight: `${Math.min(320, window.innerHeight - 24)}px`,
    transformOrigin: openUp ? 'bottom left' : 'top left',
    '--select-shift': openUp ? '4px' : '-4px',
  }
}

function onDocumentPointerDown(event: PointerEvent) {
  const target = event.target as Node
  if (trigger.value?.contains(target) || menu.value?.contains(target)) return
  closeMenu()
}

function onScroll(event: Event) {
  if (menu.value?.contains(event.target as Node)) return
  closeMenu()
}

async function openMenu() {
  if (props.disabled || open.value) return
  open.value = true
  activeIndex.value = selectedIndex.value >= 0 ? selectedIndex.value : 0
  placeMenu()
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
  window.addEventListener('resize', placeMenu)
  window.addEventListener('scroll', onScroll, true)
  await nextTick()
  placeMenu()
}

function closeMenu() {
  if (!open.value) return
  open.value = false
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  window.removeEventListener('resize', placeMenu)
  window.removeEventListener('scroll', onScroll, true)
}

function toggle() {
  if (open.value) closeMenu()
  else void openMenu()
}

function choose(index: number) {
  const option = props.options[index]
  if (!option) return
  model.value = option.value
  closeMenu()
  trigger.value?.focus()
}

function onKeydown(event: KeyboardEvent) {
  if (props.disabled) return
  const count = props.options.length
  if (!count) return
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (!open.value) void openMenu()
      else activeIndex.value = (activeIndex.value + 1) % count
      break
    case 'ArrowUp':
      event.preventDefault()
      if (!open.value) {
        void openMenu()
        activeIndex.value = count - 1
      } else {
        activeIndex.value = (activeIndex.value - 1 + count) % count
      }
      break
    case 'Home':
      if (!open.value) return
      event.preventDefault()
      activeIndex.value = 0
      break
    case 'End':
      if (!open.value) return
      event.preventDefault()
      activeIndex.value = count - 1
      break
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (open.value) choose(activeIndex.value)
      else void openMenu()
      break
    case 'Escape':
      if (open.value) {
        event.preventDefault()
        closeMenu()
      }
      break
    case 'Tab':
      closeMenu()
      break
  }
}

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) closeMenu()
  },
)

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  window.removeEventListener('resize', placeMenu)
  window.removeEventListener('scroll', onScroll, true)
})
</script>

<template>
  <div class="select-control">
    <button
      ref="trigger"
      class="select-trigger"
      :class="{ 'is-open': open }"
      type="button"
      role="combobox"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-controls="listId"
      :aria-activedescendant="open && activeIndex >= 0 ? optionId(activeIndex) : undefined"
      :aria-label="ariaLabel"
      :disabled="disabled"
      @click="toggle"
      @keydown="onKeydown"
    >
      <span class="select-value">{{ selectedLabel }}</span>
      <Icon name="chevron-down" class="select-caret" />
    </button>

    <Teleport to="body">
      <Transition name="select-pop">
        <div
          v-if="open"
          :id="listId"
          ref="menu"
          class="select-menu"
          role="listbox"
          :aria-label="ariaLabel"
          :style="menuStyle"
        >
          <button
            v-for="(option, index) in options"
            :id="optionId(index)"
            :key="option.value"
            class="select-option"
            :class="{ 'is-active': index === activeIndex, 'is-selected': option.value === model }"
            type="button"
            role="option"
            :aria-selected="option.value === model"
            tabindex="-1"
            @mouseenter="activeIndex = index"
            @click="choose(index)"
          >
            <span class="select-option-label">{{ option.label }}</span>
            <Icon v-if="option.value === model" name="check" class="select-check" />
          </button>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>
