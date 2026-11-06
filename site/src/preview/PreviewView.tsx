import type { ComponentChildren } from 'preact'
import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { loadFile, loadPreviewFile, loadPreviewShared } from '../data'
import { useLoad } from '../glossary/useLoad'
import { COPY_REFUSED_HINT, copyToClipboard } from '../review/copy'
import { fileHref, termHref, type Route } from '../route'
import type { BatchJSON, FileData, Origin, PreviewFile, PreviewShared, PreviewUnit, Term } from '../types'
import { COPY_NOT_SENT } from '../view/PreviewLink'
import { OriginBadge } from '../view/OriginBadge'
import { fullText, goJSON, rebuildBatch } from './rebuild'
import { sha8 } from './sha8'

type PreviewRoute = Extract<Route, { kind: 'preview' }>

const VERDICT: Record<string, string> = { reject: '退回', warn: '收下但有警告', accept: '收下' }

interface Issue {
  code: string
  message: string
}

/** The ids (glossary/<sha8 of KR>/<sha8 of ZH>) of the terms, once computed; the preview data does not carry them. */
function useTermIds(terms: readonly Term[]): Map<Term, string> {
  const [ids, setIds] = useState(new Map<Term, string>())
  useEffect(() => {
    let live = true
    Promise.all(terms.map(async (t) => [t, `glossary/${await sha8(t.kr)}/${await sha8(t.zh)}`] as const)).then(
      (pairs) => {
        if (live) setIds(new Map(pairs))
      },
      () => {
        // No digest: the terms stay unlinked.
      },
    )
    return () => {
      live = false
    }
  }, [terms])
  return ids
}

function Back({ path, ptr }: { path: string; ptr: string }) {
  return (
    <p>
      <a class="pv-back" href={fileHref(path, ptr)}>
        ← 回到檔案頁的這個單位
      </a>
    </p>
  )
}

function Section({ title, children }: { title: string; children: ComponentChildren }) {
  return (
    <section class="pv-section">
      <h2>{title}</h2>
      {children}
    </section>
  )
}

/** A line of text that may be long: in a box of its own that scrolls. */
function Box({ children, label }: { children: string; label?: string }) {
  return (
    <pre class="pv-box" tabIndex={0} aria-label={label}>
      {children}
    </pre>
  )
}

function Facts({ rows }: { rows: [string, ComponentChildren][] }) {
  return (
    <dl class="tv-facts pv-facts">
      {rows.map(([k, v]) => (
        <>
          <dt>{k}</dt>
          <dd>{v}</dd>
        </>
      ))}
    </dl>
  )
}

function State({ pu }: { pu: PreviewUnit }) {
  const notes: string[] = []
  if (pu.st.frozen) notes.push('不送翻：凍結檔案')
  if (pu.st.needs_review) notes.push('等人處理，現在不會發出')
  if (pu.st.leased_to) notes.push(`已租給批次 ${pu.st.leased_to}`)
  const rows: [string, ComponentChildren][] = [['待譯', pu.st.pending ? '是' : '否']]
  if (pu.st.attempts !== undefined) rows.push(['已試次數', String(pu.st.attempts)])
  if (pu.st.last_error) rows.push(['上次失敗', pu.st.last_error])
  return (
    <Section title="狀態">
      {notes.map((n) => (
        <p key={n} class="pv-note">
          {n}
        </p>
      ))}
      <Facts rows={rows} />
    </Section>
  )
}

function Composition({ batch, pf, pu, shared }: { batch: BatchJSON; pf: PreviewFile; pu: PreviewUnit; shared: PreviewShared }) {
  const unit = batch.units[0]!
  const terms = useMemo(() => pu.t.map((n) => pf.terms[n] as Term), [pf, pu])
  const ids = useTermIds(terms)
  const keywords = Object.entries(batch.keywords)
  const keep = unit.keep
  return (
    <>
      <State pu={pu} />
      <Section title="指引">
        <ul class="pv-guides">
          {batch.guides.map((path) => (
            <li key={path}>
              <details>
                <summary>{path}</summary>
                <Box>{shared.guides.find((g) => g.path === path)?.text ?? ''}</Box>
              </details>
            </li>
          ))}
        </ul>
      </Section>
      <Section title="詞彙">
        {batch.glossary.length === 0 ? (
          <p class="pv-none">沒有詞彙。</p>
        ) : (
          <ul class="pv-terms">
            {batch.glossary.map((t, i) => {
              const id = ids.get(terms[i] as Term)
              const label = `${t.kr} → ${t.zh}`
              return (
                <li key={i}>
                  {id !== undefined && !t.provisional ? <a href={termHref(id)}>{label}</a> : label}
                  {t.provisional && <span class="pv-tag">暫定</span>}
                  {t.status && <span class="pv-tag">{t.status}</span>}
                </li>
              )
            })}
          </ul>
        )}
      </Section>
      <Section title="keywords">
        {keywords.length === 0 ? (
          <p class="pv-none">沒有 keywords。</p>
        ) : (
          <ul class="pv-keywords">
            {keywords.map(([id, v]) => (
              <li key={id}>
                <code>{id}</code> {v}
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title="單位">
        <Facts
          rows={[
            ['原因', unit.reason],
            ['說話者', unit.speaker || '（無）'],
            ...(unit.prev_kr ? ([['舊 KR', unit.prev_kr]] as [string, string][]) : []),
            ...(unit.prev_zh ? ([['舊譯', unit.prev_zh]] as [string, string][]) : []),
            ...(unit.draft ? ([['草稿', unit.draft]] as [string, string][]) : []),
            ...(unit.hint ? ([['提示', unit.hint]] as [string, string][]) : []),
            ...Object.entries(unit.refs ?? {}).map(([k, v]): [string, string] => [`參考 ${k}`, v]),
            ['KR', unit.kr],
            ...(unit.jp ? ([['JP', `${unit.jp.value}（${unit.jp.method}）`]] as [string, string][]) : []),
            ...(unit.en ? ([['EN', `${unit.en.value}（${unit.en.method}）`]] as [string, string][]) : []),
            ['keep', keepText(keep)],
          ]}
        />
      </Section>
      <Section title="上下文">
        {batch.context.length === 0 ? (
          <p class="pv-none">沒有上下文行。</p>
        ) : (
          <ol class="pv-context">
            {batch.context.map((c) => (
              <li key={c.key}>
                {c.speaker && <span class="pv-tag">{c.speaker}</span>}
                <span lang="ko">{c.kr}</span>
                <br />
                <span lang="zh-TW">{c.zh}</span>
              </li>
            ))}
          </ol>
        )}
      </Section>
      <RecordAndCheck pu={pu} rules={shared.rules_commit} />
    </>
  )
}

function keepText(k: BatchJSON['units'][number]['keep']): string {
  const parts: string[] = []
  if (k.ids.length > 0) parts.push(`id：${k.ids.join(' ')}`)
  if (k.placeholders.length > 0) parts.push(`佔位符：${k.placeholders.join(' ')}`)
  if (k.tags.length > 0) parts.push(`標籤：${k.tags.join(' ')}`)
  if (k.dante_wrapper) parts.push('Dante 外框')
  return parts.length > 0 ? parts.join('；') : '（沒有）'
}

function RecordAndCheck({ pu, rules }: { pu: PreviewUnit; rules: string }) {
  const rec = pu.rec
  if (rec === undefined) {
    return (
      <Section title="紀錄與檢查">
        <p class="pv-note">沒有譯文可檢查</p>
        {pu.nt && <Facts rows={[['註記', <span class="pv-pre">{pu.nt}</span>]]} />}
      </Section>
    )
  }
  const rows: [string, ComponentChildren][] = [['來源', <OriginBadge origin={rec.origin as Origin} />]]
  if (rec.model) rows.push(['模型', rec.model])
  if (rec.batch) rows.push(['批次', rec.batch])
  if (rec.since) rows.push(['自', rec.since])
  if (rec.rules) rows.push(['規則', `${rec.rules}（${rec.rules === rules ? '與現行規則相同' : '不是現行規則'}）`])
  if (rec.flags && rec.flags.length > 0) rows.push(['旗標', rec.flags.join('、')])
  if (pu.nt) rows.push(['註記', <span class="pv-pre">{pu.nt}</span>])
  const issues = (pu.chk?.issues ?? []) as Issue[]
  return (
    <Section title="紀錄與檢查">
      {rec.stale && <p class="pv-note">這是舊 KR 的譯文</p>}
      {rec.origin === 'fallback' && <p class="pv-note">替代文字（EN 或 KR），不是譯文：結果僅供參考</p>}
      {rec.origin === 'human' && <p class="pv-note">人工譯文</p>}
      <Facts rows={rows} />
      {pu.chk && (
        <div class={`pv-check pv-check-${pu.chk.verdict}`}>
          <p>
            以今天的送出檢查看：<strong>{VERDICT[pu.chk.verdict] ?? pu.chk.verdict}</strong>
          </p>
          <Box label="被檢查的譯文">{pu.chk.zh}</Box>
          {issues.length > 0 && (
            <ul class="pv-issues">
              {issues.map((i, k) => (
                <li key={k}>
                  <code>{i.code}</code> {i.message}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </Section>
  )
}

/** The button that copies the full text; where the browser says no, the text is selected in a box, with a hint. */
function CopyFull({ text }: { text: string }) {
  const box = useRef<HTMLTextAreaElement>(null)
  const [refused, setRefused] = useState(false)
  const [said, setSaid] = useState('')
  const copy = async () => {
    const result = await copyToClipboard(text, box.current!)
    setRefused(result !== 'copied')
    setSaid(result === 'copied' ? '已複製全文' : '')
  }
  return (
    <>
      <div class="pv-actions">
        <button type="button" onClick={copy}>
          複製全文
        </button>
      </div>
      <p class={refused ? 'tv-copyhint' : 'rv-said'} role="status">
        {refused ? COPY_REFUSED_HINT : said}
      </p>
      <textarea
        ref={box}
        class={refused ? 'tv-copybox' : 'tv-copybox is-hidden'}
        readOnly
        aria-label="全文"
        {...(refused ? {} : { tabIndex: -1, 'aria-hidden': 'true' as const })}
        rows={8}
        value={text}
      />
    </>
  )
}

function Full({ text }: { text: string }) {
  return (
    <>
      <CopyFull text={text} />
      <pre class="pv-box pv-full" tabIndex={0} aria-label="全文">
        {text}
      </pre>
    </>
  )
}

interface Docs {
  file: FileData
  pf: PreviewFile
  shared: PreviewShared
}

const TABS = [
  ['compose', '組合'],
  ['full', '全文'],
] as const

function Loaded({ route, docs }: { route: PreviewRoute; docs: Docs }) {
  const { path, ptr } = route
  const { file, pf, shared } = docs
  const [tab, setTab] = useState<'compose' | 'full'>('compose')
  const index = file.units.findIndex((u) => u.p === ptr)
  const unit = file.units[index]
  const pu = pf.units[index] ?? null
  // Left and right switch the tab, and the focus goes with it.
  const onTabKey = (e: KeyboardEvent) => {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
    e.preventDefault()
    const next = tab === 'compose' ? 'full' : 'compose'
    setTab(next)
    document.getElementById(`pv-tab-${next}`)?.focus()
  }

  // The batch, its JSON and its full text; the data of the file and of the preview may disagree (one of them is newer), which the rebuild reports by throwing.
  const built = useMemo(() => {
    if (pu === null || unit === undefined || unit.o === 'copy') return undefined
    try {
      const batch = rebuildBatch(path, file, pf, shared, index)
      return { batch, json: goJSON(batch), full: fullText(shared, batch) }
    } catch {
      return 'broken' as const
    }
  }, [path, file, pf, shared, index])

  let body: ComponentChildren
  if (unit === undefined) {
    body = (
      <p class="pv-gone" role="status">
        找不到這個單位：<code>{ptr}</code>。它可能已被改掉或刪除。
      </p>
    )
  } else if (built === 'broken') {
    body = (
      <p class="pv-gone" role="status">
        預覽資料與檔案資料不一致，請重新整理或等網站更新
      </p>
    )
  } else if (built === undefined || pu === null) {
    body = (
      <p class="pv-gone" role="status">
        {COPY_NOT_SENT}
      </p>
    )
  } else {
    body = (
      <>
        <div class="pv-tabs" role="tablist" onKeyDown={onTabKey}>
          {TABS.map(([id, label]) => (
            <button
              key={id}
              type="button"
              role="tab"
              id={`pv-tab-${id}`}
              data-tab={id}
              aria-selected={tab === id}
              aria-controls="pv-panel"
              tabIndex={tab === id ? 0 : -1}
              class={tab === id ? 'is-on' : ''}
              onClick={() => setTab(id)}
            >
              {label}
            </button>
          ))}
        </div>
        <div role="tabpanel" id="pv-panel" aria-labelledby={`pv-tab-${tab}`} class="pv-panel">
          {tab === 'compose' ? <Composition batch={built.batch} pf={pf} pu={pu} shared={shared} /> : <Full text={built.full} />}
        </div>
        <details class="pv-json">
          <summary>批次 JSON</summary>
          <Box>{built.json}</Box>
        </details>
      </>
    )
  }
  return (
    <div class="preview">
      <header class="pv-head">
        <h1>翻譯預覽</h1>
        <p class="pv-where">
          <code>{path}</code> <code>{ptr}</code>
        </p>
        <p>依規則 <code>{shared.rules_commit.slice(0, 7)}</code> 模擬</p>
      </header>
      <Back path={path} ptr={ptr} />
      {body}
    </div>
  )
}

/** What a unit's translation request is made of, and how its translation fares under today's submit check. */
export function PreviewView({ route }: { route: PreviewRoute }) {
  const key = `${route.path}\n${route.ptr}`
  const loaded = useLoad(key, async () => {
    const [file, pf, shared] = await Promise.all([loadFile(route.path), loadPreviewFile(route.path), loadPreviewShared()])
    return { file, pf, shared }
  })
  if (loaded === undefined) return <p role="status">預覽載入中…</p>
  if (loaded.status === 'error') return <Notice error={loaded.error} />
  return <Loaded key={key} route={route} docs={loaded.data} />
}
