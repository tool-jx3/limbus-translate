import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { LANG_ATTR, type ScopeEntry, type ScopeGroup, type ScopeKind, type ScopeProposal, type ScopeReport, type ScopeThresholds } from '../types'
import { ConfirmedLine, IdCopy, UnitLink } from './parts'

const ENTRY_PAGE = 50

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
export function groupClass(g: { n: number; used: number }, t: ScopeThresholds): string {
  if (g.n < t.min_n) return '太少'
  const rate = g.used / g.n
  if (rate >= t.used) return '有用'
  if (rate <= t.not_used) return '沒用'
  return g.n - g.used >= t.min_n ? '混用' : '中間'
}

const CLASS_KEY: Record<string, string> = { 有用: 'used', 沒用: 'unused', 混用: 'mixed', 太少: 'few', 中間: 'mid' }
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
                <td>
                  <span class={`sc-class sc-class-${CLASS_KEY[cls]}`}>{cls}</span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function Examples({ p }: { p: ScopeProposal }) {
  if (p.examples.length === 0) return null
  return (
    <details class="sc-examples">
      <summary>例子（{p.examples.length}）</summary>
      <ul>
        {p.examples.map((e) => (
          <li key={e.key}>
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

function ProposalCard({ p, entry, t, target }: { p: ScopeProposal; entry: ScopeEntry | undefined; t: ScopeThresholds; target: boolean }) {
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
        {p.flagged && <span class="sc-flag">失去檢查偏多</span>}
        <IdCopy id={p.id} source={title} />
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
      <Examples p={p} />
    </article>
  )
}

function EntryRow({ e, t }: { e: ScopeEntry; t: ScopeThresholds }) {
  const n = e.groups.reduce((s, g) => s + g.n, 0)
  const used = e.groups.reduce((s, g) => s + g.used, 0)
  return (
    <details class="sc-entry" data-id={e.id}>
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
          複合詞 {e.compound.n} 個單位，用了譯名 {e.compound.used}。未用譯名的單位：有說明 {e.explained}、沒說明 {e.unexplained}。
        </p>
        {e.notes !== '' && <p class="sc-line">{e.notes}</p>}
        <IdCopy id={e.id} source={`${e.kr} → ${e.zh}`} />
      </div>
    </details>
  )
}

function AllEntries({ entries, t }: { entries: ScopeEntry[]; t: ScopeReport['thresholds'] }) {
  const [q, setQ] = useState('')
  const [limit, setLimit] = useState(ENTRY_PAGE)
  const needle = q.trim().toLowerCase()
  const shown = useMemo(
    () => (needle === '' ? entries : entries.filter((e) => [e.kr, e.zh, ...e.also].some((x) => x.toLowerCase().includes(needle)))),
    [entries, needle],
  )
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
        {shown.slice(0, limit).map((e) => (
          <EntryRow key={e.id} e={e} t={t} />
        ))}
      </div>
      {limit < shown.length && (
        <button type="button" class="rv-more sc-more" onClick={() => setLimit(limit + ENTRY_PAGE)}>
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
  const found = filter.id !== undefined && shown.some((p) => p.id === filter.id)
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
      {SECTIONS.map((s) => {
        const items = shown.filter((p) => s.kinds.includes(p.kind))
        return (
          <section class="sc-section" key={s.name}>
            <h2>
              {s.name} <span class="rv-n">{items.length} 項</span>
            </h2>
            <p class="sc-desc">{s.desc}</p>
            {items.length === 0 && <p class="rv-empty">沒有建議</p>}
            <div class="rv-list">
              {items.map((p) => (
                <ProposalCard key={p.id} p={p} entry={byEntry.get(`${p.kr}\u0000${p.zh}`)} t={report.thresholds} target={p.id === filter.id} />
              ))}
            </div>
          </section>
        )
      })}
      <AllEntries entries={report.entries} t={report.thresholds} />
    </div>
  )
}
