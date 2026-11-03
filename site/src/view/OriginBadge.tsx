import type { Origin } from '../types'

/** Where a TW text came from, as colour and as text (the colour is not the only sign). */
export function OriginBadge({ origin }: { origin: Origin }) {
  return (
    <span class={`origin origin-${origin}`} lang="en">
      {origin}
    </span>
  )
}
