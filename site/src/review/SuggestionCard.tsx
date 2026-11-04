import { stripTags } from '../search/match'
import { TmpText } from '../tmp/TmpText'
import { LANG_ATTR, type JPENClass, type JPENGroup, type SuggestionItem } from '../types'
import { variantDiff } from './diff'
import { suggestionUnits } from './filter'
import { suggestionKindName } from './names'
import { ConfirmedLine, DiffText, IdCopy, Pending, Row, UnitLink, UnitLinks, splitKey } from './parts'

function clip(s: string, n: number): string {
  const chars = Array.from(stripTags(s).replace(/\s+/g, ' ').trim())
  return chars.length > n ? `${chars.slice(0, n).join('')}…` : chars.join('')
}

/** The badge of a group in the JP/EN view. */
export const JPEN_BADGES: Record<JPENClass, string> = {
  explained: 'JP/EN 可解釋',
  partial: '部分可解釋',
  same: 'JP/EN 相同',
}

const pairOf = (j: JPENGroup) => `${j.jp === undefined ? '-' : `+${j.jp}`}\u0000${j.en === undefined ? '-' : `+${j.en}`}`

/** The JP/EN groups of a variant; a pair that other translations have too is marked. */
function JPENRows({ groups, shared }: { groups: JPENGroup[]; shared: Set<string> }) {
  return (
    <ul class="rv-jpens">
      {groups.map((j) => {
        const isShared = shared.has(pairOf(j))
        return (
          <li key={pairOf(j)} class={isShared ? 'rv-jpen is-shared' : 'rv-jpen'}>
            <span class="rv-jpen-label">JP</span>
            <span class="rv-jpen-jp" lang={LANG_ATTR.jp}>
              {j.jp === undefined ? '—' : <TmpText text={j.jp} raw={false} lang="jp" />}
            </span>
            <span class="rv-jpen-label">EN</span>
            <span class="rv-jpen-en" lang={LANG_ATTR.en}>
              {j.en === undefined ? '—' : <TmpText text={j.en} raw={false} lang="en" />}
            </span>
            <span class="rv-jpen-n">{j.count} 個</span>
            {isShared && <span class="rv-jpen-shared">同樣的 JP/EN 有其他譯法</span>}
          </li>
        )
      })}
    </ul>
  )
}

function Inconsistent({ item, jpen }: { item: SuggestionItem; jpen: boolean }) {
  const variants = [...(item.variants ?? [])].sort((a, b) => b.count - a.count)
  const top = variants[0]
  if (!top) return null
  const total = suggestionUnits(item)
  // The JP/EN pairs that more than one translation has: where the group really differs.
  const seen = new Map<string, number>()
  for (const v of variants) for (const p of new Set((v.jpen ?? []).map(pairOf))) seen.set(p, (seen.get(p) ?? 0) + 1)
  const shared = new Set([...seen].filter(([, n]) => n > 1).map(([p]) => p))
  return (
    <>
      <dl class="rv-src">
        {item.kr !== undefined && <Row label="KR" lang="kr" text={item.kr} />}
        {item.jp !== undefined && <Row label="JP" lang="jp" text={item.jp} />}
        {item.en !== undefined && <Row label="EN" lang="en" text={item.en} />}
      </dl>
      <div class="rv-vars-title">TW 的譯法（和最多的那種不同的地方以底色標出）</div>
      <div class="rv-vars">
        {variants.map((v, k) => (
          <div key={v.zh} class={k === 0 ? 'rv-var is-top' : 'rv-var'}>
            <div class="rv-bar">
              <i style={{ width: `${Math.max(6, Math.round((64 * v.count) / top.count))}px` }} />
              {v.count} 個
            </div>
            <div class="rv-zh">
              <DiffText pieces={variantDiff(top.zh, v.zh)} />
            </div>
            <div class="rv-example">
              {v.keys[0] !== undefined && (
                <>
                  例：
                  <UnitLink unitKey={v.keys[0]} text={splitKey(v.keys[0]).path} />
                </>
              )}
            </div>
            {jpen && v.jpen && v.jpen.length > 0 && <JPENRows groups={v.jpen} shared={shared} />}
          </div>
        ))}
      </div>
      <details class="rv-more">
        <summary>全部 {total} 個單位</summary>
        <ul>
          {variants.map((v) => (
            <li key={v.zh}>
              {clip(v.zh, 24)}（{v.count}）：
              <UnitLinks keys={v.keys} />
            </li>
          ))}
        </ul>
      </details>
    </>
  )
}

function Evidence({ item }: { item: SuggestionItem }) {
  if (item.evidence.length === 0) return null
  return (
    <ul class="rv-evidence">
      {item.evidence.map((e, k) => (
        <li key={k}>
          {e.key !== undefined && <UnitLink unitKey={e.key} />}
          {e.count !== undefined && <span class="rv-times">{e.count} 次</span>}
          {e.note !== undefined && <p class="rv-note-text">{e.note}</p>}
          {(e.kr !== undefined || e.zh !== undefined) && (
            <dl class="rv-src">
              {e.kr !== undefined && <Row label="KR" lang="kr" text={e.kr} />}
              {e.zh !== undefined && <Row label="TW" lang="zh" text={e.zh} />}
            </dl>
          )}
        </li>
      ))}
    </ul>
  )
}

export function SuggestionCard({ item, target, jpen = false }: { item: SuggestionItem; target: boolean; jpen?: boolean }) {
  const inconsistent = item.kind === 'inconsistent'
  const units = suggestionUnits(item)
  const source = item.kr ?? item.evidence.find((e) => e.kr !== undefined)?.kr ?? item.evidence.find((e) => e.zh !== undefined)?.zh ?? item.title
  const cls = ['rv-card', item.declined ? 'is-confirmed' : '', target ? 'is-target' : ''].filter(Boolean).join(' ')
  return (
    <article class={cls} data-id={item.id}>
      <div class="rv-head">
        <span class="rv-kind rv-kind-sugg">{suggestionKindName(item.kind)}</span>
        <span class="rv-where">
          {inconsistent
            ? `${item.family ?? ''}／${item.field ?? ''}・${units} 個單位・${item.variants?.length ?? 0} 種譯法`
            : item.title}
        </span>
        {jpen && inconsistent && item.jpen_class && <span class={`rv-badge rv-badge-${item.jpen_class}`}>{JPEN_BADGES[item.jpen_class]}</span>}
        <Pending pending={item.pending} units={units} />
        <IdCopy id={item.id} source={source} />
      </div>
      <ConfirmedLine confirmed={item.declined} />
      {inconsistent ? <Inconsistent item={item} jpen={jpen} /> : <Evidence item={item} />}
    </article>
  )
}
