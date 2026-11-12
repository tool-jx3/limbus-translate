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
  /** Whether a missing glossary rendering sends a batch back: "hard", or "warn". */
  glossary: 'hard' | 'warn'
  /** The nodes the routing lists for the family; [] for the fallback. */
  nodes: string[]
}

/** The third-level mark of a file. */
export interface FileTag {
  kind: 'character' | 'chapter' | 'event'
  value: string
  label: string
}

/** A node of the group tree of the sidebar; files are positions in Index.files, listed directly in the group. */
export interface IndexGroup {
  kind: 'top' | 'sub' | 'chapter' | 'character' | 'event'
  node: string
  label: string
  files: number[]
  children: IndexGroup[]
}

/** What the index says of one file. */
export interface FileSummary {
  path: string
  family: string
  /** The node of the file in the group tree, and its sub-node, if any. */
  top: string
  sub?: string
  /** The third-level mark, if any. */
  tag?: FileTag
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
  /** The group tree of the sidebar, folded and ordered. */
  groups: IndexGroup[]
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
  /** A provisional term (kind "term-new", or "name" when one is merged into it) only: the term the card is about. */
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
  /** An entry is a hard candidate with a rate above hard_rate over at least hard_min_n units. */
  hard_rate: number
  hard_min_n: number
}

/** The units an entry hits in one family ("family:<id>") or one top node of the file tree ("group:StoryData/", "group:Skills"). */
export interface ScopeGroup {
  name: string
  kind: 'family' | 'group'
  n: number
  used: number
  /** Of the units that do not use the rendering: every aligned EN/JP that can be compared holds the entry's en/jp (same), every one misses it (other), or there is nothing to compare or EN and JP disagree (unknown). */
  same: number
  other: number
  unknown: number
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
  /** The Japanese and English of the entry (one form, or a list of the forms of the word, any of which counts: Forms), and the sense it stands for with where that comes from (game:<file>#<pointer> or a limbuscompany.wiki.gg URL); each is absent where the glossary has none. */
  jp?: Forms
  en?: Forms
  meaning?: string
  sources?: string[]
  also: string[]
  files: string[]
  not_files: string[]
  match: string
  /** 'hard'; 'off' for general vocabulary, which warns of no missing rendering (its counts are for information, it is never a hard candidate and has no proposal); or absent for an entry that only warns. */
  check?: string
  notes: string
  groups: ScopeGroup[]
  /** The units the entry hits and how many use the rendering (the real hits of Report.Units: a substring entry's compound units count; the groups leave them out), their rate, and whether the entry is worth considering for check: hard. */
  n: number
  used: number
  rate: number
  hard_candidate: boolean
  compound: ScopeCount
  explained: number
  unexplained: number
  /** The units that do not use the rendering by sense, as in ScopeGroup. */
  same: number
  other: number
  unknown: number
  /** At most 3 units of each group that does not use the rendering. */
  examples: ScopeExample[]
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

/** data/glossary/index.json: the entries of the glossary with their counts, and the limits the hard candidates follow. */
export interface GlossaryIndex {
  thresholds: ScopeThresholds
  entries: ScopeEntry[]
}

/** One unit an entry hits (data/glossary/<a>-<b>.json). A unit that lacks the rendering has its full KR and TW (cut at 400 characters), its id and, if a person confirmed it, accepted; a unit that uses it has a piece of each around the term. */
export interface GlossaryUnit {
  /** "<file path>|<pointer>". */
  key: string
  family: string
  top: string
  sub?: string
  tag?: FileTag
  origin: Origin
  kr: string
  zh: string
  note?: string
  /** warning/term-missing/…, on a unit without the rendering only. */
  id?: string
  accepted?: Confirmed
}

/** data/glossary/<a>-<b>.json, the file of the entry glossary/<a>/<b>. */
export interface GlossaryFile {
  id: string
  missing: GlossaryUnit[]
  used: GlossaryUnit[]
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
  /** Why the unit changed (Release records from causes_version 1 on). */
  causes?: UpdateEntryCause[]
}

/** One reason an entry changed; the rule kinds always carry `id`. */
export interface UpdateEntryCause {
  kind: string
  id?: string
  batch?: string
  reason?: string
  note?: string
}

/** A rule an entry cause points to; the key in UpdateRecord.rules is `<kind>:<id>`. */
export interface UpdateRuleInfo {
  kind: string
  id: string
  path: string
  title: string
  commit: string
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
  /** Absent on a record made before causes existed. */
  causes_version?: number
  /** The rules the entries' causes point to, keyed `<kind>:<id>`. */
  rules?: Record<string, UpdateRuleInfo>
  /** The number of entries with an unknown cause. */
  unknown?: number
}

/** The `lang` attribute of the text of each language, so that the browser picks the right glyphs. */
export const LANG_ATTR: Record<Lang, string> = {
  kr: 'ko',
  jp: 'ja',
  en: 'en',
  zh: 'zh-Hant-TW',
}

// The translation preview (internal/site/preview.go): data/preview.json and
// data/preview/<path>. The batch of a unit is rebuilt from them and files/<path>.
// data/preview/<path> is compact (the *Wire types): what a reader can put back
// is left out. expandPreviewFile (preview/decode.ts) turns it into PreviewFile,
// which is what the page and rebuildBatch work on.

/** A guide file of data/preview.json. */
export interface PreviewGuide {
  path: string
  text: string
}

/** data/preview.json: what every batch shares. */
export interface PreviewShared {
  rules_commit: string
  /** The text of prompts/translate.md. */
  prompt: string
  /** Numbered in the order some batch names them first; PreviewUnit.g are numbers in this list. */
  guides: PreviewGuide[]
}

/** The jp or en of a glossary entry (rules.Forms): one form as a string, several as a list. */
export type Forms = string | string[]

/** A glossary entry (rules.Term) in the order of the Go fields; `units` is a batch's only. */
export interface Term {
  kr: string
  jp?: Forms
  en?: Forms
  zh: string
  status: string
  notes?: string
  forbidden?: string[]
  also?: string[]
  category?: string[]
  files?: string[]
  not_files?: string[]
  match?: string
  /** 'hard', 'off' (no warning of a missing rendering) or 'warn', absent for warn. */
  check?: string
  provisional?: boolean
  units?: string[]
}

/** What a translation must keep from KR (batch.Constraints). */
export interface Constraints {
  ids: string[]
  placeholders: string[]
  tags: string[]
  dante_wrapper: boolean
}

/** The part of a batch unit that comes from the pending item. */
export interface PreviewItem {
  reason: string
  prev_kr?: string
  prev_zh?: string
  draft?: string
  hint?: string
  refs?: Record<string, string>
}

export interface PreviewState {
  pending: boolean
  needs_review: boolean
  leased_to?: string
  frozen: boolean
  attempts?: number
  last_error?: string
}

export interface PreviewRecord {
  origin: string
  model?: string
  batch?: string
  since?: string
  rules?: string
  flags?: string[]
  /** True when the record was made from another KR text than the present one. */
  stale: boolean
}

export interface PreviewCheck {
  verdict: string
  zh: string
  issues: unknown[]
}

/** What the batch of one unit has that files/<path> and PreviewFile do not. */
export interface PreviewUnit {
  sp: string
  /** Guides: numbers in PreviewShared.guides. */
  g: number[]
  /** Glossary entries: numbers in PreviewFile.terms. */
  t: number[]
  /** Keyword ids, sorted. */
  k: string[]
  /** Existing names: numbers in PreviewFile.names. */
  n: number[]
  it: PreviewItem
  /** The methods of the aligned EN and JP strings. */
  em?: string
  jm?: string
  keep: Constraints
  /** Lines of context: [unit number] or [unit number, another translation]. */
  c: (number | string)[][]
  st: PreviewState
  rec?: PreviewRecord
  chk?: PreviewCheck
  /** The notes on the unit, joined by a line feed. */
  nt?: string
}

/** data/preview/<path>; units has the length and order of files/<path>, null for a copy. */
export interface PreviewFile {
  family: string
  glossary_check: string
  terms: Term[]
  /** The existing names of the batches of the file, each once; their units are null. */
  names: Omit<NameRef, 'units'>[]
  keywords: Record<string, string>
  units: (PreviewUnit | null)[]
}

/** An existing name a batch's units use (batch.NameRef). */
export interface NameRef {
  kr: string
  zh: string
  count: number
  others?: { zh: string; units: number }[]
  jp?: string
  en?: string
  /** The keys of the units of the batch the name is for. */
  units: string[]
}

export interface BatchUnit {
  key: string
  file: string
  field: string
  speaker: string
  kr: string
  reason: string
  prev_kr?: string
  prev_zh?: string
  draft?: string
  hint?: string
  en?: { value: string; method: string }
  jp?: { value: string; method: string }
  refs?: Record<string, string>
  keep: Constraints
}

export interface ContextLine {
  key: string
  speaker: string
  kr: string
  zh: string
}

/** The batch JSON `batch next` writes (batch.Batch), keys in Go's order. */
export interface BatchJSON {
  id: string
  family: string
  glossary_check?: string
  file: string
  files: string[]
  guides: string[]
  glossary: Term[]
  keywords: Record<string, string>
  names: NameRef[]
  units: BatchUnit[]
  context: ContextLine[]
}

/** PreviewFile.rd: the values most records of a file have; a record leaves out the fields equal to them. */
export interface PreviewRecDefaults {
  origin: string
  rules: string
  since: string
}

/** PreviewState as written: a field at its zero value is left out, and so is st when all are. */
export interface PreviewStateWire {
  pending?: boolean
  needs_review?: boolean
  leased_to?: string
  frozen?: boolean
  attempts?: number
  last_error?: string
}

/** PreviewRecord as written: origin, rules and since are left out where they equal PreviewFile.rd. */
export interface PreviewRecordWire {
  origin?: string
  model?: string
  batch?: string
  since?: string
  rules?: string
  flags?: string[]
  stale?: boolean
}

/** PreviewCheck as written; a record whose check accepts its file text with no issue has none. */
export interface PreviewCheckWire {
  verdict: string
  /** Absent where the text checked is the zh of the unit in files/<path>. */
  zh?: string
  issues?: unknown[]
}

/** PreviewUnit as written. */
export interface PreviewUnitWire {
  sp: string
  g: number[]
  t: number[]
  k: string[]
  n: number[]
  /** prev_zh is left out where pz is 1. */
  it: PreviewItem
  /** 1: the prev_zh is the zh of the unit in files/<path>. */
  pz?: number
  /** Methods of the aligned EN and JP strings, one character each: p pointer, i id, s sequence, - none. Absent when neither. */
  m?: string
  /** Absent when nothing is to be kept. */
  keep?: Constraints
  c: (number | string)[][]
  st?: PreviewStateWire
  rec?: PreviewRecordWire
  chk?: PreviewCheckWire
  nt?: string
}

/** data/preview/<path> as written. */
export interface PreviewFileWire {
  family: string
  glossary_check: string
  terms: Term[]
  names: Omit<NameRef, 'units'>[]
  keywords: Record<string, string>
  /** Present when some unit has a record. */
  rd?: PreviewRecDefaults
  units: (PreviewUnitWire | null)[]
}

/** A unit of the gold sample (data/gold.json). zh is the translation of the record; shipped is the text now built into the pack, when it differs; stale: the record's translation is not the one that was judged. */
export interface GoldUnit {
  key: string
  path: string
  ptr: string
  family: string
  origin: string
  kr: string
  jp?: string
  en?: string
  zh: string
  /** The built text in the pack when it differs from zh (the record's translation). */
  shipped?: string
  verdict: '' | 'correct' | 'wrong' | 'unsure'
  note: string
  stale: boolean
}

/** A row of the report: a family and an origin (origin is absent in the rows of a family; both are empty in the total). */
export interface GoldRow {
  family: string
  origin?: string
  sampled: number
  judged: number
  correct: number
  wrong: number
  unsure: number
  accuracy: number | null
  stale: number
}

export interface GoldReport {
  rows: GoldRow[]
  families: GoldRow[]
  total: GoldRow
  /** Units of the sample that are not in the pack. */
  missing?: number
  /** True when the report was made without the raw worktree: the KR check was left out and missing is not known. */
  kr_unknown?: boolean
}

export interface GoldData {
  per_family?: number
  /** Units of the sample that are not in the pack. */
  missing?: number
  units: GoldUnit[]
  report?: GoldReport
}

/** One rendering of a name, with the units that carry it. */
export interface NameRendering {
  zh: string
  units: number
}

/** One JP/EN pair a KR name has, with the renderings it got (most units first) and example unit keys ("<file path>|<pointer>"). */
export interface NameVariant {
  jp?: string
  en?: string
  renderings: NameRendering[]
  examples: string[]
}

/** One KR name of the table of existing names. */
export interface NameRow {
  kr: string
  units: number
  variants: NameVariant[]
  /** The glossary has an entry for the name. */
  glossary?: true
  /** The name was declined. */
  declined?: true
  /** The review has a different new rendering for the name. */
  review?: true
}

/** data/names.json: the table of existing names (KR order), made before the last update of the translations. */
export interface NamesData {
  /** The snapshot the table was made before; "" when there is no table. */
  raw_tag: string
  /** The name fields the table was made from. */
  fields: string[]
  names: NameRow[]
}
