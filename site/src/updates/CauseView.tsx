import type { JSX } from 'preact'
import { useContext, useEffect, useMemo, useRef, useState } from 'preact/hooks'
import type { UpdateEntry, UpdateRecord } from '../types'
import { adjustmentReason, CauseNav } from './CauseChips'
import { CAUSE_KIND_NAMES, causeKey, groupEntries, type CauseGroup } from './causes'

/** The distinct notes of the causes of a group's kind (English, from the history). */
function notesOf(g: CauseGroup): string[] {
  const notes = new Set<string>()
  for (const e of g.entries) for (const c of e.causes ?? []) if (c.kind === g.kind && c.note) notes.add(c.note)
  return [...notes]
}

/** The distinct reasons of the adjustments of a group, each with its number of entries, the most used first. */
function reasonsOf(g: CauseGroup): [string, number][] {
  const counts = new Map<string, number>()
  for (const e of g.entries) {
    const reasons = new Set<string>()
    for (const c of e.causes ?? []) {
      const r = causeKey(c) === g.key ? adjustmentReason(c) : undefined
      if (r) reasons.add(r)
    }
    for (const r of reasons) counts.set(r, (counts.get(r) ?? 0) + 1)
  }
  return [...counts].sort((a, b) => b[1] - a[1])
}

const panelId = (key: string) => `cg-${key}`
const sectionId = (key: string) => `cg-group-${key}`

function GroupHead({ group: g, open, onToggle }: { group: CauseGroup; open: boolean; onToggle: () => void }) {
  const name = CAUSE_KIND_NAMES[g.kind] ?? g.kind
  const what = g.id ?? g.reason
  return (
    <h2 class="cg-title">
      <button type="button" class="cg-head" aria-expanded={open ? 'true' : 'false'} aria-controls={panelId(g.key)} onClick={onToggle}>
        {/* The mark stays on the line of the name, however narrow the screen. */}
        <span class="cg-lead">
          <span class="cg-mark" aria-hidden="true">
            {open ? '▾' : '▸'}
          </span>
          <span class="cg-name">{what ? `${name} ${what}` : name}</span>
        </span>
        {/* The title of an adjustment is its file name, which the name has already. */}
        {g.title && g.title !== g.id && <span class="cg-subtitle">{g.title}</span>}
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
 * the entries of an opened group. The chips of the cards of a group lead to
 * the other groups through CauseNav, which this provides for them.
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
  const nav = useContext(CauseNav)
  const groups = useMemo(() => groupEntries(record, kept), [record, kept])
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(open ? [open] : []))
  // The group to bring into view once it is drawn open: a new object each
  // time, so that the same group can be asked for again.
  const [target, setTarget] = useState<{ key: string }>()
  const toggle = (key: string) =>
    setExpanded((s) => {
      const next = new Set(s)
      if (!next.delete(key)) next.add(key)
      return next
    })
  const show = (key: string) => {
    setExpanded((s) => (s.has(key) ? s : new Set(s).add(key)))
    setTarget({ key })
  }
  // The group a chip of this view was clicked for: the URL then names it too,
  // which need not show it a second time.
  const jumped = useRef('')
  const jump = (key: string) => {
    jumped.current = key
    show(key)
  }
  // A group named by the URL is shown, also when the URL changes while this is.
  useEffect(() => {
    const shown = jumped.current === open
    jumped.current = ''
    if (open && !shown) show(open)
  }, [open])
  // Its panel is hidden until it is open, so the header (never hidden) is what
  // is scrolled to, after the render that opened it. The header takes the
  // focus where it is lost (the chip that led here went with its view) or in
  // this view, not from a control elsewhere, such as a filter whose change
  // drew this view again.
  const view = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!target) return
    setTarget(undefined)
    const section = expanded.has(target.key) ? document.getElementById(sectionId(target.key)) : null
    if (!section) return
    section.scrollIntoView?.({ block: 'start' })
    const active = document.activeElement
    if (!active || active === document.body || view.current?.contains(active)) {
      section.querySelector<HTMLElement>('.cg-head')?.focus({ preventScroll: true })
    }
  }, [target, expanded])
  const legacy = groups.some((g) => g.legacy)
  return (
    <div class="cg" ref={view}>
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
        const notes = isOpen && g.kind === 'unknown' ? notesOf(g) : []
        const reasons = isOpen && (g.kind === 'adjustment' || g.kind === 'adjustment-expired') ? reasonsOf(g) : []
        return (
          <section key={g.key} id={sectionId(g.key)} class="cg-group">
            <GroupHead group={g} open={isOpen} onToggle={() => toggle(g.key)} />
            <div id={panelId(g.key)} class="cg-panel" hidden={!isOpen}>
              {notes.length > 0 && (
                <ul class="cg-notes">
                  {notes.map((n) => (
                    <li key={n}>說明：{n}</li>
                  ))}
                </ul>
              )}
              {reasons.length > 0 && (
                <ul class="cg-notes cg-reasons">
                  {reasons.map(([r, n]) => (
                    <li key={r}>
                      理由：{r}（{n} 筆）
                    </li>
                  ))}
                </ul>
              )}
              {isOpen && (
                <CauseNav.Provider value={{ href: nav.href, current: g.key, show: jump }}>{draw(g.entries, g)}</CauseNav.Provider>
              )}
            </div>
          </section>
        )
      })}
    </div>
  )
}
