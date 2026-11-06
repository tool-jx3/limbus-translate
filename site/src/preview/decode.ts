// Expands data/preview/<path> as lct site writes it (compact: whatever can be
// put back from files/<path> or from the defaults of the file is left out) to
// the shape the preview page and rebuildBatch work on. The format is
// described by PreviewUnit in internal/site/preview.go.
import { DataError } from '../data'
import type {
  Constraints,
  FileData,
  PreviewCheck,
  PreviewFile,
  PreviewFileWire,
  PreviewRecDefaults,
  PreviewRecord,
  PreviewState,
  PreviewUnit,
  PreviewUnitWire,
} from '../types'

const METHODS: Record<string, string | undefined> = { p: 'pointer', i: 'id', s: 'sequence' }

/** The method of one language from its character in `m`; undefined for "-" (no aligned string). */
function method(where: string, c: string | undefined): string | undefined {
  if (c === '-') return undefined
  const m = c === undefined ? undefined : METHODS[c]
  if (m === undefined) throw new DataError('format', `${where}: no alignment method for ${JSON.stringify(c)}`)
  return m
}

function expandRecord(where: string, wire: NonNullable<PreviewUnitWire['rec']>, rd: PreviewRecDefaults | undefined): PreviewRecord {
  if (!rd) throw new DataError('format', `${where}: a record but no defaults for records`)
  const rec: PreviewRecord = { origin: wire.origin ?? rd.origin, stale: wire.stale === true }
  if (wire.model !== undefined) rec.model = wire.model
  if (wire.batch !== undefined) rec.batch = wire.batch
  const since = wire.since ?? rd.since
  if (since !== '') rec.since = since
  const rules = wire.rules ?? rd.rules
  if (rules !== '') rec.rules = rules
  if (wire.flags !== undefined) rec.flags = wire.flags
  return rec
}

function expandUnit(where: string, wire: PreviewUnitWire, fu: FileData['units'][number], rd: PreviewRecDefaults | undefined): PreviewUnit {
  const it = { ...wire.it }
  if (wire.pz === 1) it.prev_zh = fu.zh
  const keep: Constraints = wire.keep ?? { ids: [], placeholders: [], tags: [], dante_wrapper: false }
  const st: PreviewState = { pending: false, needs_review: false, frozen: false, ...wire.st }
  const out: PreviewUnit = { sp: wire.sp, g: wire.g, t: wire.t, k: wire.k, it, keep, c: wire.c, st }
  if (wire.m !== undefined) {
    if (wire.m.length !== 2) throw new DataError('format', `${where}: the methods ${JSON.stringify(wire.m)} are not two characters`)
    const em = method(where, wire.m[0])
    const jm = method(where, wire.m[1])
    if (em !== undefined) out.em = em
    if (jm !== undefined) out.jm = jm
  }
  if (wire.rec) {
    out.rec = expandRecord(where, wire.rec, rd)
    const chk: PreviewCheck = {
      verdict: wire.chk?.verdict ?? 'accept',
      zh: wire.chk?.zh ?? fu.zh,
      issues: wire.chk?.issues ?? [],
    }
    out.chk = chk
  } else if (wire.chk) {
    throw new DataError('format', `${where}: a check but no record`)
  }
  if (wire.nt !== undefined) out.nt = wire.nt
  return out
}

/** The preview of a file with every default put back; `file` is files/<path> of the same file. */
export function expandPreviewFile(wire: PreviewFileWire, file: FileData): PreviewFile {
  if (wire.units.length !== file.units.length) {
    throw new DataError('format', `${file.path}: ${file.units.length} units in the file, ${wire.units.length} units in the preview`)
  }
  return {
    family: wire.family,
    glossary_check: wire.glossary_check,
    terms: wire.terms,
    keywords: wire.keywords,
    units: wire.units.map((u, i) => (u === null ? null : expandUnit(`${file.path} #${i}`, u, file.units[i] as FileData['units'][number], wire.rd))),
  }
}
