import type { NamesOnly, NamesSort } from '../route'
import type { NameRow } from '../types'

const byText = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0)

/** Whether a name has more than one way: two variants, or a variant with two renderings. */
const isMulti = (r: NameRow) => r.variants.length >= 2 || r.variants.some((v) => v.renderings.length >= 2)

const ONLY: Record<Exclude<NamesOnly, ''>, (r: NameRow) => boolean> = {
  multi: isMulti,
  review: (r) => r.review === true,
  glossary: (r) => r.glossary === true,
  declined: (r) => r.declined === true,
}

/** The rows whose KR, any variant's JP or EN, or any rendering has the text q (any case), and that the only filter keeps; the order is kept. */
export function filterNames(rows: NameRow[], q: string, only: NamesOnly): NameRow[] {
  const needle = q.trim().toLowerCase()
  const keep = only === '' ? undefined : ONLY[only]
  return rows.filter(
    (r) =>
      (keep === undefined || keep(r)) &&
      (needle === '' ||
        [r.kr, ...r.variants.flatMap((v) => [v.jp ?? '', v.en ?? '', ...v.renderings.map((x) => x.zh)])].some((s) => s.toLowerCase().includes(needle))),
  )
}

/** A sorted copy: most units first with ties by KR, or by KR. */
export function sortNames(rows: NameRow[], sort: NamesSort): NameRow[] {
  const copy = [...rows]
  return sort === 'kr' ? copy.sort((a, b) => byText(a.kr, b.kr)) : copy.sort((a, b) => b.units - a.units || byText(a.kr, b.kr))
}
