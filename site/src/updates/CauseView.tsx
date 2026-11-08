import type { JSX } from 'preact'
import { useEffect, useMemo, useRef, useState } from 'preact/hooks'
import type { UpdateEntry, UpdateRecord } from '../types'
import { CAUSE_KIND_NAMES, groupEntries, type CauseGroup } from './causes'

/** The distinct notes of the causes of a group's kind (English, from the history). */
function notesOf(g: CauseGroup): string[] {
  const notes = new Set<string>()
  for (const e of g.entries) for (const c of e.causes ?? []) if (c.kind === g.kind && c.note) notes.add(c.note)
  return [...notes]
}

const panelId = (key: string) => `cg-${key}`
const sectionId = (key: string) => `cg-group-${key}`

function GroupHead({ group: g, open, onToggle }: { group: CauseGroup; open: boolean; onToggle: () => void }) {
  const name = CAUSE_KIND_NAMES[g.kind] ?? g.kind
  const what = g.id ?? g.reason
  return (
    <h2 class="cg-title">
      <button type="button" class="cg-head" aria-expanded={open ? 'true' : 'false'} aria-controls={panelId(g.key)} onClick={onToggle}>
        <span class="cg-mark" aria-hidden="true">
          {open ? '▾' : '▸'}
        </span>
        <span class="cg-name">{what ? `${name} ${what}` : name}</span>
        {g.title && <span class="cg-subtitle">{g.title}</span>}
        <span class="cg-count">
          {g.entries.length} 筆、{g.files} 個檔案
        </span>
        {(g.path || g.commit) && (
          <span class="cg-rule">
            {g.path && <code>{g.path}</code>}
            {g.commit && <code>{g.commit.slice(0, 7)}</code>}
          </span>
        )}
      </button>
    </h2>
  )
}

/**
 * The changes grouped by cause, each group collapsed to its header until it is
 * opened. `open` is a group to show opened and scrolled into view; `draw` draws
 * the entries of an opened group.
 */
export function CauseView({
  record,
  kept,
  open,
  draw,
}: {
  record: UpdateRecord
  kept: UpdateEntry[]
  open: string
  draw: (entries: UpdateEntry[], group: CauseGroup) => JSX.Element
}) {
  const groups = useMemo(() => groupEntries(record, kept), [record, kept])
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(open ? [open] : []))
  const toggle = (key: string) =>
    setExpanded((s) => {
      const next = new Set(s)
      if (!next.delete(key)) next.add(key)
      return next
    })
  // A group named by the URL is opened, also when the URL changes while this is shown.
  useEffect(() => {
    if (open) setExpanded((s) => (s.has(open) ? s : new Set(s).add(open)))
  }, [open])
  // And scrolled to once it is open: its panel is hidden until then, so the
  // header (never hidden) is what is scrolled to, after the render that opened it.
  const openIsExpanded = open !== '' && expanded.has(open)
  const scrolledTo = useRef('')
  useEffect(() => {
    if (!openIsExpanded || scrolledTo.current === open) return
    const section = document.getElementById(sectionId(open))
    if (!section) return
    scrolledTo.current = open
    section.scrollIntoView?.({ block: 'start' })
  }, [open, openIsExpanded])
  const legacy = groups.some((g) => g.legacy)
  return (
    <div class="cg">
      {legacy && (
        <p class="cg-legacy" role="status">
          這個 Release 的紀錄尚未補上原因，暫以舊的分類顯示。
        </p>
      )}
      <div class="cg-all">
        <button type="button" onClick={() => setExpanded(new Set(groups.map((g) => g.key)))}>
          全部展開
        </button>
        <button type="button" onClick={() => setExpanded(new Set())}>
          全部收合
        </button>
      </div>
      {groups.map((g) => {
        const isOpen = expanded.has(g.key)
        const notes = g.kind === 'unknown' ? notesOf(g) : []
        return (
          <section key={g.key} id={sectionId(g.key)} class="cg-group">
            <GroupHead group={g} open={isOpen} onToggle={() => toggle(g.key)} />
            <div id={panelId(g.key)} class="cg-panel" hidden={!isOpen}>
              {isOpen && notes.length > 0 && (
                <ul class="cg-notes">
                  {notes.map((n) => (
                    <li key={n}>說明：{n}</li>
                  ))}
                </ul>
              )}
              {isOpen && draw(g.entries, g)}
            </div>
          </section>
        )
      })}
    </div>
  )
}
