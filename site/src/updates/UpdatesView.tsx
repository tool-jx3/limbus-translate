import { useEffect, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { DataError, loadUpdates } from '../data'
import { updateHref } from '../route'
import type { UpdateRelease } from '../types'

type Loaded = { status: 'ready'; releases: UpdateRelease[] } | { status: 'error'; error: DataError }

const COUNTS: { key: 'new' | 'kr_changed' | 'deleted' | 'zh_changed'; name: string; cls: string }[] = [
  { key: 'new', name: '新增', cls: 'is-new' },
  { key: 'kr_changed', name: 'KR 修改', cls: 'is-kr' },
  { key: 'deleted', name: '刪除', cls: 'is-deleted' },
  { key: 'zh_changed', name: '只有譯文改變', cls: 'is-zh' },
]

/** The day of a release, in the viewer's own time zone (YYYY-MM-DD). */
export function day(releasedAt: string): string {
  const t = new Date(releasedAt)
  if (Number.isNaN(t.getTime())) return releasedAt.slice(0, 10)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${t.getFullYear()}-${pad(t.getMonth() + 1)}-${pad(t.getDate())}`
}

function Row({ release: r }: { release: UpdateRelease }) {
  return (
    <li class="uv-row">
      {r.recorded && r.kind !== 'first' ? (
        <a class="uv-tag" href={updateHref(r.tag)}>
          {r.tag}
        </a>
      ) : (
        <span class="uv-tag">{r.tag}</span>
      )}
      <time class="uv-date" dateTime={r.released_at}>
        {day(r.released_at)}
      </time>
      {r.kind === 'first' ? (
        <span class="uv-first">第一版，沒有可比對的上一版</span>
      ) : !r.recorded ? (
        <span class="uv-first">還沒有比對紀錄，下次建置後補上</span>
      ) : (
        <>
          {r.prev_tag && <span class="uv-prev">比對 {r.prev_tag}</span>}
          <span class={r.kind === 'game' ? 'uv-origin is-game' : 'uv-origin is-rebuild'}>
            {r.kind === 'game' ? `遊戲更新 ${r.raw_tag}` : '重建'}
          </span>
          <span class="uv-counts">
            {COUNTS.filter((c) => r.counts[c.key] > 0).map((c) => (
              <span key={c.key} class={`uv-count ${c.cls}`}>
                {c.name} {r.counts[c.key]}
              </span>
            ))}
            {r.other_files > 0 && <span class="uv-other">其他檔案 {r.other_files}</span>}
          </span>
        </>
      )}
    </li>
  )
}

/** The page of the updates: one row per Release, the newest first. */
export function UpdatesView() {
  const [state, setState] = useState<Loaded>()
  useEffect(() => {
    let live = true
    loadUpdates().then(
      (doc) => {
        if (live) setState({ status: 'ready', releases: doc.releases })
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
    <section class="updates">
      <h1>更新</h1>
      <ol class="uv-list">
        {state.releases.map((r) => (
          <Row key={r.tag} release={r} />
        ))}
      </ol>
    </section>
  )
}
