import type { DataCounts } from '../types'

/** The counts of a snapshot that differs in nothing. */
export const ZERO_COUNTS: DataCounts = {
  files_added: 0,
  files_removed: 0,
  new_groups: 0,
  gone_groups: 0,
  keys_added: 0,
  keys_removed: 0,
  entries_added: 0,
  entries_removed: 0,
  tokens_added: 0,
  tokens_removed: 0,
}

/** The counts as the list shows them: what is new first, then what is gone. */
export const COUNT_CHIPS: { key: keyof DataCounts; name: string; cls: 'is-new' | 'is-deleted' }[] = [
  { key: 'new_groups', name: '新種類', cls: 'is-new' },
  { key: 'files_added', name: '新檔案', cls: 'is-new' },
  { key: 'keys_added', name: '新欄位', cls: 'is-new' },
  { key: 'entries_added', name: '新條目', cls: 'is-new' },
  { key: 'tokens_added', name: '新標記', cls: 'is-new' },
  { key: 'gone_groups', name: '消失的種類', cls: 'is-deleted' },
  { key: 'files_removed', name: '移除檔案', cls: 'is-deleted' },
  { key: 'keys_removed', name: '移除欄位', cls: 'is-deleted' },
  { key: 'entries_removed', name: '移除條目', cls: 'is-deleted' },
  { key: 'tokens_removed', name: '消失的標記', cls: 'is-deleted' },
]
