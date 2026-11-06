import { useEffect, useMemo, useState } from 'preact/hooks'
import { filterFiles } from '../data'
import { fileHref } from '../route'
import type { FileSummary, FileTag, Index, IndexGroup } from '../types'

export interface DirectoryProps {
  index: Index
  /** The file filter; the directory shows what matches it, every level open. */
  query: string
  /** The path of the file in view. */
  current?: string
}

type TagKind = FileTag['kind']

/** The three selects, in the order they are drawn. */
const SELECTS: { kind: TagKind; label: string }[] = [
  { kind: 'chapter', label: '章節' },
  { kind: 'character', label: '角色' },
  { kind: 'event', label: '活動' },
]

/** The speakers of a file that the filter matched, to show why the file is there. */
function matchedSpeakers(f: FileSummary, needle: string): string[] {
  if (needle === '') return []
  return f.speakers.filter((s) => s.toLowerCase().includes(needle))
}

/** The options of one select: the distinct tags of that kind, chapters by number, characters by label, events alphabetically. */
function tagOptions(files: FileSummary[], kind: TagKind): FileTag[] {
  const seen = new Map<string, FileTag>()
  for (const f of files) if (f.tag?.kind === kind && !seen.has(f.tag.value)) seen.set(f.tag.value, f.tag)
  const tags = [...seen.values()]
  const byText = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0)
  if (kind === 'chapter') return tags.sort((a, b) => Number(a.value) - Number(b.value))
  return tags.sort((a, b) => byText(a.label, b.label))
}

/** A group with only the files in `keep`, children pruned; null when nothing is left. `total` counts the children too. */
interface Shown {
  group: IndexGroup
  key: string
  files: number[]
  children: Shown[]
  total: number
}

function prune(group: IndexGroup, parent: string, keep: (n: number) => boolean): Shown | null {
  const key = `${parent}/${group.kind}:${group.node}`
  const files = group.files.filter(keep)
  const children = group.children.map((c) => prune(c, key, keep)).filter((c): c is Shown => c !== null)
  const total = files.length + children.reduce((sum, c) => sum + c.total, 0)
  return total === 0 ? null : { group, key, files, children, total }
}

/** The keys of the levels from the top down to the group that lists file n directly; [] when none does. */
function pathTo(groups: Shown[], n: number): string[] {
  for (const s of groups) {
    if (s.files.includes(n)) return [s.key]
    const below = pathTo(s.children, n)
    if (below.length > 0) return [s.key, ...below]
  }
  return []
}

/** The group tree of the index, from index.groups: each level opens and closes; a filter or a select opens all it has hits in. */
export function Directory({ index, query, current }: DirectoryProps) {
  const needle = query.trim().toLowerCase()
  const [pick, setPick] = useState<{ kind: TagKind; value: string } | null>(null)
  const filtering = needle !== '' || pick !== null

  const keep = useMemo(() => {
    const matched = new Set(filterFiles(index, query))
    return (n: number) => {
      if (!matched.has(n)) return false
      if (pick === null) return true
      const tag = index.files[n]?.tag
      return tag?.kind === pick.kind && tag.value === pick.value
    }
  }, [index, query, pick])

  const shown = useMemo(() => index.groups.map((g) => prune(g, '', keep)).filter((s): s is Shown => s !== null), [index, keep])
  const options = useMemo(() => SELECTS.map((s) => ({ ...s, tags: tagOptions(index.files, s.kind) })), [index])

  const currentPos = useMemo(() => index.files.findIndex((f) => f.path === current), [index, current])
  // Every group of the whole tree, not of the filtered one: the levels of the current file open whatever the filter says.
  const currentKeys = useMemo(
    () => (currentPos < 0 ? [] : pathTo(index.groups.map((g) => prune(g, '', () => true)).filter((s): s is Shown => s !== null), currentPos)),
    [index, currentPos],
  )
  const [open, setOpen] = useState<ReadonlySet<string>>(() => new Set(currentKeys))

  // Moving to another file (the next-file button) opens its levels; the ones open stay open.
  useEffect(() => {
    if (currentKeys.length > 0) setOpen((prev) => (currentKeys.every((k) => prev.has(k)) ? prev : new Set([...prev, ...currentKeys])))
  }, [currentKeys])

  const toggle = (key: string) =>
    setOpen((prev) => {
      const next = new Set(prev)
      if (!next.delete(key)) next.add(key)
      return next
    })

  const choose = (kind: TagKind, value: string) => setPick(value === '' ? null : { kind, value })

  const renderGroup = (s: Shown) => {
    const expanded = filtering || open.has(s.key)
    return (
      <section key={s.key} class={`dir-family dir-${s.group.kind}`}>
        <h2>
          <button type="button" aria-expanded={expanded} onClick={() => toggle(s.key)}>
            <span class="dir-caret" aria-hidden="true">{expanded ? '▾' : '▸'}</span>
            <span class="dir-family-id">{s.group.label}</span>
            <span class="dir-count">{s.total}</span>
          </button>
        </h2>
        {expanded && (
          <>
            {s.children.map(renderGroup)}
            {s.files.length > 0 && (
              <ul>
                {s.files.map((n) => {
                  const f = index.files[n]
                  if (!f) return null
                  const who = matchedSpeakers(f, needle)
                  return (
                    <li key={f.path}>
                      <a href={fileHref(f.path)} aria-current={f.path === current ? 'page' : undefined}>
                        <span class="dir-path">{f.path}</span>
                        {f.label !== '' && <span class="dir-label">{f.label}</span>}
                        {who.length > 0 && <span class="dir-who">{who.join('、')}</span>}
                      </a>
                    </li>
                  )
                })}
              </ul>
            )}
          </>
        )}
      </section>
    )
  }

  return (
    <nav class="directory" aria-label="檔案目錄">
      <div class="dir-selects">
        {options.map(({ kind, label, tags }) =>
          tags.length === 0 ? null : (
            <label key={kind}>
              <span>{label}</span>
              <select value={pick?.kind === kind ? pick.value : ''} onChange={(e) => choose(kind, e.currentTarget.value)}>
                <option value="">全部</option>
                {tags.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            </label>
          ),
        )}
      </div>
      {shown.length === 0 ? <p class="dir-empty">沒有符合的檔案</p> : shown.map(renderGroup)}
    </nav>
  )
}
