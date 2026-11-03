// The JSON the viewer reads. Keys and shapes are those of internal/site
// (file.go and export.go) and of the site spec, section 5; format 1.

/** data/meta.json: where the data comes from and how much there is. */
export interface Meta {
  /** The version of the layout and keys of the data; 1. */
  format: number
  /** The snapshot of the KR, JP and EN texts. */
  raw_tag: string
  /** The rules commit the pack was built under. */
  rules_commit: string
  /** The commit of the tw branch the TW texts are from. */
  tw_commit: string
  /** The Release tag of tw_commit; "" when it has none. */
  release: string
  /** The languages of the data and of the search corpus: kr, jp, en, zh. */
  langs: Lang[]
  files: number
  units: number
}

/** A family of files in data/index.json: files are positions in Index.files. */
export interface IndexFamily {
  id: string
  files: number[]
}

/** What the index says of one file. */
export interface FileSummary {
  path: string
  family: string
  /** The place a story starts in; "" outside StoryData/. */
  label: string
  /** The tellers of the file, each once, first seen first. */
  speakers: string[]
  units: number
  /** The number of units by Unit.o; an origin with no unit is not a key. */
  origins: Partial<Record<Origin, number>>
  /** The KR strings of the file that the language has no string for. */
  missing: { en: number; jp: number }
}

/** data/index.json. The position of a file in files is its number in search/keys.json. */
export interface Index {
  families: IndexFamily[]
  files: FileSummary[]
}

/**
 * Where a TW text came from: the origin of the unit's record, or `stand-in`
 * when there is no record or the built pack has another string than the record.
 */
export type Origin = 'paratranz' | 'claude' | 'human' | 'copy' | 'fallback' | 'stand-in'

/** The four languages of a unit; zh is the TW text. */
export type Lang = 'kr' | 'jp' | 'en' | 'zh'

/** One string of a file in the four languages. */
export interface Unit {
  /** JSON pointer. */
  p: string
  /** Field name. */
  f: string
  /** Index of the dataList element, or -1. */
  i: number
  kr: string
  /** Left out where the language has no string aligned to the unit. */
  jp?: string
  en?: string
  /** The string of the built pack, what a player sees. */
  zh: string
  o: Origin
  /** Flags of the record, such as needs_review; left out when there are none. */
  fl?: string[]
}

/** data/files/<path>.json. A file with no strings has units: []. */
export interface FileData {
  path: string
  family: string
  units: Unit[]
}

/** The `lang` attribute of the text of each language, so that the browser picks the right glyphs. */
export const LANG_ATTR: Record<Lang, string> = {
  kr: 'ko',
  jp: 'ja',
  en: 'en',
  zh: 'zh-Hant-TW',
}
