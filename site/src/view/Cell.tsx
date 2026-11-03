import { TmpText } from '../tmp/TmpText'
import { LANG_ATTR, type Lang, type Unit } from '../types'
import { MarkChips, type MarkState } from './marks'
import { OriginBadge } from './OriginBadge'

export const LANG_LABEL: Record<Lang, string> = { kr: 'KR', jp: 'JP', en: 'EN', zh: 'TW' }

/** The origin marks and flags under a TW text: each origin once, then each flag. */
function ZhMeta({ units }: { units: readonly Unit[] }) {
  const origins = [...new Set(units.map((u) => u.o))]
  const flags = [...new Set(units.flatMap((u) => u.fl ?? []))]
  return (
    <span class="cell-meta">
      {origins.map((o) => (
        <OriginBadge key={o} origin={o} />
      ))}
      {flags.map((f) => (
        <span key={f} class="flag" lang="en">
          {f}
        </span>
      ))}
    </span>
  )
}

export interface CellProps {
  lang: Lang
  /** One unit, or the teller with its title. */
  units: readonly Unit[]
  raw: boolean
  /** The ARIA role, where the cell is a cell of a table. */
  role?: 'cell'
  /** The review chips under the TW text. */
  marks?: MarkState
}

/**
 * The text of the units in one language. A unit the language has no string for
 * is a dash. The TW cell carries the origin of its text.
 */
export function Cell({ lang, units, raw, role, marks }: CellProps) {
  const present = units.filter((u) => u[lang] !== undefined)
  const reviewed = lang === 'zh' && units.some((u) => (u.rv?.length ?? 0) > 0)
  return (
    <div class={reviewed ? `cell cell-${lang} has-review` : `cell cell-${lang}`} role={role} data-lang={lang} data-tag={LANG_LABEL[lang]} lang={LANG_ATTR[lang]}>
      {present.length === 0 ? (
        <span class="missing" title="沒有對應的字串">
          —
        </span>
      ) : (
        present.map((u) => (
          <span key={u.p} class={`u u-${u.f}`}>
            <TmpText text={u[lang] ?? ''} raw={raw} lang={lang} />
          </span>
        ))
      )}
      {lang === 'zh' && <ZhMeta units={units} />}
      {lang === 'zh' && marks !== undefined && <MarkChips units={units} marks={marks} />}
    </div>
  )
}
