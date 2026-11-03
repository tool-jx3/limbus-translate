// The JSON the viewer reads. Keys and shapes are those of internal/site
// (file.go and export.go) and of the site spec, section 5; format 1.

/** data/meta.json: where the data comes from and how much there is. */
export interface Meta {
  /** The version of the layout and keys of the data; 2. */
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
  /** The review items nobody confirmed, by page. */
  review: ReviewCounts
}

/** The number of unconfirmed items of each page of the review (the lines, for the to-do page). */
export interface ReviewCounts {
  suggestions: number
  warnings: number
  notes: number
  todo: number
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
  /** The ids of the unconfirmed review items the unit is in, sorted; left out when there are none. */
  rv?: string[]
  /** The translator's notes on the unit (confirmed ones too), joined by a line feed; left out when there are none. */
  nt?: string
  /** 1 when the unit is in the pending list (shown as waiting for retranslation); left out otherwise. */
  pq?: 1
}

/** data/files/<path>.json. A file with no strings has units: []. */
export interface FileData {
  path: string
  family: string
  units: Unit[]
}

/** A person's decision on a review item: why, and when (YYYY-MM-DD). */
export interface Confirmed {
  reason: string
  date: string
}

/** One row of the evidence of a rule suggestion; each key but count is left out when empty. */
export interface Evidence {
  /** The unit the row is about. */
  key?: string
  kr?: string
  zh?: string
  /** The whole text of a translator's note. */
  note?: string
  count?: number
}

/** One translation of a group of inconsistent translations, with every unit that shows it. */
export interface Variant {
  zh: string
  count: number
  /** Unit keys ("<file path>|<pointer>"), in file order and then unit order. */
  keys: string[]
}

/**
 * One rule suggestion of review/suggestions.json. family, field, kr, jp, en and
 * variants are those of kind "inconsistent" only; declined is left out of the
 * ones nobody confirmed.
 */
export interface SuggestionItem {
  id: string
  kind: string
  title: string
  family?: string
  field?: string
  kr?: string
  jp?: string
  en?: string
  variants?: Variant[]
  evidence: Evidence[]
  /** How many of the units of the item are in the pending list. */
  pending: number
  declined?: Confirmed
}

/** One check warning: a code about one KR text and its translation, with every unit that has the pair. */
export interface WarningItem {
  id: string
  code: string
  kr: string
  zh: string
  message: string
  keys: string[]
  families: string[]
  origins: string[]
  pending: number
  accepted?: Confirmed
}

/** One translator's note. id is made of the key as written in the notes; key is the unit it is placed on. */
export interface NoteItem {
  id: string
  key: string
  family: string
  batch: string
  note: string
  kr: string
  zh: string
  o: string
  pending: number
  declined?: Confirmed
}

/** One line of a to-do section; key is the unit it is about, if any. */
export interface TodoItem {
  text: string
  key?: string
}

/** One kind of case that only adjusting or retranslating solves. */
export interface TodoSection {
  kind: string
  title: string
  hint: string
  items: TodoItem[]
}

/** The pages of the review. */
export type ReviewTab = 'suggestions' | 'warnings' | 'notes' | 'todo'

/** The documents of data/review/, by page. */
export interface ReviewData {
  suggestions: { items: SuggestionItem[] }
  warnings: { items: WarningItem[] }
  notes: { items: NoteItem[] }
  todo: { sections: TodoSection[] }
}

/** The `lang` attribute of the text of each language, so that the browser picks the right glyphs. */
export const LANG_ATTR: Record<Lang, string> = {
  kr: 'ko',
  jp: 'ja',
  en: 'en',
  zh: 'zh-Hant-TW',
}
