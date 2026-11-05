import { reactive } from 'vue'

export interface ConfirmRequest {
  title: string
  message?: string
  confirmText?: string
  cancelText?: string
  tone?: 'primary' | 'danger'
  checkbox?: string
  checkboxHint?: string
  /**
   * Runs while the dialog stays open. On rejection the dialog shows the error
   * inline instead of closing, so failures never look like success.
   */
  action?: (checked: boolean) => Promise<void>
}

export interface ConfirmState {
  open: boolean
  request: ConfirmRequest | null
  error: string
  pending: boolean
}

export const confirmState = reactive<ConfirmState>({
  open: false,
  request: null,
  error: '',
  pending: false,
})

let resolver: ((result: { ok: boolean; checked: boolean }) => void) | null = null

export function confirmDialog(request: ConfirmRequest): Promise<{ ok: boolean; checked: boolean }> {
  if (resolver) {
    resolver({ ok: false, checked: false })
    resolver = null
  }
  confirmState.request = request
  confirmState.error = ''
  confirmState.pending = false
  confirmState.open = true
  return new Promise((resolve) => {
    resolver = resolve
  })
}

export function closeConfirm(ok: boolean, checked = false) {
  confirmState.open = false
  confirmState.pending = false
  confirmState.error = ''
  const resolve = resolver
  resolver = null
  confirmState.request = null
  resolve?.({ ok, checked })
}
