import { createContext } from 'preact'
import { useContext } from 'preact/hooks'
import type { UpdateEntry, UpdateEntryCause } from '../types'
import { CAUSE_KIND_NAMES, causeKey } from './causes'

/**
 * Where a cause chip leads. `href` gives the hash of the by-cause view with a
 * group open (empty: the chip is not a link); `current` is the group whose
 * cards are drawn, whose chip is left out and the others marked 「也屬於」.
 */
export interface CauseNavValue {
  href: (groupKey: string) => string
  current?: string
}

export const CauseNav = createContext<CauseNavValue>({ href: () => '' })

/** The text of a chip: the kind, and the id of a rule. */
function label(c: UpdateEntryCause): string {
  const name = CAUSE_KIND_NAMES[c.kind] ?? c.kind
  return c.id ? `${name} ${c.id}` : name
}

function Chip({ cause: c, href }: { cause: UpdateEntryCause; href: string }) {
  const text = label(c)
  const full = c.reason ? `${text}（${c.reason}）` : text
  if (!href) {
    return (
      <span class="uc-cause" title={full}>
        {text}
      </span>
    )
  }
  return (
    // A row of the table toggles on a click; the chip only goes to its group.
    <a class="uc-cause" href={href} title={full} onClick={(e) => e.stopPropagation()}>
      {text}
    </a>
  )
}

/** One chip per cause of the entry, each leading to its group; in a group, the chips of the other groups. */
export function CauseChips({ entry }: { entry: UpdateEntry }) {
  const nav = useContext(CauseNav)
  const seen = new Map<string, UpdateEntryCause>()
  for (const c of entry.causes ?? []) if (!seen.has(causeKey(c))) seen.set(causeKey(c), c)
  const chips = [...seen].filter(([key]) => key !== nav.current)
  if (chips.length === 0) return null
  const list = chips.map(([key, c]) => <Chip key={key} cause={c} href={nav.href(key)} />)
  if (nav.current === undefined) return <>{list}</>
  return (
    <span class="uc-also">
      <span class="uc-also-label">也屬於</span>
      {list}
    </span>
  )
}

/** The entry has causes, so that its chips are drawn instead of the old coarse one. */
export function hasCauses(e: UpdateEntry): boolean {
  return (e.causes?.length ?? 0) > 0
}
