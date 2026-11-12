// Rebuilds, in the browser, the batch JSON that `lct batch preview` (and
// `batch next`) would write for one unit, from data/files/<path>,
// data/preview/<path> and data/preview.json. The text must equal Go's
// byte for byte; the contract test reads site/fixtures/preview/.
import type { BatchJSON, BatchUnit, ContextLine, FileData, NameRef, PreviewFile, PreviewShared, Term } from '../types'

/**
 * The paths of the maps Go writes with their keys sorted (Batch.Keywords,
 * BatchUnit.Refs). A path is the keys from the root, an array adding nothing:
 * the refs of the units of a batch are "units.refs".
 */
const BATCH_MAPS: ReadonlySet<string> = new Set(['keywords', 'units.refs'])

const utf8 = new TextEncoder()

/** Orders strings by their UTF-8 bytes, as Go sorts map keys. */
function byBytes(a: string, b: string): number {
  const x = utf8.encode(a)
  const y = utf8.encode(b)
  for (let i = 0; i < x.length && i < y.length; i++) {
    if (x[i] !== y[i]) return (x[i] as number) - (y[i] as number)
  }
  return x.length - y.length
}

function write(v: unknown, indent: string, path: string, sorted: ReadonlySet<string>): string | undefined {
  if (v === null || typeof v !== 'object') return JSON.stringify(v)
  const inner = indent + '  '
  if (Array.isArray(v)) {
    if (v.length === 0) return '[]'
    return '[\n' + v.map((e) => inner + (write(e, inner, path, sorted) ?? 'null')).join(',\n') + '\n' + indent + ']'
  }
  const rec = v as Record<string, unknown>
  const keys = Object.keys(rec)
  if (sorted.has(path)) keys.sort(byBytes)
  const lines: string[] = []
  for (const k of keys) {
    const text = write(rec[k], inner, path === '' ? k : `${path}.${k}`, sorted)
    if (text !== undefined) lines.push(`${inner}${JSON.stringify(k)}: ${text}`)
  }
  if (lines.length === 0) return '{}'
  return '{\n' + lines.join(',\n') + '\n' + indent + '}'
}

function encode(value: unknown, sorted: ReadonlySet<string>): string {
  const s = (write(value, '', '', sorted) ?? 'null').replaceAll('\u2028', '\\u2028').replaceAll('\u2029', '\\u2029')
  return s + '\n'
}

/**
 * The text Go's json.Encoder writes with SetEscapeHTML(false) and SetIndent("", "  "):
 * JSON.stringify(v, null, 2) with U+2028 and U+2029 escaped and a line feed at the end.
 * Objects keep the order they were made in (a Go struct); see batchJSON for maps.
 */
export function goJSON(value: unknown): string {
  return encode(value, new Set())
}

/**
 * goJSON of a batch, with its maps (keywords, the refs of each unit) written
 * with their keys in UTF-8 byte order, as Go does, which a JS object cannot keep
 * (integer-like keys come first). They are found by path, so a copy of the batch
 * is written the same.
 */
export function batchJSON(batch: BatchJSON): string {
  return encode(batch, BATCH_MAPS)
}

/** A plain copy of the map with its keys in byte order (the order the object keeps, integer-like keys apart). */
function goMap(m: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const k of Object.keys(m).sort(byBytes)) out[k] = m[k] as string
  return out
}

/** The batch the unit at `index` of the file gets when it is sent for translation alone. */
export function rebuildBatch(path: string, file: FileData, pf: PreviewFile, shared: PreviewShared, index: number): BatchJSON {
  const fu = file.units[index]
  const pu = pf.units[index]
  if (!fu || !pu) throw new Error(`${path}: unit ${index} has no preview (a copy is not sent for translation)`)
  if (pf.units.length !== file.units.length) {
    throw new Error(`${path}: ${file.units.length} units in the file, ${pf.units.length} units in the preview`)
  }
  const key = `${path}|${fu.p}`

  const glossary = pu.t.map((n): Term => {
    const t = pf.terms[n]
    if (!t) throw new Error(`${path}: no term ${n}`)
    return { ...t, units: [key] }
  })
  const names = pu.n.map((n): NameRef => {
    const r = pf.names[n]
    if (!r) throw new Error(`${path}: no name ${n}`)
    return { ...r, units: [key] }
  })
  const keywords: Record<string, string> = {}
  for (const id of pu.k) {
    const name = pf.keywords[id]
    if (name === undefined) throw new Error(`${path}: no keyword ${id}`)
    keywords[id] = name
  }

  const unit: Partial<BatchUnit> = {
    key,
    file: path,
    field: fu.f,
    speaker: pu.sp,
    kr: fu.kr,
    reason: pu.it.reason,
  }
  if (pu.it.prev_kr) unit.prev_kr = pu.it.prev_kr
  if (pu.it.prev_zh) unit.prev_zh = pu.it.prev_zh
  if (pu.it.draft) unit.draft = pu.it.draft
  if (pu.it.hint) unit.hint = pu.it.hint
  if (pu.em) {
    if (fu.en === undefined) throw new Error(`${key}: the preview has an EN method but the file no EN string`)
    unit.en = { value: fu.en, method: pu.em }
  }
  if (pu.jm) {
    if (fu.jp === undefined) throw new Error(`${key}: the preview has a JP method but the file no JP string`)
    unit.jp = { value: fu.jp, method: pu.jm }
  }
  if (pu.it.refs && Object.keys(pu.it.refs).length > 0) unit.refs = goMap(pu.it.refs)
  unit.keep = pu.keep

  const context = pu.c.map((c): ContextLine => {
    const i = c[0] as number
    const cu = file.units[i]
    const cp = pf.units[i]
    if (!cu || !cp) throw new Error(`${path}: context unit ${i} has no preview (a copy is not a line of context)`)
    return { key: `${path}|${cu.p}`, speaker: cp.sp, kr: cu.kr, zh: c.length > 1 ? (c[1] as string) : cu.zh }
  })

  const batch: Partial<BatchJSON> = { id: 'preview', family: pf.family }
  if (pf.glossary_check !== '') batch.glossary_check = pf.glossary_check
  batch.file = path
  batch.files = [path]
  batch.guides = pu.g.map((n) => {
    const g = shared.guides[n]
    if (!g) throw new Error(`no guide ${n}`)
    return g.path
  })
  batch.glossary = glossary
  batch.keywords = goMap(keywords)
  batch.names = names
  batch.units = [unit as BatchUnit]
  batch.context = context
  return batch as BatchJSON
}

/** The text a translator is handed: the prompt, every guide of the batch, and the batch JSON. */
export function fullText(shared: PreviewShared, batch: BatchJSON): string {
  const parts = [`# prompts/translate.md\n\n${shared.prompt}`]
  for (const path of batch.guides) {
    const g = shared.guides.find((x) => x.path === path)
    if (!g) throw new Error(`no guide ${path}`)
    parts.push(`# ${path}\n\n${g.text}`)
  }
  parts.push(`# 批次 JSON\n\n${batchJSON(batch)}`)
  return parts.join('\n\n')
}
