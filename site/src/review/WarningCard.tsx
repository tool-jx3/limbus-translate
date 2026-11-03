import { stripTags } from '../search/match'
import { OriginBadge } from '../view/OriginBadge'
import type { Origin, WarningItem } from '../types'
import { WARNING_MARKS } from './marks'
import { warningInfo } from './names'
import { ConfirmedLine, IdCopy, Pending, Row, UnitLinks } from './parts'

const chars = (s: string) => Array.from(stripTags(s)).length

export function WarningCard({ item, target }: { item: WarningItem; target: boolean }) {
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
      <ConfirmedLine confirmed={item.accepted} />
      <dl class="rv-src">
        <Row label="KR" lang="kr" text={item.kr} />
        <Row label="TW" lang="zh" text={item.zh} {...(mark ? { mark } : {})} />
      </dl>
      <UnitLinks keys={item.keys} />
    </article>
  )
}
