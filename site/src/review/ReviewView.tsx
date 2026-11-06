import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { DataError, loadReview } from '../data'
import { reviewHref, type Route } from '../route'
import type { Settings } from '../settings'
import type { Index, Meta, NoteItem, ReviewData, ReviewTab, ScopeReport, SuggestionItem, WarningItem } from '../types'
import { filterNotes, filterSuggestions, filterWarnings, jpenCounts, sortSuggestions, type ReviewFilter } from './filter'
import { warningInfo } from './names'
import { NoteCard } from './NoteCard'
import { ShownText } from './parts'
import { filterProposals, ScopeBody } from './ScopeView'
import { SuggestionCard } from './SuggestionCard'
import { TodoView } from './TodoView'
import { WarningCard } from './WarningCard'

type ReviewRoute = Extract<Route, { kind: 'review' }>

const PAGE = 100
const TABS: { tab: ReviewTab; name: string }[] = [
  { tab: 'suggestions', name: '規則建議' },
  { tab: 'warnings', name: '檢查警告' },
  { tab: 'notes', name: '譯者註記' },
  { tab: 'todo', name: '待辦清單' },
  { tab: 'scope', name: '詞彙範圍' },
]
const QUICK = ['新詞', '諧音', '俗語', '沿用', '標籤']

type Loaded = { tab: ReviewTab } & ({ status: 'ready'; data: ReviewData[ReviewTab] } | { status: 'error'; error: DataError })

/** The document of a tab of the review, once loaded; undefined while it loads. */
function useReview(tab: ReviewTab): Loaded | undefined {
  const [state, setState] = useState<Loaded>()
  useEffect(() => {
    let live = true
    loadReview(tab).then(
      (data) => {
        if (live) setState({ tab, status: 'ready', data })
      },
      (e: unknown) => {
        if (live) setState({ tab, status: 'error', error: e instanceof DataError ? e : new DataError('network', String(e)) })
      },
    )
    return () => {
      live = false
    }
  }, [tab])
  return state?.tab === tab ? state : undefined
}

/** Goes to the page with other filters; a change of the text replaces the entry so that typing leaves one step in the history. */
function go(route: ReviewRoute, change: Partial<Pick<ReviewRoute, 'family' | 'sub' | 'q' | 'all' | 'mode' | 'jpen'>>, replace = false): void {
  const next = { family: route.family, sub: route.sub, q: route.q, all: route.all, mode: route.mode, jpen: route.jpen, ...change }
  const href = reviewHref(route.tab, next)
  if (replace) location.replace(href)
  else location.hash = href
}

const unique = (xs: string[]) => [...new Set(xs)].sort()

interface Option {
  value: string
  label: string
}

function Select({ name, label, value, options, onChange }: { name: string; label: string; value: string; options: Option[]; onChange: (v: string) => void }) {
  return (
    <label>
      {label}
      <select name={name} value={value} onChange={(e) => onChange(e.currentTarget.value)}>
        <option value="">全部</option>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  )
}

interface ListProps<T> {
  items: T[]
  /** The id the address names. */
  id: string | undefined
  idOf: (item: T) => string
  /** The card of an item; before is what comes before the item in the list, if anything. */
  render: (item: T, target: boolean, previous: T | undefined) => preact.ComponentChildren
}

/** The items 100 at a time, always with the one the address names, which is scrolled to. */
function ItemList<T>({ items, id, idOf, render }: ListProps<T>) {
  const [limit, setLimit] = useState(PAGE)
  const box = useRef<HTMLDivElement>(null)
  const at = id === undefined ? -1 : items.findIndex((it) => idOf(it) === id)
  const shown = Math.min(items.length, Math.max(limit, at + 1))
  useEffect(() => {
    box.current?.querySelector('.is-target')?.scrollIntoView?.({ block: 'center' })
  }, [id, at >= 0])
  if (items.length === 0) return <p class="rv-empty">沒有符合的項目</p>
  return (
    <>
      <div class="rv-list" ref={box}>
        {items.slice(0, shown).map((it, k) => render(it, idOf(it) === id, items[k - 1]))}
      </div>
      {shown < items.length && (
        <button type="button" class="rv-more" onClick={() => setLimit(shown + PAGE)}>
          再顯示 {PAGE} 項（共 {items.length}）
        </button>
      )}
    </>
  )
}

function Section({ code, count, message }: { code: string; count: number; message: string }) {
  const info = warningInfo(code, message)
  return (
    <div class="rv-section">
      <h2>
        {info.name} <span class="rv-n">{count} 項</span>
      </h2>
      <p>{info.desc}</p>
    </div>
  )
}

function Missing({ id }: { id: string | undefined }) {
  return (
    <p class="rv-missing" role="status">
      找不到項目 <code>{id}</code>：它可能已解決，或被目前的篩選擋住了。
    </p>
  )
}

interface ToolsProps {
  route: ReviewRoute
  /** Number of items shown after the filters. */
  count: number
  /** Number of confirmed items in the tab. */
  confirmed: number
  families: string[]
  kinds?: Option[]
  kindLabel?: string
  sort?: { value: 'units' | 'variants'; onChange: (v: 'units' | 'variants') => void }
  quick?: boolean
  placeholder?: string
  /** The 只看 KR／連同 JP/EN switch, and in the JP/EN view the class filter with its counts. */
  jpen?: Record<'explained' | 'partial' | 'same', number>
}

function Tools({ route, count, confirmed, families, kinds, kindLabel, sort, quick, jpen, placeholder = '篩選 ID、檔名、KR、TW、註記' }: ToolsProps) {
  return (
    <div class="rv-tools">
      {jpen && (
        <label>
          檢視
          <select name="mode" value={route.mode} onChange={(e) => go(route, { mode: e.currentTarget.value === 'jpen' ? 'jpen' : 'kr', jpen: '' })}>
            <option value="kr">只看 KR</option>
            <option value="jpen">連同 JP/EN</option>
          </select>
        </label>
      )}
      {jpen && route.mode === 'jpen' && (
        <Select
          name="jpen"
          label="JP/EN"
          value={route.jpen}
          options={[
            { value: 'explained', label: `可解釋（${jpen.explained}）` },
            { value: 'partial', label: `部分可解釋（${jpen.partial}）` },
            { value: 'same', label: `相同（${jpen.same}）` },
          ]}
          onChange={(v) => go(route, { jpen: v === 'explained' || v === 'partial' || v === 'same' ? v : '' })}
        />
      )}
      {families.length > 0 && (
        <Select name="family" label="族群" value={route.family} options={families.map((f) => ({ value: f, label: f }))} onChange={(family) => go(route, { family })} />
      )}
      {kinds && kindLabel && <Select name="sub" label={kindLabel} value={route.sub} options={kinds} onChange={(sub) => go(route, { sub })} />}
      {sort && (
        <label>
          排序
          <select name="sort" value={sort.value} onChange={(e) => sort.onChange(e.currentTarget.value === 'variants' ? 'variants' : 'units')}>
            <option value="units">與 lct suggest 相同</option>
            <option value="variants">譯法多的在前</option>
          </select>
        </label>
      )}
      <input
        type="search"
        name="q"
        placeholder={placeholder}
        aria-label="篩選文字"
        value={route.q}
        onInput={(e) => go(route, { q: e.currentTarget.value }, true)}
      />
      {quick && (
        <span class="rv-quick">
          {QUICK.map((w) => (
            <button key={w} type="button" onClick={() => go(route, { q: w })}>
              {w}
            </button>
          ))}
        </span>
      )}
      <label class="rv-all">
        <input type="checkbox" checked={route.all} onChange={(e) => go(route, { all: e.currentTarget.checked })} />
        顯示已確認的 {confirmed} 項
      </label>
      <span class="rv-count">符合 {count} 項</span>
    </div>
  )
}

function Suggestions({ route, items }: { route: ReviewRoute; items: SuggestionItem[] }) {
  const [sort, setSort] = useState<'units' | 'variants'>('units')
  const byJPEN = route.mode === 'jpen'
  const filter: ReviewFilter = { ...route, jpen: byJPEN ? route.jpen : '' }
  const shown = useMemo(
    () => sortSuggestions(filterSuggestions(items, filter, route.id), sort),
    [items, route.family, route.sub, route.q, route.all, route.id, sort, byJPEN, route.jpen],
  )
  const counts = useMemo(() => jpenCounts(items, filter, route.id), [items, route.family, route.sub, route.q, route.all, route.id])
  const kinds = unique(items.map((s) => s.kind)).map((k) => ({ value: k, label: k }))
  return (
    <>
      <Tools
        route={route}
        count={shown.length}
        confirmed={items.filter((s) => s.declined).length}
        families={unique(items.flatMap((s) => (s.family ? [s.family] : [])))}
        kinds={kinds}
        kindLabel="種類"
        sort={{ value: sort, onChange: setSort }}
        jpen={counts}
      />
      <div class="rv-body">
        {route.id !== undefined && !shown.some((s) => s.id === route.id) && <Missing id={route.id} />}
        <ItemList
          key={`${route.family}|${route.sub}|${route.q}|${route.all}|${sort}|${route.mode}|${route.jpen}`}
          items={shown}
          id={route.id}
          idOf={(s) => s.id}
          render={(s, target) => <SuggestionCard key={s.id} item={s} target={target} jpen={byJPEN} />}
        />
      </div>
    </>
  )
}

function Warnings({ route, items }: { route: ReviewRoute; items: WarningItem[] }) {
  const shown = useMemo(() => filterWarnings(items, route, route.id), [items, route.family, route.sub, route.q, route.all, route.id])
  const counts = new Map<string, number>()
  for (const w of items) counts.set(w.code, (counts.get(w.code) ?? 0) + 1)
  const sectionCounts = new Map<string, number>()
  for (const w of shown) sectionCounts.set(w.code, (sectionCounts.get(w.code) ?? 0) + 1)
  const kinds = [...counts.keys()].sort().map((c) => ({ value: c, label: `${warningInfo(c, '').name}（${counts.get(c)}）` }))
  return (
    <>
      <Tools
        route={route}
        count={shown.length}
        confirmed={items.filter((w) => w.accepted).length}
        families={unique(items.flatMap((w) => w.families))}
        kinds={kinds}
        kindLabel="檢查"
      />
      <div class="rv-body">
        {route.id !== undefined && !shown.some((w) => w.id === route.id) && <Missing id={route.id} />}
        <ItemList
          key={`${route.family}|${route.sub}|${route.q}|${route.all}`}
          items={shown}
          id={route.id}
          idOf={(w) => w.id}
          render={(w, target, previous) => (
            <>
              {previous?.code !== w.code && <Section key={`h-${w.code}`} code={w.code} count={sectionCounts.get(w.code) ?? 0} message={w.message} />}
              <WarningCard key={w.id} item={w} target={target} showAccepted={route.all} />
            </>
          )}
        />
      </div>
    </>
  )
}

function Notes({ route, items }: { route: ReviewRoute; items: NoteItem[] }) {
  const shown = useMemo(() => filterNotes(items, route, route.id), [items, route.family, route.q, route.all, route.id])
  return (
    <>
      <Tools route={route} count={shown.length} confirmed={items.filter((n) => n.declined).length} families={unique(items.map((n) => n.family))} quick />
      <div class="rv-body">
        {route.id !== undefined && !shown.some((n) => n.id === route.id) && <Missing id={route.id} />}
        <ItemList key={`${route.family}|${route.q}|${route.all}`} items={shown} id={route.id} idOf={(n) => n.id} render={(n, target) => <NoteCard key={n.id} item={n} target={target} />} />
      </div>
    </>
  )
}

function Scope({ route, report }: { route: ReviewRoute; report: ScopeReport }) {
  const filter = { q: route.q, all: route.all, id: route.id }
  const count = filterProposals(report.proposals, filter).length
  return (
    <>
      <Tools route={route} count={count} confirmed={report.proposals.filter((p) => p.declined).length} families={[]} placeholder="篩選建議的 ID、KR、譯名、樣式" />
      <ScopeBody report={report} filter={filter} />
    </>
  )
}

export interface ReviewViewProps {
  route: ReviewRoute
  index: Index
  meta: Meta
  /** The languages and the raw tags of the header: the cards show the game text by them. */
  settings: Settings
}

/** The review page: the items that need a person, by tab. */
export function ReviewView({ route, meta, settings }: ReviewViewProps) {
  const loaded = useReview(route.tab)
  const shown = useMemo(() => ({ langs: settings.langs, raw: settings.rawTags }), [settings.langs, settings.rawTags])
  return (
    <ShownText.Provider value={shown}>
      <div class="review">
        <h1>審核</h1>
        <p class="rv-lede">這次建置需要人看的地方。每一項都連到所在檔案的那一列；看完後在對話中貼上項目 ID，說你要怎麼改。</p>
        <p class="rv-howto">按「複製 ID」，貼到對話中再加上你的決定；Claude 會改規則或把項目記為確認過，下次建置後這頁就跟著更新。</p>
        <nav class="rv-tabs" role="tablist" aria-label="審核項目類別">
          {TABS.map(({ tab, name }) => (
            <a key={tab} role="tab" class="rv-tab" href={reviewHref(tab)} aria-selected={tab === route.tab ? 'true' : 'false'}>
              {name}
              <span class="rv-n">{meta.review[tab]}</span>
            </a>
          ))}
        </nav>
        {loaded === undefined && <p role="status">載入中…</p>}
        {loaded?.status === 'error' && <Notice error={loaded.error} />}
        {loaded?.status === 'ready' && route.tab === 'suggestions' && <Suggestions route={route} items={(loaded.data as ReviewData['suggestions']).items} />}
        {loaded?.status === 'ready' && route.tab === 'warnings' && <Warnings route={route} items={(loaded.data as ReviewData['warnings']).items} />}
        {loaded?.status === 'ready' && route.tab === 'notes' && <Notes route={route} items={(loaded.data as ReviewData['notes']).items} />}
        {loaded?.status === 'ready' && route.tab === 'scope' && <Scope route={route} report={loaded.data as ReviewData['scope']} />}
        {loaded?.status === 'ready' && route.tab === 'todo' && (
          <div class="rv-body">
            <TodoView sections={(loaded.data as ReviewData['todo']).sections} />
          </div>
        )}
      </div>
    </ShownText.Provider>
  )
}
