// The JSON the viewer reads. Keys and shapes are those of internal/site
// (file.go and export.go) and of the site spec, section 5; format 2.

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
  /** The scope proposals nobody declined. */
  scope: number
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
  /** The units of the variant by their JP and EN, the most units first. */
  jpen?: JPENGroup[]
}

/** Units of a variant with one JP and one EN string; jp or en is left out where the units have none. */
export interface JPENGroup {
  jp?: string
  en?: string
  count: number
  keys: string[]
}

/** Whether JP and EN tell the translations of an inconsistent group apart. */
export type JPENClass = 'same' | 'explained' | 'partial'

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
  jpen_class?: JPENClass
  /** A provisional term (kind "term-new") only: the term the card is about. */
  term?: TermRef
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
  /** A term-missing group only: the glossary entry, and every unit that lacks its rendering (keys: those not accepted). */
  term?: TermRef
  units?: TermUnit[]
}

/** A glossary entry: the original term and its rendering. */
export interface TermRef {
  kr: string
  zh: string
}

/** One unit of a term-missing group; id is its own warning id. */
export interface TermUnit {
  key: string
  id: string
  kr: string
  zh: string
  note?: string
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
export type ReviewTab = 'suggestions' | 'warnings' | 'notes' | 'todo' | 'scope'

/** The documents of data/review/, by page. */
export interface ReviewData {
  suggestions: { items: SuggestionItem[] }
  warnings: { items: WarningItem[] }
  notes: { items: NoteItem[] }
  todo: { sections: TodoSection[] }
  scope: ScopeReport
}

/** The limits the scope proposals are made by (review/scope.json, internal/scope). */
export interface ScopeThresholds {
  min_n: number
  used: number
  not_used: number
  explained: number
  lost_units: number
  lost_share: number
}

/** The units an entry hits in one family ("family:<id>"), one directory group ("StoryData/**") or the rest ("rest"). */
export interface ScopeGroup {
  name: string
  kind: 'family' | 'dir'
  n: number
  used: number
}

export interface ScopeCount {
  n: number
  used: number
}

/** One approved glossary entry and its counts. */
export interface ScopeEntry {
  id: string
  kr: string
  zh: string
  also: string[]
  files: string[]
  not_files: string[]
  match: string
  notes: string
  groups: ScopeGroup[]
  compound: ScopeCount
  explained: number
  unexplained: number
}

export type ScopeKind = 'not-files' | 'files' | 'substring' | 'word' | 'mixed' | 'explained'

export interface ScopeExample {
  key: string
  group: string
  kr: string
  zh: string
}

/** One change of an entry, or one entry a person should look at. */
export interface ScopeProposal {
  id: string
  kind: ScopeKind
  kr: string
  zh: string
  patterns: string[]
  /** 'substring' or 'word' on those two kinds, '' on the others. */
  match: string
  mixed_groups: string[]
  warn_removed: number
  warn_added: number
  used_lost: number
  used_gained: number
  flagged: boolean
  examples: ScopeExample[]
  declined?: Confirmed
}

/** review/scope.json. */
export interface ScopeReport {
  thresholds: ScopeThresholds
  entries: ScopeEntry[]
  proposals: ScopeProposal[]
}

/** What changed in a unit between two releases. */
export type UpdateKind = 'new' | 'kr-changed' | 'deleted' | 'zh-changed'
/** Why a zh-changed unit changed. */
export type UpdateCause = 'adjustment' | 'late' | 'retranslated'

/** The number of changed units of each kind. */
export interface UpdateCounts {
  new: number
  kr_changed: number
  deleted: number
  zh_changed: number
}

/** One release in data/updates/index.json (newest first). */
export interface UpdateRelease {
  tag: string
  released_at: string
  prev_tag?: string
  raw_tag: string
  prev_raw_tag?: string
  kind: 'game' | 'rebuild' | 'first'
  counts: UpdateCounts
  copy_counts: UpdateCounts
  other_files: number
  files: number
  /** false for the first release and for one whose record is not made yet: no counts and no page. */
  recorded: boolean
}

/** One changed unit of data/updates/<tag>.json. */
export interface UpdateEntry {
  file: string
  field: string
  ptr: string
  old_ptr?: string
  kind: UpdateKind
  old_kr?: string
  new_kr?: string
  old_zh?: string
  new_zh?: string
  old_origin?: string
  new_origin?: string
  copy?: boolean
  cause?: UpdateCause
}

/** data/updates/<tag>.json: what one release changed. */
export interface UpdateRecord {
  tag: string
  prev_tag: string
  released_at: string
  prev_released_at: string
  raw_tag: string
  prev_raw_tag: string
  rules_commit: string
  prev_rules_commit: string
  counts: UpdateCounts
  copy_counts: UpdateCounts
  other_files: string[]
  entries: UpdateEntry[]
}

/** The `lang` attribute of the text of each language, so that the browser picks the right glyphs. */
export const LANG_ATTR: Record<Lang, string> = {
  kr: 'ko',
  jp: 'ja',
  en: 'en',
  zh: 'zh-Hant-TW',
}
