import { reviewHref } from '../route'
import { warningInfo } from '../review/names'
import { TmpText } from '../tmp/TmpText'
import type { SuggestionItem, Unit } from '../types'

/** One chip under a TW text: what the unit has to look at. */
export interface Mark {
  /** The review item ID, "nt" for a note that only the unit holds, "pq" for the retranslation mark. */
  id: string
  kind: 'sugg' | 'warn' | 'note' | 'pq'
  label: string
  /** Whether a click opens a detail. */
  openable: boolean
}

/** The chips of a unit: its review items, a note it holds, and whether it waits for retranslation. */
export function unitMarks(u: Unit): Mark[] {
  const out: Mark[] = []
  let note = false
  for (const id of u.rv ?? []) {
    const [type = '', code = ''] = id.split('/')
    if (type === 'inconsistent') out.push({ id, kind: 'sugg', label: '不一致', openable: true })
    else if (type === 'warning') out.push({ id, kind: 'warn', label: `警告：${warningInfo(code, '').name}`, openable: true })
    else if (type === 'note') {
      note = true
      out.push({ id, kind: 'note', label: '註記', openable: true })
    }
  }
  if (u.nt !== undefined && !note) out.push({ id: 'nt', kind: 'note', label: '註記', openable: true })
  if (u.pq === 1) out.push({ id: 'pq', kind: 'pq', label: '待重翻', openable: false })
  return out
}

/** The suggestions the details need: undefined until asked for. */
export type SuggestionsState = { status: 'loading' } | { status: 'error' } | { status: 'ready'; items: SuggestionItem[] }

/** Which chip is open, and what the file view has loaded for it. */
export interface MarkState {
  /** The pointer of the unit and the ID of the mark that is open. */
  open?: { p: string; id: string }
  toggle: (p: string, id: string) => void
  suggestions?: SuggestionsState
}

/** The chips under the TW text of some units. */
export function MarkChips({ units, marks }: { units: readonly Unit[]; marks: MarkState }) {
  const chips = units.flatMap((u) => unitMarks(u).map((m) => ({ u, m })))
  if (chips.length === 0) return null
  return (
    <span class="chips">
      {chips.map(({ u, m }) => {
        if (!m.openable) {
          return (
            <span key={`${u.p}\n${m.id}`} class={`chip chip-${m.kind}`}>
              {m.label}
            </span>
          )
        }
        const open = marks.open?.p === u.p && marks.open.id === m.id
        return (
          <button key={`${u.p}\n${m.id}`} type="button" class={`chip chip-${m.kind}`} aria-expanded={open} onClick={() => marks.toggle(u.p, m.id)}>
            {m.label}
          </button>
        )
      })}
    </span>
  )
}

function Inconsistent({ id, suggestions }: { id: string; suggestions: SuggestionsState | undefined }) {
  if (suggestions === undefined || suggestions.status === 'loading') return <p class="mark-wait">載入中…</p>
  if (suggestions.status === 'error') return <p class="mark-wait">載入不到這一項。</p>
  const item = suggestions.items.find((i) => i.id === id)
  if (item === undefined) return <p class="mark-wait">審核頁裡沒有這一項，網站資料可能比這個檔案舊。</p>
  const variants = [...(item.variants ?? [])].sort((a, b) => b.count - a.count)
  return (
    <>
      <p class="mark-title">同一句原文有 {variants.length} 種譯法</p>
      <ul class="mark-variants">
        {variants.map((v) => (
          <li key={v.zh} class="mark-variant">
            <span class="mark-n">{v.count} 個</span>
            <span lang="zh-Hant-TW">
              <TmpText text={v.zh} raw={false} lang="zh" />
            </span>
          </li>
        ))}
      </ul>
    </>
  )
}

function Detail({ unit, id, suggestions }: { unit: Unit; id: string; suggestions: SuggestionsState | undefined }) {
  const type = id.split('/')[0]
  if (type === 'inconsistent') {
    return (
      <>
        <Inconsistent id={id} suggestions={suggestions} />
        <a href={reviewHref('suggestions', { id })}>到審核頁看這一項</a>
      </>
    )
  }
  if (type === 'warning') {
    const info = warningInfo(id.split('/')[1] ?? '', '')
    return (
      <>
        <p class="mark-title">{info.name}</p>
        {info.desc !== '' && <p>{info.desc}</p>}
        <a href={reviewHref('warnings', { id })}>到審核頁看這一項</a>
      </>
    )
  }
  return (
    <>
      <p class="mark-note" lang="zh-Hant-TW">
        {unit.nt ?? ''}
      </p>
      {type === 'note' && <a href={reviewHref('notes', { id })}>到審核頁看這一項</a>}
    </>
  )
}

/** The detail of the open chip, under the row of its unit; nothing where the open chip is of another row. */
export function MarkDetail({ units, marks }: { units: readonly Unit[]; marks: MarkState }) {
  const { open } = marks
  if (open === undefined) return null
  const unit = units.find((u) => u.p === open.p)
  if (unit === undefined || !unitMarks(unit).some((m) => m.id === open.id && m.openable)) return null
  return (
    <div class="mark-detail" data-mark={open.id}>
      <Detail unit={unit} id={open.id} suggestions={marks.suggestions} />
    </div>
  )
}
