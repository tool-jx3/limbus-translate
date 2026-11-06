import { useMemo } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { loadGlossary } from '../data'
import { glossaryHref, termHref, type GlossaryOnly, type GlossarySort, type Route } from '../route'
import { LANG_ATTR, type ScopeEntry, type ScopeThresholds } from '../types'
import { useLoad } from './useLoad'

type GlossaryRoute = Extract<Route, { kind: 'glossary' }>

/** A share as a percentage, one decimal ("76.9%"). */
export const percent = (share: number): string => `${(share * 100).toFixed(1)}%`

/** A share as it reads in a sentence: 95% for 0.95, 87.5% for 0.875. */
const plain = (share: number): string => `${Number((share * 100).toFixed(1))}%`

const nonUsing = (e: ScopeEntry) => e.n - e.used
const byText = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0)

/** The ids of the families an entry hits ("family:<id>" groups). */
const familiesOf = (e: ScopeEntry) => e.groups.filter((g) => g.kind === 'family').map((g) => g.name.replace(/^family:/, ''))

/** The entries the filters keep, in the order asked: sort missing is most units without the rendering first, rate the lowest rate first. */
export function filterEntries(entries: ScopeEntry[], f: { q: string; sort: GlossarySort; only: GlossaryOnly; family: string }): ScopeEntry[] {
  const needle = f.q.trim().toLowerCase()
  const kept = entries.filter(
    (e) =>
      (f.only === '' || (f.only === 'hard' ? e.hard_candidate : nonUsing(e) > 0)) &&
      (f.family === '' || familiesOf(e).includes(f.family)) &&
      (needle === '' || [e.kr, e.zh, ...e.also, e.notes].some((x) => x.toLowerCase().includes(needle))),
  )
  const order: Record<GlossarySort, (a: ScopeEntry, b: ScopeEntry) => number> = {
    missing: (a, b) => nonUsing(b) - nonUsing(a) || byText(a.kr, b.kr),
    kr: (a, b) => byText(a.kr, b.kr) || byText(a.zh, b.zh),
    rate: (a, b) => a.rate - b.rate || byText(a.kr, b.kr),
    'rate-desc': (a, b) => b.rate - a.rate || byText(a.kr, b.kr),
  }
  return kept.sort(order[f.sort])
}

/** The scope of an entry in a line: files, not_files and match; word is the default and goes unsaid, so an entry with no files, no not_files and match word reads 全部. */
export function scopeSummary(e: ScopeEntry): string[] {
  const parts: string[] = []
  if (e.files.length > 0) parts.push(`files：${e.files.join('、')}`)
  if (e.not_files.length > 0) parts.push(`not_files：${e.not_files.join('、')}`)
  if (e.match === 'substring') parts.push('match：substring')
  return parts.length > 0 ? parts : ['全部']
}

/** The words of the limits an entry is a hard candidate by, with the values of the data. */
export function hardSentence(t: ScopeThresholds): string {
  return `應用率高於 ${plain(t.hard_rate)}、出現至少 ${t.hard_min_n} 個單位`
}

/** Goes to the list with other filters; a change of the text replaces the entry so that typing leaves one step in the history. */
function go(route: GlossaryRoute, change: Partial<Pick<GlossaryRoute, 'q' | 'sort' | 'only' | 'family'>>, replace = false): void {
  const href = glossaryHref({ q: route.q, sort: route.sort, only: route.only, family: route.family, ...change })
  if (replace) location.replace(href)
  else location.hash = href
}

const SORTS: { value: GlossarySort; label: string }[] = [
  { value: 'missing', label: '沒用數（多到少）' },
  { value: 'kr', label: '原詞' },
  { value: 'rate', label: '應用率（低到高）' },
  { value: 'rate-desc', label: '應用率（高到低）' },
]

const ONLY: { value: GlossaryOnly; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'missing', label: '有沒用的' },
  { value: 'hard', label: '可考慮退回' },
]

function Row({ e }: { e: ScopeEntry }) {
  return (
    <tr>
      <th scope="row" class="gl-name">
        <a href={termHref(e.id)} class="gl-kr" lang={LANG_ATTR.kr}>
          {e.kr}
        </a>
        {e.hard_candidate && <span class="gl-hard">可考慮退回</span>}
      </th>
      <td>
        <span class="gl-zh" lang={LANG_ATTR.zh}>
          {e.zh}
        </span>
        {e.also.length > 0 && (
          <span class="gl-also" lang={LANG_ATTR.zh}>
            （也可：{e.also.join('、')}）
          </span>
        )}
      </td>
      <td>
        <span class={e.check === 'hard' ? 'gl-check gl-check-hard' : 'gl-check'}>{e.check === 'hard' ? 'hard' : 'warn'}</span>
      </td>
      <td class="gl-scope">{scopeSummary(e).join('；')}</td>
      <td class="gl-num">{e.n}</td>
      <td class="gl-num">{e.used}</td>
      <td class="gl-num">{nonUsing(e)}</td>
      <td class="gl-num">{percent(e.rate)}</td>
    </tr>
  )
}

/** The glossary page: every entry with its counts, searchable, sortable, and filtered to what is worth a look. */
export function GlossaryView({ route }: { route: GlossaryRoute }) {
  const loaded = useLoad('glossary', loadGlossary)
  const data = loaded?.status === 'ready' ? loaded.data : undefined
  const shown = useMemo(
    () => (data ? filterEntries(data.entries, route) : []),
    [data, route.q, route.sort, route.only, route.family],
  )
  const families = useMemo(() => [...new Set((data?.entries ?? []).flatMap(familiesOf))].sort(), [data])
  return (
    <div class="glossary">
      <h1>詞彙</h1>
      <p class="rv-lede">
        詞彙表的每一條，與它命中的單位裡有多少用了譯名。點一條看用了與沒用的全部單位。
      </p>
      {loaded?.status === 'error' && <Notice error={loaded.error} />}
      {loaded === undefined && <p role="status">詞彙載入中…</p>}
      {data && (
        <>
          <div class="rv-tools">
            <input
              type="search"
              name="glossary-q"
              aria-label="搜尋詞條"
              placeholder="搜尋原詞、譯名、also、註記"
              value={route.q}
              onInput={(e) => go(route, { q: e.currentTarget.value }, true)}
            />
            <label>
              排序
              <select name="sort" value={route.sort} onChange={(e) => go(route, { sort: e.currentTarget.value as GlossarySort })}>
                {SORTS.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              只看
              <select name="only" value={route.only} onChange={(e) => go(route, { only: e.currentTarget.value as GlossaryOnly })}>
                {ONLY.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              族群
              <select name="family" value={route.family} onChange={(e) => go(route, { family: e.currentTarget.value })}>
                <option value="">全部</option>
                {families.map((f) => (
                  <option key={f} value={f}>
                    {f}
                  </option>
                ))}
              </select>
            </label>
            <span class="rv-count">
              {shown.length} 條（共 {data.entries.length} 條）
            </span>
          </div>
          {route.only === 'hard' && (
            <p class="gl-hard-note">
              可考慮退回：{hardSentence(data.thresholds)}，而且條目還不是 <code>check: hard</code>。確定要退回時，把條目改成 <code>check: hard</code>。
            </p>
          )}
          {shown.length === 0 ? (
            <p class="rv-empty">沒有符合的詞條</p>
          ) : (
            <div class="gl-table-wrap">
              <table class="gl-table">
                <thead>
                  <tr>
                    <th scope="col">原詞</th>
                    <th scope="col">譯名</th>
                    <th scope="col">檢查</th>
                    <th scope="col">範圍</th>
                    <th scope="col">出現</th>
                    <th scope="col">用了</th>
                    <th scope="col">沒用</th>
                    <th scope="col">應用率</th>
                  </tr>
                </thead>
                <tbody>
                  {shown.map((e) => (
                    <Row key={e.id} e={e} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  )
}
