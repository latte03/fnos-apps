import { reactive } from 'vue'

export type ToastTone = 'success' | 'error'

export interface ToastItem {
  id: number
  message: string
  tone: ToastTone
}

let counter = 0
const timers = new Map<number, number>()

export const toasts = reactive<ToastItem[]>([])

export function dismissToast(id: number) {
  const index = toasts.findIndex((t) => t.id === id)
  if (index >= 0) toasts.splice(index, 1)
  const timer = timers.get(id)
  if (timer !== undefined) window.clearTimeout(timer)
  timers.delete(id)
}

export function toast(message: string, tone: ToastTone = 'success') {
  const id = ++counter
  toasts.push({ id, message, tone })
  if (toasts.length > 3) dismissToast(toasts[0].id)
  timers.set(
    id,
    window.setTimeout(() => dismissToast(id), 3300),
  )
  return id
}
