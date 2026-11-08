import type { UpdateEntry, UpdateEntryCause, UpdateRecord } from '../types'

/** The Chinese name of each cause kind. */
export const CAUSE_KIND_NAMES: Record<string, string> = {
  rewrite: '改寫檔',
  adjustment: '人工調整',
  'adjustment-expired': '人工調整失效',
  requeue: '重翻申請',
  reuse: '沿用同文單位',
  'kr-changed': 'KR 改版',
  new: '新增',
  deleted: '刪除',
  late: '補翻',
  retranslated: '重翻',
  other: '其他',
  unknown: '無法判定',
}

/** One group of changes with the same cause. */
export interface CauseGroup {
  /** causeKey of the cause; the `g` URL parameter and the chips use it. */
  key: string
  kind: string
  id?: string
  title?: string
  path?: string
  commit?: string
  /** The reason of a retranslated group. */
  reason?: string
  entries: UpdateEntry[]
  /** The number of distinct files in the group. */
  files: number
  /** The cause is a rule (rewrite, adjustment, adjustment-expired, requeue). */
  isRule: boolean
  /** Grouped by the old coarse cause or the kind: the record has no causes. */
  legacy?: boolean
}

const RULE_KINDS = new Set(['rewrite', 'adjustment', 'adjustment-expired', 'requeue'])

/** The order of the groups that are not rules. */
const KIND_ORDER = ['kr-changed', 'new', 'deleted', 'reuse', 'late', 'retranslated', 'adjustment', 'other', 'unknown']

/** The group key of a cause: `<kind>:<id>` for a rule, `retranslated:<reason>`, else the bare kind. */
export function causeKey(c: UpdateEntryCause): string {
  if (RULE_KINDS.has(c.kind)) return `${c.kind}:${c.id ?? ''}`
  if (c.kind === 'retranslated') return `retranslated:${c.reason ?? ''}`
  return c.kind
}

/** Groups the entries of a record by cause. An entry with several causes is in each group, once. */
export function groupByCause(rec: UpdateRecord): CauseGroup[] {
  const legacy = !rec.causes_version && !rec.entries.some((e) => e.causes && e.causes.length > 0)
  const groups = new Map<string, CauseGroup>()
  const add = (key: string, c: UpdateEntryCause, e: UpdateEntry) => {
    let g = groups.get(key)
    if (!g) {
      const isRule = !legacy && RULE_KINDS.has(c.kind)
      const info = isRule ? rec.rules?.[key] : undefined
      g = { key, kind: c.kind, entries: [], files: 0, isRule }
      if (isRule) {
        g.id = c.id
        if (info) Object.assign(g, { title: info.title, path: info.path, commit: info.commit })
      }
      if (c.kind === 'retranslated' && c.reason) g.reason = c.reason
      if (legacy) g.legacy = true
      groups.set(key, g)
    }
    if (g.entries[g.entries.length - 1] !== e) g.entries.push(e)
  }
  for (const e of rec.entries) {
    if (legacy) {
      const kind = e.kind === 'zh-changed' ? (e.cause ?? 'other') : e.kind
      add(kind, { kind }, e)
    } else for (const c of e.causes ?? []) add(causeKey(c), c, e)
  }
  const out = [...groups.values()]
  for (const g of out) g.files = new Set(g.entries.map((e) => e.file)).size
  const rank = (g: CauseGroup) => {
    const i = KIND_ORDER.indexOf(g.kind)
    return i < 0 ? KIND_ORDER.length : i
  }
  return out.sort((a, b) =>
    a.isRule !== b.isRule ? (a.isRule ? -1 : 1) : a.isRule ? b.entries.length - a.entries.length : rank(a) - rank(b),
  )
}
