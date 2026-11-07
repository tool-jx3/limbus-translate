import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { reviewHref } from '../route'
import { LANG_ATTR, type ScopeEntry, type ScopeExample, type ScopeGroup, type ScopeKind, type ScopeProposal, type ScopeReport, type ScopeThresholds } from '../types'
import { ConfirmedLine, IdCopy, UnitLink } from './parts'

const ENTRY_PAGE = 50
/** How many 需要人看 proposals show before the button. */
const MIXED_TOP = 10

/** The sections of the page, in order: the kinds each holds, and what they are called. */
const SECTIONS: { name: string; desc: string; kinds: ScopeKind[] }[] = [
  { name: '排除檔案', desc: '這些檔案的譯文常不用這個譯名；建議把它們加進 not_files，不再檢查。', kinds: ['not-files'] },
  { name: '限定檔案', desc: '只有這些檔案幾乎都用這個譯名；建議把條目限定在它們（files）。', kinds: ['files'] },
  { name: '詞中詞', desc: '詞中詞（KR 詞出現在更長的詞裡）的判斷：建議改成 substring（詞中也比對）或 word（只比對整詞）。', kinds: ['substring', 'word'] },
  { name: '需要人看', desc: '有些群組只用了一部分，沒有乾淨的切法；請看群組表和例子再決定。', kinds: ['mixed'] },
  { name: '常被說明偏離', desc: '譯者在註記裡常說明為什麼不用這個譯名；也許該加 also 或改譯名。', kinds: ['explained'] },
]

const KIND_LABEL: Record<ScopeKind, string> = {
  'not-files': '排除檔案',
  files: '限定檔案',
  substring: '詞中詞・substring',
  word: '詞中詞・word',
  mixed: '需要人看',
  explained: '常被說明偏離',
}

/** How a group reads by the thresholds: 有用, 沒用, 混用 (enough units lack the rendering), 太少 (too few units to say), 中間 (between, with few lacking). */
export type GroupClass = 'uses' | 'notUse' | 'mixed' | 'few' | 'between'

export const CLASS_LABEL: Record<GroupClass, string> = { uses: '有用', notUse: '沒用', mixed: '混用', few: '太少', between: '中間' }

export function groupClass(g: { n: number; used: number }, t: ScopeThresholds): GroupClass {
  if (g.n < t.min_n) return 'few'
  const rate = g.used / g.n
  if (rate >= t.used) return 'uses'
  if (rate <= t.not_used) return 'notUse'
  return g.n - g.used >= t.min_n ? 'mixed' : 'between'
}

/** Why a group is read as 太少 or 中間: it is not counted by the proposals. */
function classTitle(c: GroupClass, t: ScopeThresholds): string | undefined {
  if (c === 'few') return `出現不到 ${t.min_n} 個單位，不計入建議`
  if (c === 'between') return `未用譯名的不到 ${t.min_n} 個單位，不計入建議`
  return undefined
}

const pct = (share: number) => `${Math.round(share * 100)}%`

/** What a card with 失去檢查偏多 loses, by the thresholds. */
const flagText = (t: ScopeThresholds) => `不再檢查的用了譯名的單位至少 ${t.lost_units} 個，或佔 ${pct(t.lost_share)} 以上；預設不採用`

/** The content of a proposal that goes with its id when copied: its kind, patterns and match, so that a skill that finds the id in older local data can see the proposal has changed. */
export function proposalDetail(p: ScopeProposal): string {
  const parts = [`kind: ${p.kind}`]
  if (p.patterns.length > 0) parts.push(`patterns: ${p.patterns.join(', ')}`)
  if (p.match !== '') parts.push(`match: ${p.match}`)
  return `（${parts.join('；')}）`
}

/** The thresholds the labels and proposals follow, in words and with the values of the data. */
function Legend({ t }: { t: ScopeThresholds }) {
  return (
    <ul class="sc-legend" aria-label="門檻">
      <li>群組：族群（family:）與頂層分群（group:）；少於 {t.min_n} 個單位的群組不計入建議。</li>
      <li>
        有用：{pct(t.used)} 以上的單位用了譯名；沒用：{pct(t.not_used)} 以下。
      </li>
      <li>
        混用：介於 {pct(t.not_used)} 到 {pct(t.used)} 之間，且沒用譯名的單位至少 {t.min_n} 個；不到 {t.min_n} 個算中間，不計入建議。
      </li>
      <li>
        詞中詞：複合詞單位至少 {t.min_n} 個且 {pct(t.used)} 以上用了譯名，建議 substring；已是 substring 而 {pct(t.not_used)} 以下用了譯名，建議 word。
      </li>
      <li>常被說明偏離：沒用譯名而註記說明的單位至少 {t.explained} 個。</li>
      <li>失去檢查偏多：不再檢查的用了譯名的單位至少 {t.lost_units} 個，或佔該條目用了譯名的 {pct(t.lost_share)} 以上；這樣的建議預設不採用。</li>
    </ul>
  )
}

/** The units of an entry that do not use its rendering (each hit is in a family group and a node group, so the groups would count it twice). */
const nonUsing = (e: ScopeEntry | undefined) => (e ? e.n - e.used : 0)
const percent = (used: number, n: number) => (n === 0 ? 0 : Math.round((used / n) * 100))

/** The effect of a proposal as a line: only the numbers that are not zero; '' when none is. */
export function effectText(p: ScopeProposal): string {
  return [
    ['移除警告', p.warn_removed],
    ['新增警告', p.warn_added],
    ['少檢查', p.used_lost],
    ['多檢查', p.used_gained],
  ]
    .filter(([, n]) => (n as number) > 0)
    .map(([label, n]) => `${label} ${n}`)
    .join('、')
}

/** The groups of an entry: name, units, units with the rendering, the rate with a meter, and how it reads. */
function GroupTable({ groups, t }: { groups: ScopeGroup[]; t: ScopeThresholds }) {
  if (groups.length === 0) return <p class="rv-empty">沒有命中的單位</p>
  return (
    <div class="sc-table-wrap">
      <table class="sc-table">
        <thead>
          <tr>
            <th scope="col">群組</th>
            <th scope="col">出現</th>
            <th scope="col">用了譯名</th>
            <th scope="col">比例</th>
            <th scope="col" title="沒用譯名的單位中，可比對的 EN／JP 都含條目的 en／jp">同義</th>
            <th scope="col" title="沒用譯名的單位中，可比對的 EN／JP 都不含條目的 en／jp">他義</th>
            <th scope="col">判斷</th>
          </tr>
        </thead>
        <tbody>
          {groups.map((g) => {
            const cls = groupClass(g, t)
            const pct = percent(g.used, g.n)
            return (
              <tr key={g.name}>
                <th scope="row">{g.name}</th>
                <td>{g.n}</td>
                <td>{g.used}</td>
                <td>
                  <span class="sc-rate">
                    <span class="sc-meter" aria-hidden="true">
                      <i style={{ width: `${pct}%` }} />
                    </span>
                    {pct}%
                  </span>
                </td>
                <td>{g.same}</td>
                <td>{g.other}</td>
                <td title={classTitle(cls, t)}>
                  <span class={`sc-class sc-class-${cls}`}>{CLASS_LABEL[cls]}</span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function Examples({ items, label = '例子' }: { items: ScopeExample[]; label?: string }) {
  if (items.length === 0) return null
  return (
    <details class="sc-examples">
      <summary>
        {label}（{items.length}）
      </summary>
      <ul>
        {items.map((e) => (
          <li key={`${e.group}\u0000${e.key}`}>
            <span class="rv-where">{e.group}・</span>
            <UnitLink unitKey={e.key} />
            <div lang={LANG_ATTR.kr}>{e.kr}</div>
            <div lang={LANG_ATTR.zh}>{e.zh}</div>
          </li>
        ))}
      </ul>
    </details>
  )
}

function ProposalCard({ p, entry, t, target, all }: { p: ScopeProposal; entry: ScopeEntry | undefined; t: ScopeThresholds; target: boolean; all: boolean }) {
  const cls = ['rv-card', p.declined ? 'is-confirmed' : '', target ? 'is-target' : ''].filter(Boolean).join(' ')
  const effect = effectText(p)
  const title = `${p.kr} → ${p.zh}`
  return (
    <article class={cls} data-id={p.id}>
      <div class="rv-head">
        <span class="rv-kind rv-kind-sugg">{KIND_LABEL[p.kind]}</span>
        <h3 class="sc-title">
          <span lang={LANG_ATTR.kr}>{p.kr}</span> → <span lang={LANG_ATTR.zh}>{p.zh}</span>
        </h3>
        {p.flagged && (
          <span class="sc-flag" title={flagText(t)}>
            失去檢查偏多
          </span>
        )}
        <IdCopy id={p.id} source={title} detail={proposalDetail(p)} />
      </div>
      <ConfirmedLine confirmed={p.declined} verb="已否決" />
      {p.patterns.length > 0 && (
        <p class="sc-patterns">
          樣式：
          {p.patterns.map((x) => (
            <code key={x}>{x}</code>
          ))}
        </p>
      )}
      {p.match !== '' && <p class="sc-line">建議 match：<code>{p.match}</code></p>}
      {p.mixed_groups.length > 0 && <p class="sc-line">另有混用：{p.mixed_groups.join('、')}</p>}
      {p.kind === 'mixed' && p.mixed_groups.length === 0 && <p class="sc-line">各群組都很少用這個譯名。</p>}
      {effect !== '' && <p class="sc-effect">{effect}</p>}
      {entry && <GroupTable groups={entry.groups} t={t} />}
      <Examples items={p.examples} />
      {entry && (
        <a class="sc-entry-link" href={reviewHref('scope', { id: entry.id, all })}>
          看條目
        </a>
      )}
    </article>
  )
}

function EntryRow({ e, t, target }: { e: ScopeEntry; t: ScopeThresholds; target: boolean }) {
  const n = e.groups.reduce((s, g) => s + g.n, 0)
  const used = e.groups.reduce((s, g) => s + g.used, 0)
  return (
    <details class={target ? 'sc-entry is-target' : 'sc-entry'} data-id={e.id} open={target}>
      <summary>
        <span class="sc-entry-name">
          <span lang={LANG_ATTR.kr}>{e.kr}</span> → <span lang={LANG_ATTR.zh}>{e.zh}</span>
        </span>
        <span class="sc-entry-total">
          {used}/{n}
        </span>
        <span class="sc-entry-meta">
          {e.also.length > 0 && <span>also：{e.also.join('、')}</span>}
          {e.files.length > 0 && <span>files：{e.files.join('、')}</span>}
          {e.not_files.length > 0 && <span>not_files：{e.not_files.join('、')}</span>}
          {e.match !== '' && <span>match：{e.match}</span>}
        </span>
      </summary>
      <div class="sc-entry-body">
        <GroupTable groups={e.groups} t={t} />
        <p class="sc-line">
          複合詞 {e.compound.n} 個單位，用了譯名 {e.compound.used}。未用譯名的單位：有說明 {e.explained}、沒說明 {e.unexplained}。同義 {e.same}、他義 {e.other}、<span title="沒有可比對的 EN／JP，或 EN／JP 判斷不一致">未知 {e.unknown}</span>。
        </p>
        {e.notes !== '' && <p class="sc-line">{e.notes}</p>}
        <Examples items={e.examples} label="沒用譯名的例子" />
        <IdCopy id={e.id} source={`${e.kr} → ${e.zh}`} />
      </div>
    </details>
  )
}

function AllEntries({ entries, t, id }: { entries: ScopeEntry[]; t: ScopeReport['thresholds']; id: string | undefined }) {
  const [q, setQ] = useState('')
  const [limit, setLimit] = useState(ENTRY_PAGE)
  const needle = q.trim().toLowerCase()
  const shown = useMemo(
    () => (needle === '' ? entries : entries.filter((e) => e.id === id || [e.kr, e.zh, ...e.also].some((x) => x.toLowerCase().includes(needle)))),
    [entries, needle, id],
  )
  const at = id === undefined ? -1 : shown.findIndex((e) => e.id === id)
  const count = Math.min(shown.length, Math.max(limit, at + 1))
  return (
    <section class="sc-all">
      <h2>
        全部條目 <span class="rv-n">{entries.length} 條</span>
      </h2>
      <div class="rv-tools">
        <input
          type="search"
          name="entry-q"
          placeholder="搜尋 KR、譯名、also"
          aria-label="搜尋條目"
          value={q}
          onInput={(e) => {
            setQ(e.currentTarget.value)
            setLimit(ENTRY_PAGE)
          }}
        />
        <span class="rv-count">符合 {shown.length} 條</span>
      </div>
      {shown.length === 0 && <p class="rv-empty">沒有符合的條目</p>}
      <div class="sc-entries">
        {shown.slice(0, count).map((e) => (
          <EntryRow key={e.id} e={e} t={t} target={e.id === id} />
        ))}
      </div>
      {count < shown.length && (
        <button type="button" class="rv-more sc-more" onClick={() => setLimit(count + ENTRY_PAGE)}>
          顯示更多（共 {shown.length}）
        </button>
      )}
    </section>
  )
}

export interface ScopeFilter {
  q: string
  all: boolean
  id: string | undefined
}

/** The proposals that stay after the text and the 顯示已確認 switch; a declined one stays when the address names it. */
export function filterProposals(items: ScopeProposal[], f: ScopeFilter): ScopeProposal[] {
  const needle = f.q.trim().toLowerCase()
  return items.filter(
    (p) =>
      !(p.declined && !f.all && p.id !== f.id) &&
      (needle === '' || [p.id, p.kr, p.zh, ...p.patterns].some((x) => x.toLowerCase().includes(needle))),
  )
}

/** The body of the 詞彙範圍 tab: the proposals by section, then all the entries. */
export function ScopeBody({ report, filter }: { report: ScopeReport; filter: ScopeFilter }) {
  const shown = useMemo(() => filterProposals(report.proposals, filter), [report, filter.q, filter.all, filter.id])
  const byEntry = useMemo(() => new Map(report.entries.map((e) => [`${e.kr}\u0000${e.zh}`, e])), [report])
  const box = useRef<HTMLDivElement>(null)
  const found = filter.id !== undefined && (shown.some((p) => p.id === filter.id) || report.entries.some((e) => e.id === filter.id))
  // 需要人看 is expanded only for the filter it was expanded under, so a new search or the 顯示已確認 switch folds it back in the same render.
  const filterKey = `${filter.q}\u0000${filter.all}`
  const [expandedKey, setExpandedKey] = useState<string>()
  const allMixed = expandedKey === filterKey
  useEffect(() => {
    box.current?.querySelector('.is-target')?.scrollIntoView?.({ block: 'center' })
  }, [filter.id, found])
  return (
    <div class="rv-body" ref={box}>
      {filter.id !== undefined && !found && (
        <p class="rv-missing" role="status">
          找不到項目 <code>{filter.id}</code>：它可能已解決，或被目前的篩選擋住了。
        </p>
      )}
      <Legend t={report.thresholds} />
      {SECTIONS.map((s) => {
        let items = shown.filter((p) => s.kinds.includes(p.kind))
        const total = items.length
        const collapsible = s.kinds.includes('mixed')
        if (collapsible && !allMixed && total > MIXED_TOP) {
          const rank = (p: ScopeProposal) => nonUsing(byEntry.get(`${p.kr}\u0000${p.zh}`))
          const top = new Set([...items].sort((a, b) => rank(b) - rank(a)).slice(0, MIXED_TOP))
          items = items.filter((p) => top.has(p) || p.id === filter.id)
        }
        const sorted = collapsible ? [...items].sort((a, b) => nonUsing(byEntry.get(`${b.kr}\u0000${b.zh}`)) - nonUsing(byEntry.get(`${a.kr}\u0000${a.zh}`))) : items
        return (
          <section class="sc-section" key={s.name}>
            <h2>
              {s.name} <span class="rv-n">{total} 項</span>
            </h2>
            <p class="sc-desc">{s.desc}</p>
            {total === 0 && <p class="rv-empty">沒有建議</p>}
            <div class="rv-list">
              {sorted.map((p) => (
                <ProposalCard key={p.id} p={p} entry={byEntry.get(`${p.kr}\u0000${p.zh}`)} t={report.thresholds} target={p.id === filter.id} all={filter.all} />
              ))}
            </div>
            {collapsible && sorted.length < total && (
              <button type="button" class="rv-more sc-more-mixed" onClick={() => setExpandedKey(filterKey)}>
                顯示更多（共 {total}）
              </button>
            )}
          </section>
        )
      })}
      <AllEntries entries={report.entries} t={report.thresholds} id={filter.id} />
    </div>
  )
}
