let depth = 0

/** Locks body scroll while at least one overlay is mounted. */
export function pushOverlay() {
  depth += 1
  if (depth === 1) document.body.style.overflow = 'hidden'
}

export function popOverlay() {
  depth = Math.max(0, depth - 1)
  if (depth === 0) document.body.style.overflow = ''
}
