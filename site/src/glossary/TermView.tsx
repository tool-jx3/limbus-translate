import { useMemo, useRef, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { loadGlossary, loadGlossaryEntry } from '../data'
import { stripTags } from '../search/match'
import { copyText, copyToClipboard } from '../review/copy'
import { termMark } from '../review/marks'
import { ConfirmedLine, IdCopy, Row, ShownText, UnitLink } from '../review/parts'
import { reviewHref, termHref, type Route, type TermFilter } from '../route'
import type { Settings } from '../settings'
import { OriginBadge } from '../view/OriginBadge'
import { LANG_ATTR, type FileTag, type GlossaryFile, type GlossaryUnit, type ScopeEntry } from '../types'
import { hardSentence, percent, scopeSummary } from './GlossaryView'
import { useLoad } from './useLoad'

type TermRoute = Extract<Route, { kind: 'term' }>

/** How many units with the rendering show at a time. */
const PAGE = 100

const FILTER_KEYS = ['family', 'group', 'chapter', 'character', 'event', 'origin', 'q'] as const

/** The filters of an address. */
const filterOf = (r: TermRoute): TermFilter => ({ family: r.family, group: r.group, chapter: r.chapter, character: r.character, event: r.event, origin: r.origin, q: r.q })

/** Whether a unit stays under the filters: every one that is set has to hold. */
export function keepUnit(u: GlossaryUnit, f: TermFilter): boolean {
  if (f.family !== '' && u.family !== f.family) return false
  if (f.group !== '' && u.top !== f.group) return false
  if (f.chapter !== '' && !(u.tag?.kind === 'chapter' && u.tag.value === f.chapter)) return false
  if (f.character !== '' && !(u.tag?.kind === 'character' && u.tag.value === f.character)) return false
  if (f.event !== '' && !(u.tag?.kind === 'event' && u.tag.value === f.event)) return false
  if (f.origin !== '' && u.origin !== f.origin) return false
  const needle = f.q.trim().toLowerCase()
  return needle === '' || [u.kr, u.zh, u.note ?? '', u.key].some((x) => x.toLowerCase().includes(needle))
}

/** A regular expression that marks each of the terms, the longest first; undefined where there is none. */
function termsMark(terms: string[]): RegExp | undefined {
  const marks = [...new Set(terms)].filter((t) => t !== '').sort((a, b) => b.length - a.length).map((t) => termMark(t)?.source)
  return marks.length === 0 ? undefined : new RegExp(marks.join('|'), 'gu')
}

const unique = (xs: string[]) => [...new Set(xs)].sort()

/** The distinct tags of one kind among the units, chapters by number, the others by label. */
function tagOptions(units: GlossaryUnit[], kind: FileTag['kind']): FileTag[] {
  const seen = new Map<string, FileTag>()
  for (const u of units) if (u.tag?.kind === kind && !seen.has(u.tag.value)) seen.set(u.tag.value, u.tag)
  const tags = [...seen.values()]
  if (kind === 'chapter') return tags.sort((a, b) => Number(a.value) - Number(b.value))
  return tags.sort((a, b) => (a.label < b.label ? -1 : a.label > b.label ? 1 : 0))
}

/** Goes to the entry with other filters; a change of the text replaces the entry so that typing leaves one step in the history. */
function go(route: TermRoute, change: Partial<TermFilter>, replace = false): void {
  const href = termHref(route.id, { ...filterOf(route), ...change })
  if (replace) location.replace(href)
  else location.hash = href
}

function Select({ name, label, value, options, onChange }: { name: string; label: string; value: string; options: { value: string; label: string }[]; onChange: (v: string) => void }) {
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

/** The top of the page: what the entry says, how much it is used where, and where else to look at it. */
function Top({ e, hard }: { e: ScopeEntry; hard: string }) {
  const termgroup = e.id.replace(/^glossary\//, 'termgroup/')
  return (
    <section class="tv-top">
      <p class="tv-back">
        <a href="#/g">← 詞彙</a>
      </p>
      <h1>
        <span lang={LANG_ATTR.kr}>{e.kr}</span> → <span lang={LANG_ATTR.zh}>{e.zh}</span>
      </h1>
      <dl class="tv-facts">
        {e.also.length > 0 && (
          <>
            <dt>也可</dt>
            <dd lang={LANG_ATTR.zh}>{e.also.join('、')}</dd>
          </>
        )}
        <dt>檢查</dt>
        <dd>
          {e.check === 'hard' ? 'hard' : 'warn'}
          {e.hard_candidate && <span class="gl-hard" title={hard}>可考慮退回</span>}
        </dd>
        <dt>範圍</dt>
        <dd>{scopeSummary(e).join('；')}</dd>
        <dt>應用率</dt>
        <dd>
          {percent(e.rate)}（{e.used} / {e.n}）
        </dd>
        {e.notes !== '' && (
          <>
            <dt>註記</dt>
            <dd>{e.notes}</dd>
          </>
        )}
      </dl>
      <div class="gl-table-wrap">
        <table class="sc-table tv-groups">
          <thead>
            <tr>
              <th scope="col">群組</th>
              <th scope="col">出現</th>
              <th scope="col">用了</th>
              <th scope="col">應用率</th>
            </tr>
          </thead>
          <tbody>
            {e.groups.map((g) => (
              <tr key={g.name}>
                <th scope="row">{g.name}</th>
                <td>{g.n}</td>
                <td>{g.used}</td>
                <td>{g.n === 0 ? '0%' : `${Math.round((g.used / g.n) * 100)}%`}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p class="tv-links">
        <a href={reviewHref('warnings', { q: termgroup })}>審核頁的警告</a>
        <a href={reviewHref('scope', { id: e.id })}>詞彙範圍的條目</a>
      </p>
      <IdCopy id={e.id} source={`${e.kr} → ${e.zh}`} />
    </section>
  )
}

function Filters({ route, units }: { route: TermRoute; units: GlossaryUnit[] }) {
  const tagOpts = (kind: FileTag['kind']) => tagOptions(units, kind).map((t) => ({ value: t.value, label: t.label }))
  const plain = (xs: string[]) => unique(xs).map((v) => ({ value: v, label: v }))
  return (
    <div class="rv-tools tv-filters">
      <input
        type="search"
        name="unit-q"
        aria-label="篩選單位"
        placeholder="篩選 KR、TW、註記、檔名"
        value={route.q}
        onInput={(e) => go(route, { q: e.currentTarget.value }, true)}
      />
      <Select name="family" label="族群" value={route.family} options={plain(units.map((u) => u.family))} onChange={(v) => go(route, { family: v })} />
      <Select name="group" label="分群" value={route.group} options={plain(units.map((u) => u.top))} onChange={(v) => go(route, { group: v })} />
      <Select name="chapter" label="章節" value={route.chapter} options={tagOpts('chapter')} onChange={(v) => go(route, { chapter: v })} />
      <Select name="character" label="角色" value={route.character} options={tagOpts('character')} onChange={(v) => go(route, { character: v })} />
      <Select name="event" label="活動" value={route.event} options={tagOpts('event')} onChange={(v) => go(route, { event: v })} />
      <Select name="origin" label="來源" value={route.origin} options={plain(units.map((u) => u.origin))} onChange={(v) => go(route, { origin: v })} />
    </div>
  )
}

function MissingUnit({ u, mark }: { u: GlossaryUnit; mark: RegExp | undefined }) {
  return (
    <li class={u.accepted ? 'tv-unit rv-tunit is-confirmed' : 'tv-unit rv-tunit'}>
      <div class="rv-tunit-head">
        <UnitLink unitKey={u.key} />
        <OriginBadge origin={u.origin} />
        {u.id !== undefined && <IdCopy id={u.id} source={u.zh} />}
      </div>
      <ConfirmedLine confirmed={u.accepted} />
      <dl class="rv-src">
        <Row label="KR" lang="kr" text={u.kr} {...(mark ? { mark } : {})} />
        <Row label="TW" lang="zh" text={u.zh} />
      </dl>
      {u.note !== undefined && <p class="rv-note-text">{u.note}</p>}
    </li>
  )
}

function UsedUnit({ u, krMark, zhMark }: { u: GlossaryUnit; krMark: RegExp | undefined; zhMark: RegExp | undefined }) {
  return (
    <li class="tv-unit rv-tunit">
      <div class="rv-tunit-head">
        <UnitLink unitKey={u.key} />
        <OriginBadge origin={u.origin} />
      </div>
      <dl class="rv-src">
        <Row label="KR" lang="kr" text={u.kr} {...(krMark ? { mark: krMark } : {})} />
        <Row label="TW" lang="zh" text={u.zh} {...(zhMark ? { mark: zhMark } : {})} />
      </dl>
      {u.note !== undefined && <p class="rv-note-text">{u.note}</p>}
    </li>
  )
}

/** The units with the rendering, 100 at a time. It is keyed by the filters, so that a change of them starts again at 100. */
function UsedList({ units, krMark, zhMark }: { units: GlossaryUnit[]; krMark: RegExp | undefined; zhMark: RegExp | undefined }) {
  const [limit, setLimit] = useState(PAGE)
  const shown = Math.min(units.length, limit)
  return (
    <>
      {units.length === 0 ? (
        <p class="rv-empty">沒有符合的單位</p>
      ) : (
        <ul class="rv-tunits tv-list">
          {units.slice(0, shown).map((u) => (
            <UsedUnit key={u.key} u={u} krMark={krMark} zhMark={zhMark} />
          ))}
        </ul>
      )}
      {shown < units.length && (
        <button type="button" class="rv-more" onClick={() => setLimit(shown + PAGE)}>
          再顯示 {PAGE} 個（共 {units.length}）
        </button>
      )}
    </>
  )
}

/** The units the entry file holds, under the filters of the address. */
function Units({ route, entry, file }: { route: TermRoute; entry: ScopeEntry; file: GlossaryFile }) {
  const filter = filterOf(route)
  const filterKey = FILTER_KEYS.map((k) => filter[k]).join('\u0000')
  const missing = useMemo(() => file.missing.filter((u) => keepUnit(u, filter)), [file, filterKey])
  const used = useMemo(() => file.used.filter((u) => keepUnit(u, filter)), [file, filterKey])
  const all = useMemo(() => [...file.missing, ...file.used], [file])
  const krMark = useMemo(() => termMark(entry.kr), [entry.kr])
  const zhMark = useMemo(() => termsMark([entry.zh, ...entry.also]), [entry.zh, entry.also])
  const list = useRef<HTMLUListElement>(null)
  const [said, setSaid] = useState('')

  // The unconfirmed units under the filters, as ID「TW」→ a line each: what to paste to the conversation to decide on them.
  const copy = async () => {
    const lines = missing.filter((u) => u.accepted === undefined && u.id !== undefined).map((u) => copyText(u.id ?? '', stripTags(u.zh)))
    if (lines.length === 0) {
      setSaid('沒有未確認的單位可複製')
      return
    }
    if (!list.current) return
    const result = await copyToClipboard(lines.join('\n'), list.current)
    setSaid(result === 'copied' ? `已複製 ${lines.length} 個單位，貼到對話中再加上你的決定` : '瀏覽器不讓複製，已選取單位，請按 Ctrl/⌘+C')
  }

  return (
    <>
      <Filters route={route} units={all} />
      <section class="tv-missing">
        <h2>沒用譯名 {missing.length} 項</h2>
        <div class="rv-tools">
          <button type="button" class="rv-copy" onClick={copy}>
            複製這些單位
          </button>
          <span class="rv-said" role="status">
            {said}
          </span>
        </div>
        {missing.length === 0 ? (
          <p class="rv-empty">沒有符合的單位</p>
        ) : (
          <ul class="rv-tunits tv-list" ref={list}>
            {missing.map((u) => (
              <MissingUnit key={u.key} u={u} mark={krMark} />
            ))}
          </ul>
        )}
      </section>
      <section class="tv-used">
        <h2>用了譯名 {used.length} 項</h2>
        <UsedList key={filterKey} units={used} krMark={krMark} zhMark={zhMark} />
      </section>
    </>
  )
}

/** The entry file, once loaded: the units, or what went wrong. */
function Body({ route, entry }: { route: TermRoute; entry: ScopeEntry }) {
  const loaded = useLoad(route.id, () => loadGlossaryEntry(route.id))
  if (loaded === undefined) return <p role="status">單位載入中…</p>
  if (loaded.status === 'error') return <Notice error={loaded.error} />
  return <Units route={route} entry={entry} file={loaded.data} />
}

/** One entry of the glossary: its content, and every unit it hits, with the units that lack the rendering first. */
export function TermView({ route, settings }: { route: TermRoute; settings: Settings }) {
  const loaded = useLoad('glossary', loadGlossary)
  const shown = useMemo(() => ({ langs: settings.langs, raw: settings.rawTags }), [settings.langs, settings.rawTags])
  const data = loaded?.status === 'ready' ? loaded.data : undefined
  const entry = data?.entries.find((e) => e.id === route.id)
  return (
    <ShownText.Provider value={shown}>
      <div class="glossary term">
        {loaded?.status === 'error' && <Notice error={loaded.error} />}
        {loaded === undefined && <p role="status">詞彙載入中…</p>}
        {data && !entry && (
          <>
            <p class="tv-back">
              <a href="#/g">← 詞彙</a>
            </p>
            <p class="rv-missing" role="status">
              找不到詞條 <code>{route.id}</code>：它可能已被改掉或刪除。
            </p>
          </>
        )}
        {data && entry && (
          <>
            <Top e={entry} hard={hardSentence(data.thresholds)} />
            <Body route={route} entry={entry} />
          </>
        )}
      </div>
    </ShownText.Provider>
  )
}
