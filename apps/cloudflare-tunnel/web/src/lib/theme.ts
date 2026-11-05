import { computed, ref, watch } from 'vue'

export type ThemePref = 'system' | 'light' | 'dark'

const THEME_KEY = 'tunnel-theme'
const MOTION_KEY = 'tunnel-reduce-motion'

function readThemePref(): ThemePref {
  try {
    const stored = localStorage.getItem(THEME_KEY)
    if (stored === 'light' || stored === 'dark' || stored === 'system') return stored
  } catch {
    /* storage disabled */
  }
  return 'system'
}

function readMotion(): boolean {
  try {
    return localStorage.getItem(MOTION_KEY) === '1'
  } catch {
    return false
  }
}

const media = window.matchMedia('(prefers-color-scheme: dark)')

export const themePref = ref<ThemePref>(readThemePref())
export const systemDark = ref(media.matches)
export const reduceMotion = ref(readMotion())
export const resolvedTheme = computed<'light' | 'dark'>(() =>
  themePref.value === 'system' ? (systemDark.value ? 'dark' : 'light') : themePref.value,
)

function persist(key: string, value: string | null) {
  try {
    if (value === null) localStorage.removeItem(key)
    else localStorage.setItem(key, value)
  } catch {
    /* storage disabled */
  }
}

function apply() {
  document.documentElement.dataset.theme = resolvedTheme.value
  document.documentElement.classList.toggle('reduce-motion', reduceMotion.value)
}

media.addEventListener('change', (event) => {
  systemDark.value = event.matches
})

watch([resolvedTheme, reduceMotion], apply)
apply()

export function setThemePref(pref: ThemePref) {
  themePref.value = pref
  persist(THEME_KEY, pref)
}

/** Sidebar toggle: flip to the opposite of what is currently visible. */
export function toggleTheme() {
  setThemePref(resolvedTheme.value === 'dark' ? 'light' : 'dark')
}

export function setReduceMotion(value: boolean) {
  reduceMotion.value = value
  persist(MOTION_KEY, value ? '1' : null)
}
