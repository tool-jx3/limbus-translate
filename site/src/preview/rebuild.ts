// Rebuilds, in the browser, the batch JSON that `lct batch preview` (and
// `batch next`) would write for one unit, from data/files/<path>,
// data/preview/<path> and data/preview.json. The text must equal Go's
// byte for byte; the contract test reads site/fixtures/preview/.
import type { BatchJSON, BatchUnit, ContextLine, FileData, PreviewFile, PreviewShared, Term } from '../types'

/** The text Go's json.Encoder writes with SetEscapeHTML(false) and SetIndent("", "  "). */
export function goJSON(value: unknown): string {
  const s = JSON.stringify(value, null, 2).replaceAll('\u2028', '\\u2028').replaceAll('\u2029', '\\u2029')
  return s + '\n'
}

/** The map with its keys sorted, as Go writes a map. */
function sorted(m: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const k of Object.keys(m).sort()) out[k] = m[k] as string
  return out
}

/** The batch the unit at `index` of the file gets when it is sent for translation alone. */
export function rebuildBatch(path: string, file: FileData, pf: PreviewFile, shared: PreviewShared, index: number): BatchJSON {
  const fu = file.units[index]
  const pu = pf.units[index]
  if (!fu || !pu) throw new Error(`${path}: unit ${index} has no preview (a copy is not sent for translation)`)
  const key = `${path}|${fu.p}`

  const glossary = pu.t.map((n): Term => {
    const t = pf.terms[n]
    if (!t) throw new Error(`${path}: no term ${n}`)
    return { ...t, units: [key] }
  })
  const keywords: Record<string, string> = {}
  for (const id of pu.k) keywords[id] = pf.keywords[id] ?? ''

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
  if (pu.em) unit.en = { value: fu.en ?? '', method: pu.em }
  if (pu.jm) unit.jp = { value: fu.jp ?? '', method: pu.jm }
  if (pu.it.refs && Object.keys(pu.it.refs).length > 0) unit.refs = sorted(pu.it.refs)
  unit.keep = pu.keep

  const context = pu.c.map((c): ContextLine => {
    const i = c[0] as number
    const cu = file.units[i]
    const spk = pf.units[i]?.sp ?? ''
    if (!cu) throw new Error(`${path}: no context unit ${i}`)
    return { key: `${path}|${cu.p}`, speaker: spk, kr: cu.kr, zh: c.length > 1 ? (c[1] as string) : cu.zh }
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
  batch.keywords = sorted(keywords)
  batch.units = [unit as BatchUnit]
  batch.context = context
  return batch as BatchJSON
}

/** The text a translator is handed: the prompt, every guide of the batch, and the batch JSON. */
export function fullText(shared: PreviewShared, batch: BatchJSON): string {
  const parts = [`# prompts/translate.md\n\n${shared.prompt}`]
  for (const path of batch.guides) {
    const g = shared.guides.find((x) => x.path === path)
    parts.push(`# ${path}\n\n${g?.text ?? ''}`)
  }
  parts.push(`# 批次 JSON\n\n${goJSON(batch)}`)
  return parts.join('\n\n')
}
