import { useRef, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { loadGold } from '../data'
import { useLoad } from '../glossary/useLoad'
import { COPY_REFUSED_HINT, copyToClipboard } from '../review/copy'
import type { GoldData, GoldRow, GoldUnit } from '../types'

/** Where the choices of the reader are kept, by the key of the unit. */
export const STORAGE_KEY = 'lct.gold.v1'

type Verdict = 'correct' | 'wrong' | 'unsure'
/** What the reader chose for a unit: verdict is '' until a choice is made. */
export type Choice = { verdict: Verdict | ''; note: string }
export type Choices = Record<string, Choice>

const VERDICTS: { value: Verdict; label: string }[] = [
  { value: 'correct', label: '正確' },
  { value: 'wrong', label: '有誤' },
  { value: 'unsure', label: '不確定' },
]

function readChoices(): Choices {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const doc: unknown = JSON.parse(raw)
    if (typeof doc !== 'object' || doc === null || Array.isArray(doc)) return {}
    const out: Choices = {}
    for (const [k, v] of Object.entries(doc as Record<string, unknown>)) {
      const c = v as Partial<Choice> | null
      if (c && typeof c === 'object' && VERDICTS.some((x) => x.value === c.verdict)) {
        out[k] = { verdict: c.verdict as Verdict, note: typeof c.note === 'string' ? c.note : '' }
      }
    }
    return out
  } catch {
    return {}
  }
}

function writeChoices(choices: Choices): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(choices))
  } catch {
    // Without storage the choices live as long as the page.
  }
}

/** A key the gold line cannot carry: the line is split on whitespace. */
export function pasteable(key: string): boolean {
  return !/\s/.test(key)
}

/** The lines `gold <key> <verdict> <note>` for the units the reader judged, in the order of the sample; an empty note leaves no trailing space. */
export function goldLines(units: GoldUnit[], choices: Choices): string {
  const lines: string[] = []
  for (const u of units) {
    const c = choices[u.key]
    if (!c || c.verdict === '' || !pasteable(u.key)) continue
    const note = c.note.replace(/\s+/g, ' ').trim()
    lines.push(`gold ${u.key} ${c.verdict}${note ? ` ${note}` : ''}`)
  }
  return lines.join('\n')
}

function pct(r: GoldRow): string {
  return r.accuracy === null ? '—' : `${Math.round(r.accuracy * 100)}%`
}

/**
 * Local choices that differ from what the data already holds: the others were recorded and need no sending.
 * A stale unit keeps its choice even when it equals the data: sending the same verdict again records the new
 * translation and clears the staleness.
 */
export function pending(units: GoldUnit[], choices: Choices): Choices {
  const out: Choices = {}
  for (const u of units) {
    const c = choices[u.key]
    if (c && (u.stale || !(c.verdict === u.verdict && c.note === u.note))) out[u.key] = c
  }
  return out
}

function Report({ data }: { data: GoldData }) {
  const report = data.report
  if (!report) return null
  return (
    <div class="gold-table-wrap">
      <table class="gold-report">
        <thead>
          <tr>
            <th>族群</th>
            <th>來源</th>
            <th>樣本</th>
            <th>已判定</th>
            <th>正確</th>
            <th>有誤</th>
            <th>不確定</th>
            <th>正確率</th>
            <th>過時</th>
          </tr>
        </thead>
        <tbody>
          {[...report.rows, ...report.families.map((f) => ({ ...f, origin: '全部' })), { ...report.total, family: '合計', origin: '' }].map((r, i) => (
            <tr key={i}>
              <td>{r.family}</td>
              <td>{r.origin}</td>
              <td>{r.sampled}</td>
              <td>{r.judged}</td>
              <td>{r.correct}</td>
              <td>{r.wrong}</td>
              <td>{r.unsure}</td>
              <td>{pct(r)}</td>
              <td>{r.stale}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Unit({ u, choice, onChange }: { u: GoldUnit; choice: Choice | undefined; onChange: (patch: Partial<Choice>) => void }) {
  const verdict = choice?.verdict ?? u.verdict
  const note = choice ? choice.note : u.note
  return (
    <li class="gold-unit" data-key={u.key}>
      <div class="gold-meta">
        {u.key} · {u.origin}
        {u.stale && <span class="gold-stale"> · 已過時（譯文在判定後改過）</span>}
        {choice && choice.verdict !== '' && pasteable(u.key) && <span class="gold-tocopy"> · 待複製</span>}
      </div>
      <p class="gold-text" lang="ko">KR　{u.kr}</p>
      {u.jp !== undefined && <p class="gold-text" lang="ja">JP　{u.jp}</p>}
      {u.en !== undefined && <p class="gold-text" lang="en">EN　{u.en}</p>}
      <p class="gold-text" lang="zh-TW">TW　{u.zh}</p>
      {u.shipped !== undefined && <p class="gold-text gold-meta gold-shipped">實際打包：{u.shipped}</p>}
      {!pasteable(u.key) ? (
        <p class="gold-meta gold-nopaste">此單位的 key 含空白，無法用複製貼上判定。</p>
      ) : (
        <>
      <div class="gold-verdicts" role="radiogroup" aria-label={`判定 ${u.key}`}>
        {VERDICTS.map((v) => (
          <label key={v.value}>
            <input
              type="radio"
              name={`gold-${u.key}`}
              value={v.value}
              checked={verdict === v.value}
              onClick={() => onChange({ verdict: v.value })}
            />
            {v.label}
          </label>
        ))}
      </div>
      <input
        type="text"
        class="gold-note"
        aria-label={`說明 ${u.key}`}
        placeholder="說明"
        value={note}
        disabled={verdict === ''}
        onInput={(e) => onChange({ note: e.currentTarget.value })}
      />
      {verdict === '' && <span class="gold-meta">先選判定，說明才會一起複製</span>}
        </>
      )}
    </li>
  )
}

function Loaded({ data }: { data: GoldData }) {
  const [choices, setChoices] = useState<Choices>(() => {
    const kept = pending(data.units, readChoices())
    writeChoices(kept)
    return kept
  })
  const [refused, setRefused] = useState(false)
  const [said, setSaid] = useState('')
  const box = useRef<HTMLTextAreaElement>(null)
  // The latest choices, for a copy that comes before the next render.
  const latest = useRef(choices)

  const change = (u: GoldUnit, patch: Partial<Choice>) => {
    // A patch lands on the choice so far, or on the judgment the data holds.
    const base: Choice = latest.current[u.key] ?? { verdict: u.verdict, note: u.note }
    const next = { ...latest.current, [u.key]: { ...base, ...patch } }
    latest.current = next
    writeChoices(next)
    setChoices(next)
  }

  const clear = () => {
    latest.current = {}
    writeChoices({})
    setChoices({})
    setRefused(false)
    setSaid('')
  }

  const families: string[] = []
  for (const u of data.units) if (!families.includes(u.family)) families.push(u.family)

  const missing = data.report?.missing ?? data.missing ?? 0
  const text = goldLines(data.units, choices)
  const copy = async () => {
    const text = goldLines(data.units, latest.current)
    if (box.current) box.current.value = text
    const result = await copyToClipboard(text, box.current!)
    setRefused(result !== 'copied')
    setSaid(result === 'copied' ? (text === '' ? '還沒有判定可複製' : '已複製判定') : '')
  }

  return (
    <section class="gold">
      <h1>基準樣本</h1>
      {missing > 0 && <p class="gold-missing">有 {missing} 個樣本單位在打包結果裡找不到，不在下方的樣本清單；報告仍計入。</p>}
      <Report data={data} />
      {data.units.length === 0 && <p>還沒有基準樣本。</p>}
      {families.map((f) => (
        <section key={f} class="gold-family" data-family={f}>
          <h2>{f}</h2>
          <ul class="gold-units">
            {data.units
              .filter((u) => u.family === f)
              .map((u) => (
                <Unit key={u.key} u={u} choice={choices[u.key]} onChange={(patch) => change(u, patch)} />
              ))}
          </ul>
        </section>
      ))}
      <div class="pv-actions">
        <button type="button" class="gold-copy" onClick={copy}>
          複製判定
        </button>
        <button type="button" class="gold-clear" onClick={clear}>
          清除本頁選擇
        </button>
      </div>
      <p class={refused ? 'tv-copyhint' : 'rv-said'} role="status">
        {refused ? COPY_REFUSED_HINT : said}
      </p>
      <textarea
        ref={box}
        class={refused ? 'tv-copybox' : 'tv-copybox is-hidden'}
        readOnly
        aria-label="判定"
        {...(refused ? {} : { tabIndex: -1, 'aria-hidden': 'true' as const })}
        rows={8}
        value={text}
      />
    </section>
  )
}

/** The gold sample by family, to judge: the report above, a verdict and a note for each unit, and the lines to paste to the gold-review skill. */
export function GoldView() {
  const state = useLoad('gold', loadGold)
  if (!state) return <p role="status">資料載入中…</p>
  if (state.status === 'error') return <Notice error={state.error} />
  return <Loaded data={state.data} />
}
