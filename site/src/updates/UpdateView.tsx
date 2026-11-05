import type { JSX } from 'preact'
import { useEffect, useMemo, useState } from 'preact/hooks'
import { Notice } from '../app/Notice'
import { DataError, loadUpdate } from '../data'
import { ShownText } from '../review/parts'
import { updateHref, updatesHref, type Route } from '../route'
import type { Settings } from '../settings'
import type { UpdateEntry, UpdateKind, UpdateRecord } from '../types'
import { KIND_NAMES, UpdateCard } from './UpdateCard'
import { UpdateTable } from './UpdateTable'
import { countKinds, entryFiles, entryOrigins, filterEntries, type UpdateFilter } from './filter'
import { day } from './UpdatesView'

type UpdateRoute = Extract<Route, { kind: 'update' }>

/** The entries shown at first, and added by each press of the button. */
export const UPDATE_PAGE = 300

const KINDS: { kind: UpdateKind; key: 'new' | 'kr_changed' | 'deleted' | 'zh_changed' }[] = [
  { kind: 'new', key: 'new' },
  { kind: 'kr-changed', key: 'kr_changed' },
  { kind: 'deleted', key: 'deleted' },
  { kind: 'zh-changed', key: 'zh_changed' },
]

type Loaded = { tag: string } & ({ status: 'ready'; record: UpdateRecord } | { status: 'error'; error: DataError })

function useUpdate(tag: string): Loaded | undefined {
  const [state, setState] = useState<Loaded>()
  useEffect(() => {
    let live = true
    loadUpdate(tag).then(
      (record) => {
        if (live) setState({ tag, status: 'ready', record })
      },
      (e: unknown) => {
        if (live) setState({ tag, status: 'error', error: e instanceof DataError ? e : new DataError('network', String(e)) })
      },
    )
    return () => {
      live = false
    }
  }, [tag])
  return state?.tag === tag ? state : undefined
}

/** Goes to the same update with other filters. */
function go(route: UpdateRoute, change: Partial<UpdateFilter>): void {
  location.hash = updateHref(route.tag, { kinds: route.kinds, origin: route.origin, file: route.file, copy: route.copy, ...change })
}

function Select({ name, label, value, options, onChange }: { name: string; label: string; value: string; options: string[]; onChange: (v: string) => void }) {
  return (
    <label>
      {label}
      <select name={name} value={value} onChange={(e) => onChange(e.currentTarget.value)}>
        <option value="">全部</option>
        {options.map((o) => (
          <option key={o} value={o}>
            {o}
          </option>
        ))}
      </select>
    </label>
  )
}

function Head({ record, files }: { record: UpdateRecord; files: number }) {
  const game = record.raw_tag !== record.prev_raw_tag
  return (
    <header class="uv-head">
      <h1>{record.tag}</h1>
      <p class="uv-meta">
        <time dateTime={record.released_at}>{day(record.released_at)}</time>
        <span class={game ? 'uv-origin is-game' : 'uv-origin is-rebuild'}>{game ? `遊戲更新 ${record.raw_tag}` : '重建'}</span>
        <span>比對 {record.prev_tag}</span>
        <span>{files} 個檔案有變動</span>
      </p>
      {record.other_files.length > 0 && (
        <details class="uv-others">
          <summary>其他檔案（{record.other_files.length}）</summary>
          <ul>
            {record.other_files.map((f) => (
              <li key={f}>
                <code>{f}</code>
              </li>
            ))}
          </ul>
        </details>
      )}
    </header>
  )
}

function Filters({ route, entries, settings, onSettings }: { route: UpdateRoute; entries: UpdateEntry[]; settings: Settings; onSettings: (s: Settings) => void }) {
  const counts = countKinds(entries, route.copy)
  const copies = entries.filter((e) => e.copy).length
  const toggle = (kind: UpdateKind) =>
    go(route, { kinds: route.kinds.includes(kind) ? route.kinds.filter((k) => k !== kind) : [...route.kinds, kind] })
  return (
    <div class="uv-filters">
      <div class="uv-kinds">
        {KINDS.map(({ kind, key }) => (
          <button
            key={kind}
            type="button"
            class={`uv-kind ${KIND_NAMES[kind].cls}`}
            aria-pressed={route.kinds.includes(kind) ? 'true' : 'false'}
            onClick={() => toggle(kind)}
          >
            {KIND_NAMES[kind].name} {counts[key]}
          </button>
        ))}
      </div>
      <Select name="origin" label="來源" value={route.origin} options={entryOrigins(entries)} onChange={(origin) => go(route, { origin })} />
      <Select name="file" label="檔案" value={route.file} options={entryFiles(entries)} onChange={(file) => go(route, { file })} />
      <label>
        <input type="checkbox" name="hide-copy" checked={!route.copy} onChange={(e) => go(route, { copy: !e.currentTarget.checked })} />
        隱藏不需翻譯的單位（{copies} 個）
      </label>
      <div class="uv-views" role="group" aria-label="顯示方式">
        {(['cards', 'table'] as const).map((view) => (
          <button key={view} type="button" aria-pressed={settings.updateView === view ? 'true' : 'false'} onClick={() => onSettings({ ...settings, updateView: view })}>
            {view === 'cards' ? '卡片' : '表格'}
          </button>
        ))}
      </div>
    </div>
  )
}

/** UPDATE_PAGE of the entries at first, then UPDATE_PAGE more at each press; `children` draws the ones shown. */
function Paged({ entries, children }: { entries: UpdateEntry[]; children: (shown: UpdateEntry[]) => JSX.Element }) {
  const [limit, setLimit] = useState(UPDATE_PAGE)
  const shown = entries.slice(0, limit)
  const left = entries.length - shown.length
  return (
    <>
      {children(shown)}
      {left > 0 && (
        <button type="button" class="rv-more" onClick={() => setLimit(limit + UPDATE_PAGE)}>
          再顯示 {Math.min(left, UPDATE_PAGE)} 筆
        </button>
      )}
    </>
  )
}

/** The cards of the entries, grouped by file. */
function Cards({ entries }: { entries: UpdateEntry[] }) {
  const groups = new Map<string, UpdateEntry[]>()
  for (const e of entries) {
    const g = groups.get(e.file)
    if (g) g.push(e)
    else groups.set(e.file, [e])
  }
  return (
    <>
      {[...groups].map(([file, list]) => (
        <section key={file} class="uv-group">
          <h2 class="uv-file">{file}</h2>
          <div class="uc-list">
            {list.map((e) => (
              <UpdateCard key={`${e.file}|${e.ptr}|${e.kind}`} entry={e} />
            ))}
          </div>
        </section>
      ))}
    </>
  )
}

/** The changes of one Release. */
export function UpdateView({ route, settings, onSettings }: { route: UpdateRoute; settings: Settings; onSettings: (s: Settings) => void }) {
  const state = useUpdate(route.tag)
  const shown = useMemo(() => ({ langs: settings.langs, raw: settings.rawTags }), [settings.langs, settings.rawTags])
  const filter: UpdateFilter = { kinds: route.kinds, origin: route.origin, file: route.file, copy: route.copy }
  const entries = state?.status === 'ready' ? state.record.entries : undefined
  const list = useMemo(() => (entries ? filterEntries(entries, filter) : []), [entries, route.kinds.join(','), route.origin, route.file, route.copy])

  if (!state) return <p role="status">資料載入中…</p>
  if (state.status === 'error') {
    // A Release with no record (the first one, or one the next build has not
    // reached) has no file; reloading would not bring one.
    if (state.error.kind === 'missing') {
      return (
        <section class="update">
          <p class="uv-empty">沒有 {route.tag} 的比對紀錄</p>
          <p>
            <a href={updatesHref()}>回到更新列表</a>
          </p>
        </section>
      )
    }
    return <Notice error={state.error} />
  }
  const { record } = state
  return (
    <ShownText.Provider value={shown}>
      <section class="update">
        <Head record={record} files={entryFiles(record.entries).length} />
        <Filters route={route} entries={record.entries} settings={settings} onSettings={onSettings} />
        {list.length === 0 ? (
          <p class="uv-empty">沒有符合的變動</p>
        ) : (
          // Keyed by the filter, so that a change of it starts again from the first page.
          <Paged key={`${route.kinds.join(',')}|${route.origin}|${route.file}|${route.copy}`} entries={list}>
            {(shownList) => (settings.updateView === 'table' ? <UpdateTable entries={shownList} /> : <Cards entries={shownList} />)}
          </Paged>
        )}
      </section>
    </ShownText.Provider>
  )
}
