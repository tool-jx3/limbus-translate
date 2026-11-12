import { useMemo, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { loadNames } from '../data'
import { fileHref, glossaryHref, namesHref, reviewHref, type NamesOnly, type NamesSort, type Route } from '../route'
import { LANG_ATTR, type NameRow, type NameVariant } from '../types'
import { filterNames, sortNames } from './names'
import { useLoad } from './useLoad'

type NamesRoute = Extract<Route, { kind: 'names' }>

/** How many rows are drawn at first, and how many more at each press. */
const PAGE = 200

const ONLY: { value: NamesOnly; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'multi', label: '有多種譯法' },
  { value: 'review', label: '這次有不同的新譯' },
  { value: 'glossary', label: '詞彙表已有' },
  { value: 'declined', label: '已不採用' },
]

const SORTS: { value: NamesSort; label: string }[] = [
  { value: 'units', label: '單位數（多到少）' },
  { value: 'kr', label: '原詞（A→Z）' },
]

/** Goes to the list with other filters; a change of the text replaces the entry so that typing leaves one step in the history. */
function go(route: NamesRoute, change: Partial<Pick<NamesRoute, 'q' | 'only' | 'sort'>>, replace = false): void {
  const href = namesHref({ q: route.q, only: route.only, sort: route.sort, ...change })
  if (replace) location.replace(href)
  else location.hash = href
}

/** The file and pointer of a unit key "<file>|<pointer>". */
function keyHref(key: string): string {
  const bar = key.indexOf('|')
  return bar < 0 ? fileHref(key) : fileHref(key.slice(0, bar), key.slice(bar + 1))
}

function Variant({ v }: { v: NameVariant }) {
  const [main, ...others] = v.renderings
  return (
    <div class="nm-variant">
      <div class="nm-src">
        {v.jp !== undefined && <span lang={LANG_ATTR.jp}>{v.jp}</span>}
        {v.en !== undefined && <span lang={LANG_ATTR.en}>{v.en}</span>}
      </div>
      <div class="nm-zh" lang={LANG_ATTR.zh}>
        {main && <span class="nm-main">{`${main.zh} ${main.units}`}</span>}
        {others.map((r) => (
          <span key={r.zh} class="nm-other">{`${r.zh} ${r.units}`}</span>
        ))}
      </div>
      {v.examples.length > 0 && (
        <div class="nm-example">
          {v.examples.map((k, i) => (
            <a key={k} href={keyHref(k)}>
              {i + 1}
            </a>
          ))}
        </div>
      )}
    </div>
  )
}

function Row({ r }: { r: NameRow }) {
  return (
    <tr>
      <th scope="row" class="nm-name">
        <span class="nm-kr" lang={LANG_ATTR.kr}>
          {r.kr}
        </span>
        <span class="nm-units">{r.units}</span>
      </th>
      <td>
        {r.variants.map((v, i) => (
          <Variant key={i} v={v} />
        ))}
      </td>
      <td class="nm-marks">
        {r.glossary && <a href={glossaryHref({ q: r.kr })}>詞彙表已有</a>}
        {r.declined && <span>已不採用</span>}
        {r.review && <a href={reviewHref('suggestions', { id: `name/${r.kr}` })}>這次有不同的新譯</a>}
      </td>
    </tr>
  )
}

/** The page of the existing names: what the translations called each name before the last update. View only; decisions are on the review page. */
export function NamesView({ route }: { route: NamesRoute }) {
  const loaded = useLoad('names', loadNames)
  const data = loaded?.status === 'ready' ? loaded.data : undefined
  const shown = useMemo(() => (data ? sortNames(filterNames(data.names, route.q, route.only), route.sort) : []), [data, route.q, route.only, route.sort])
  const [limit, setLimit] = useState(PAGE)
  return (
    <div class="glossary names">
      <h1>既有譯名</h1>
      {data && (
        <p class="rv-lede">
          {data.raw_tag === '' ? (
            <>還沒有既有譯名表：下一次 <code>lct plan</code> 會產生。</>
          ) : (
            <>
              <code>lct plan</code> 在 <code>{data.raw_tag}</code> 更新翻譯之前，從現行譯文的名稱欄位（<code>{data.fields.join('、')}</code>）整理的譯名。翻譯時帶給譯者參考；詞彙表有的條目以詞彙表為準。
            </>
          )}
        </p>
      )}
      {loaded?.status === 'error' && <Notice error={loaded.error} />}
      {loaded === undefined && <p role="status">既有譯名載入中…</p>}
      {data && data.raw_tag !== '' && (
        <>
          <div class="rv-tools">
            <input
              type="search"
              name="names-q"
              aria-label="搜尋譯名"
              placeholder="搜尋原詞、日文、英文、譯法"
              value={route.q}
              onInput={(e) => go(route, { q: e.currentTarget.value }, true)}
            />
            <label>
              只看
              <select name="only" value={route.only} onChange={(e) => go(route, { only: e.currentTarget.value as NamesOnly })}>
                {ONLY.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              排序
              <select name="sort" value={route.sort} onChange={(e) => go(route, { sort: e.currentTarget.value as NamesSort })}>
                {SORTS.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
            <span class="rv-count">
              {shown.length} 條（共 {data.names.length} 條）
            </span>
          </div>
          {shown.length === 0 ? (
            <p class="rv-empty">沒有符合的譯名</p>
          ) : (
            <>
              <div class="gl-table-wrap">
                <table class="gl-table nm-table">
                  <thead>
                    <tr>
                      <th scope="col">原詞</th>
                      <th scope="col">譯法</th>
                      <th scope="col">狀態</th>
                    </tr>
                  </thead>
                  <tbody>
                    {shown.slice(0, limit).map((r) => (
                      <Row key={r.kr} r={r} />
                    ))}
                  </tbody>
                </table>
              </div>
              {limit < shown.length && (
                <button type="button" class="rv-more" onClick={() => setLimit(limit + PAGE)}>
                  再顯示 {Math.min(PAGE, shown.length - limit)} 條（共 {shown.length}）
                </button>
              )}
            </>
          )}
        </>
      )}
    </div>
  )
}
