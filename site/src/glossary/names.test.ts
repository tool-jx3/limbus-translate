import { describe, expect, it } from 'vitest'
import { fixture } from '../testdata'
import type { NamesData, NameRow } from '../types'
import { filterNames, sortNames } from './names'

const rows = fixture<NamesData>('names.json').names
const krs = (r: NameRow[]) => r.map((x) => x.kr)

describe('filterNames', () => {
  it('keeps every row for an empty search and no filter', () => {
    expect(krs(filterNames(rows, '', ''))).toEqual(['검', '우레', '방패'])
  })

  it('searches the KR, the JP and EN of each variant (any case) and every rendering', () => {
    expect(krs(filterNames(rows, '우', ''))).toEqual(['우레'])
    expect(krs(filterNames(rows, '雷', ''))).toEqual(['우레'])
    expect(krs(filterNames(rows, 'SHIELD', ''))).toEqual(['방패'])
    expect(krs(filterNames(rows, 'blade', ''))).toEqual(['검'])
    expect(krs(filterNames(rows, '長劍', ''))).toEqual(['검'])
    expect(krs(filterNames(rows, '雷鳴', ''))).toEqual(['우레'])
    expect(krs(filterNames(rows, '沒有', ''))).toEqual([])
  })

  it('filters multi: a variant with two renderings, or two variants', () => {
    expect(krs(filterNames(rows, '', 'multi'))).toEqual(['검'])
    const twoVariants: NameRow = { kr: 'x', units: 2, variants: [{ renderings: [{ zh: 'a', units: 1 }], examples: [] }, { renderings: [{ zh: 'b', units: 1 }], examples: [] }] }
    const oneVariantTwoRenderings: NameRow = { kr: 'y', units: 2, variants: [{ renderings: [{ zh: 'a', units: 1 }, { zh: 'b', units: 1 }], examples: [] }] }
    expect(krs(filterNames([twoVariants, oneVariantTwoRenderings, rows[2]!], '', 'multi'))).toEqual(['x', 'y'])
  })

  it('filters review, glossary and declined', () => {
    expect(krs(filterNames(rows, '', 'review'))).toEqual(['검'])
    expect(krs(filterNames(rows, '', 'glossary'))).toEqual(['검'])
    expect(krs(filterNames(rows, '', 'declined'))).toEqual(['우레'])
  })

  it('combines the search and the filter', () => {
    expect(krs(filterNames(rows, '雷', 'review'))).toEqual([])
  })
})

describe('sortNames', () => {
  it('puts the rows with more units first, ties by KR', () => {
    const tied: NameRow[] = [{ ...rows[2]!, kr: 'b' }, { ...rows[2]!, kr: 'a' }, rows[0]!]
    expect(krs(sortNames(tied, 'units'))).toEqual(['검', 'a', 'b'])
  })

  it('sorts by KR', () => {
    expect(krs(sortNames(rows, 'kr'))).toEqual(['검', '방패', '우레'])
  })

  it('does not change the rows it is given', () => {
    const copy = [...rows]
    sortNames(rows, 'kr')
    expect(rows).toEqual(copy)
  })
})
