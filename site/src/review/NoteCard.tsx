import { OriginBadge } from '../view/OriginBadge'
import type { NoteItem, Origin } from '../types'
import { ConfirmedLine, IdCopy, Pending, Row, UnitLinks } from './parts'

/** The date of a batch id such as 20261101T045817-cf5f93d5. */
function batchDate(batch: string): string {
  const m = /^(\d{4})(\d{2})(\d{2})/.exec(batch)
  return m ? `${m[1]}-${m[2]}-${m[3]}` : batch
}

export function NoteCard({ item, target }: { item: NoteItem; target: boolean }) {
  const cls = ['rv-card', item.declined ? 'is-confirmed' : '', target ? 'is-target' : ''].filter(Boolean).join(' ')
  return (
    <article class={cls} data-id={item.id}>
      <div class="rv-head">
        <span class="rv-kind rv-kind-note">譯者註記</span>
        <span class="rv-where">
          {item.family}・{batchDate(item.batch)} 的批次
        </span>
        <OriginBadge origin={item.o as Origin} />
        <Pending pending={item.pending} units={1} />
        <IdCopy id={item.id} source={item.zh} />
      </div>
      <ConfirmedLine confirmed={item.declined} />
      <p class="rv-note-text">{item.note}</p>
      <dl class="rv-src">
        <Row label="KR" lang="kr" text={item.kr} />
        <Row label="TW" lang="zh" text={item.zh} />
      </dl>
      <UnitLinks keys={[item.key]} />
    </article>
  )
}
