import { stripTags } from '../search/match'
import { OriginBadge } from '../view/OriginBadge'
import type { Origin, TermRef, TermUnit, WarningItem } from '../types'
import { WARNING_MARKS, termMark } from './marks'
import { warningInfo } from './names'
import { ConfirmedLine, IdCopy, Pending, Row, UnitLink, UnitLinks } from './parts'

const chars = (s: string) => Array.from(stripTags(s)).length

/** A unit of a term-missing group: its source with the original term marked, its translation, the translator's note. */
function TermUnitRow({ term, unit }: { term: TermRef; unit: TermUnit }) {
  const krMark = termMark(term.kr)
  return (
    <li class={unit.accepted ? 'rv-tunit is-confirmed' : 'rv-tunit'}>
      <div class="rv-tunit-head">
        <UnitLink unitKey={unit.key} />
        <IdCopy id={unit.id} source={unit.zh} />
      </div>
      <ConfirmedLine confirmed={unit.accepted} />
      <dl class="rv-src">
        <Row label="KR" lang="kr" text={unit.kr} {...(krMark ? { mark: krMark } : {})} />
        <Row label="TW" lang="zh" text={unit.zh} />
      </dl>
      {unit.note !== undefined && <p class="rv-note-text">{unit.note}</p>}
    </li>
  )
}

/**
 * A term-missing group: one glossary entry, and each unit that lacks its rendering.
 * The units fold behind a line with their number; the group the address names and a search open them.
 */
function TermGroupCard({ item, term, units, target, showAccepted, unfold }: { item: WarningItem; term: TermRef; units: TermUnit[]; target: boolean; showAccepted: boolean; unfold: boolean }) {
  const cls = ['rv-card', item.accepted ? 'is-confirmed' : '', target ? 'is-target' : ''].filter(Boolean).join(' ')
  const info = warningInfo(item.code, item.message)
  const shown = showAccepted ? units : units.filter((u) => !u.accepted)
  // Both counts come from the units: the build gives a group one unit per key, and Keys are the keys of the units not accepted (see internal/site/review.go).
  const confirmed = units.filter((u) => u.accepted).length
  const open = units.length - confirmed
  return (
    <article class={cls} data-id={item.id}>
      <div class="rv-head">
        <span class="rv-kind rv-kind-warn">{info.name}</span>
        <span class="rv-where">
          詞彙表：{term.kr} → {term.zh}（{open} 處{confirmed > 0 && `，${confirmed} 處已確認`}）
        </span>
        {item.origins.map((o) => (
          <OriginBadge key={o} origin={o as Origin} />
        ))}
        <Pending pending={item.pending} units={item.keys.length} />
        <IdCopy id={item.id} copyId={`${item.id}@${open}`} source={`${term.kr} → ${term.zh}`} />
      </div>
      <p class="rv-msg">{item.message}</p>
      <ConfirmedLine confirmed={item.accepted} />
      {shown.length > 0 && (
        <details class="rv-tfold" open={target || unfold}>
          <summary>{shown.length} 個單位</summary>
          <ul class="rv-tunits">
            {shown.map((u) => (
              <TermUnitRow key={u.id} term={term} unit={u} />
            ))}
          </ul>
        </details>
      )}
    </article>
  )
}

export function WarningCard({ item, target, showAccepted = false, unfold = false }: { item: WarningItem; target: boolean; showAccepted?: boolean; unfold?: boolean }) {
  if (item.term && item.units) return <TermGroupCard item={item} term={item.term} units={item.units} target={target} showAccepted={showAccepted} unfold={unfold} />
  const info = warningInfo(item.code, item.message)
  const cls = ['rv-card', item.accepted ? 'is-confirmed' : '', target ? 'is-target' : ''].filter(Boolean).join(' ')
  const mark = WARNING_MARKS[item.code]
  return (
    <article class={cls} data-id={item.id}>
      <div class="rv-head">
        <span class="rv-kind rv-kind-warn">{info.name}</span>
        <span class="rv-where">
          {item.families.join('、')}・{item.keys.length} 個單位用了同一句
          {item.code === 'length-ratio' && `・KR ${chars(item.kr)} 字／TW ${chars(item.zh)} 字`}
        </span>
        {item.origins.map((o) => (
          <OriginBadge key={o} origin={o as Origin} />
        ))}
        <Pending pending={item.pending} units={item.keys.length} />
        <IdCopy id={item.id} source={item.zh} />
      </div>
      <p class="rv-msg">{item.message}</p>
      <ConfirmedLine confirmed={item.accepted} />
      <dl class="rv-src">
        <Row label="KR" lang="kr" text={item.kr} />
        <Row label="TW" lang="zh" text={item.zh} {...(mark ? { mark } : {})} />
      </dl>
      <UnitLinks keys={item.keys} />
    </article>
  )
}
