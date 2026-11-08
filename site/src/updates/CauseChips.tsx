import { createContext } from 'preact'
import { useContext } from 'preact/hooks'
import type { UpdateEntry, UpdateEntryCause } from '../types'
import { CAUSE_KIND_NAMES, causeKey } from './causes'

/**
 * Where a cause chip leads. `href` gives the hash of the by-cause view with a
 * group open (empty: the chip is not a link); `current` is the group whose
 * cards are drawn, whose chip is left out and the others marked 「也屬於」.
 * `show`, given by the by-cause view, opens a group and brings it into view:
 * a chip calls it as well as going to its link, which does nothing when the
 * URL names that group already.
 */
export interface CauseNavValue {
  href: (groupKey: string) => string
  current?: string
  show?: (groupKey: string) => void
}

export const CauseNav = createContext<CauseNavValue>({ href: () => '' })

/** The text of a chip: the kind, and the id of a rule. */
function label(c: UpdateEntryCause): string {
  const name = CAUSE_KIND_NAMES[c.kind] ?? c.kind
  return c.id ? `${name} ${c.id}` : name
}

/** The reason of an adjustment (or an expired one), which a person wrote for the unit; else none. */
export function adjustmentReason(c: UpdateEntryCause): string | undefined {
  return (c.kind === 'adjustment' || c.kind === 'adjustment-expired') && c.reason ? c.reason : undefined
}

function Reason({ text }: { text: string }) {
  return <span class="uc-reason">理由：{text}</span>
}

function Chip({ cause: c, groupKey, reason }: { cause: UpdateEntryCause; groupKey: string; reason: boolean }) {
  const nav = useContext(CauseNav)
  const text = label(c)
  const full = c.reason ? `${text}（${c.reason}）` : text
  const href = nav.href(groupKey)
  const why = reason ? adjustmentReason(c) : undefined
  const chip = href ? (
    // A row of the table toggles on a click; the chip only goes to its group.
    <a
      class="uc-cause"
      href={href}
      title={full}
      onClick={(e) => {
        e.stopPropagation()
        nav.show?.(groupKey)
      }}
    >
      {text}
    </a>
  ) : (
    <span class="uc-cause" title={full}>
      {text}
    </span>
  )
  return (
    <>
      {chip}
      {why && <Reason text={why} />}
    </>
  )
}

/**
 * One chip per cause of the entry, each leading to its group; in a group, the
 * chips of the other groups. With `reasons`, the reason of an adjustment is
 * shown next to its chip, and in the group of the adjustment in place of it.
 */
export function CauseChips({ entry, reasons = false }: { entry: UpdateEntry; reasons?: boolean }) {
  const nav = useContext(CauseNav)
  const seen = new Map<string, UpdateEntryCause>()
  for (const c of entry.causes ?? []) if (!seen.has(causeKey(c))) seen.set(causeKey(c), c)
  const own = nav.current === undefined ? undefined : seen.get(nav.current)
  const ownReason = reasons && own ? adjustmentReason(own) : undefined
  const chips = [...seen].filter(([key]) => key !== nav.current)
  const list = chips.map(([key, c]) => <Chip key={key} cause={c} groupKey={key} reason={reasons} />)
  if (nav.current === undefined) return <>{list}</>
  return (
    <>
      {ownReason && <Reason text={ownReason} />}
      {chips.length > 0 && (
        <span class="uc-also">
          <span class="uc-also-label">也屬於</span>
          {list}
        </span>
      )}
    </>
  )
}

/** The entry has causes, so that its chips are drawn instead of the old coarse one. */
export function hasCauses(e: UpdateEntry): boolean {
  return (e.causes?.length ?? 0) > 0
}
