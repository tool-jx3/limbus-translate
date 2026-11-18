import { useEffect, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { DataError, loadDataChanges } from '../data'
import { dataChangeHref, updatesHref } from '../route'
import type { DataLang, DataSnapshot } from '../types'
import { day } from '../updates/UpdatesView'
import { COUNT_CHIPS } from './counts'

type Loaded = { status: 'ready'; snapshots: DataSnapshot[] } | { status: 'error'; error: DataError }

/** The packs that are compared with KR, in the order they are shown. */
const OTHER_LANGS: DataLang[] = ['jp', 'en']

function Row({ snapshot: s }: { snapshot: DataSnapshot }) {
  const chips = COUNT_CHIPS.filter((c) => s.counts[c.key] > 0)
  const others = OTHER_LANGS.filter((lang) => (s.vs_kr[lang] ?? 0) > 0)
  return (
    <li class="uv-row">
      <a class="uv-tag" href={dataChangeHref(s.tag)}>
        {s.tag}
      </a>
      <time class="uv-date" dateTime={s.at}>
        {day(s.at)}
      </time>
      <span class="uv-prev">比對 {s.prev_tag}</span>
      {chips.length === 0 ? (
        <span class="uv-first">結構沒有變化</span>
      ) : (
        <span class="uv-counts">
          {chips.map((c) => (
            <span key={c.key} class={`uv-count ${c.cls}`}>
              {c.name} {s.counts[c.key]}
            </span>
          ))}
        </span>
      )}
      {others.map((lang) => (
        <a key={lang} class="dv-vs" href={dataChangeHref(s.tag, lang)}>
          {lang.toUpperCase()} 有 {s.vs_kr[lang]} 項和 KR 不同
        </a>
      ))}
    </li>
  )
}

/** The page of the data changes: one row per snapshot of the raw branch that has one before it, the newest first. */
export function DataChangesView() {
  const [state, setState] = useState<Loaded>()
  useEffect(() => {
    let live = true
    loadDataChanges().then(
      (doc) => {
        if (live) setState({ status: 'ready', snapshots: doc.snapshots })
      },
      (e: unknown) => {
        if (live) setState({ status: 'error', error: e instanceof DataError ? e : new DataError('network', String(e)) })
      },
    )
    return () => {
      live = false
    }
  }, [])

  if (!state) return <p role="status">資料載入中…</p>
  if (state.status === 'error') return <Notice error={state.error} />
  return (
    <section class="updates datas">
      <h1>資料變化</h1>
      <p class="dv-intro">
        遊戲每次更新，韓文原文的資料在結構上多了或少了什麼：檔案、檔案種類、JSON 欄位、條目，以及文字裡的標記。這些變化可能需要新的分流、欄位或檢查規則，也可能讓遊戲程式研究報告的結論過期。日文、英文的資料也各比一次，變化和韓文不同時另外標出。文字內容改了什麼，看<a href={updatesHref()}>更新</a>頁。
      </p>
      {state.snapshots.length === 0 ? (
        <p class="dv-empty">還沒有可比對的快照：raw 分支要有兩個以上的快照才有比對。</p>
      ) : (
        <ol class="uv-list">
          {state.snapshots.map((s) => (
            <Row key={s.tag} snapshot={s} />
          ))}
        </ol>
      )}
    </section>
  )
}
