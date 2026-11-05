import { useContext, useState } from 'preact/hooks'
import { loadFile } from '../data'
import { variantDiff } from '../review/diff'
import { DiffText, ShownText } from '../review/parts'
import { fileHref } from '../route'
import { TmpText } from '../tmp/TmpText'
import { LANG_ATTR, type Lang, type Origin, type Unit, type UpdateCause, type UpdateEntry, type UpdateKind } from '../types'
import { OriginBadge } from '../view/OriginBadge'

export const KIND_NAMES: Record<UpdateKind, { name: string; cls: string }> = {
  new: { name: '新增', cls: 'is-new' },
  'kr-changed': { name: 'KR 修改', cls: 'is-kr' },
  deleted: { name: '刪除', cls: 'is-deleted' },
  'zh-changed': { name: '只有譯文改變', cls: 'is-zh' },
}

const CAUSE_NAMES: Record<UpdateCause, string> = {
  adjustment: '人工調整',
  late: '補翻',
  retranslated: '重翻',
}

/** The old and the new text of one language: the old above the new, what the other lacks marked. */
export function TextPair({ lang, old, now }: { lang: Lang; old: string | undefined; now: string | undefined }) {
  const { raw } = useContext(ShownText)
  if (old === undefined && now === undefined) return null
  // Only one of them: no diff, and not marked as old or new.
  if (old === undefined || now === undefined) {
    const text = (old ?? now) as string
    const isOld = old !== undefined
    return (
      <div lang={LANG_ATTR[lang]} class={isOld ? 'uc-old' : 'uc-new'}>
        <span class="uc-tag">{isOld ? '舊' : '新'}</span>
        <TmpText text={text} raw={raw} lang={lang} />
      </div>
    )
  }
  if (old === now) {
    return (
      <div lang={LANG_ATTR[lang]}>
        <TmpText text={now} raw={raw} lang={lang} />
      </div>
    )
  }
  return (
    <div lang={LANG_ATTR[lang]} class="uc-pair">
      <div class="uc-old">
        <span class="uc-tag">舊</span>
        <DiffText key={raw ? 'raw' : 'drawn'} pieces={variantDiff(now, old, { raw })} raw={raw} />
      </div>
      <div class="uc-new">
        <span class="uc-tag">新</span>
        <DiffText key={raw ? 'raw' : 'drawn'} pieces={variantDiff(old, now, { raw })} raw={raw} />
      </div>
    </div>
  )
}

function Pair({ label, lang, old, now, show }: { label: string; lang: Lang; old: string | undefined; now: string | undefined; show: boolean }) {
  if (!show || (old === undefined && now === undefined)) return null
  return (
    <div class="uc-row">
      <dt>{label}</dt>
      <dd>
        <TextPair lang={lang} old={old} now={now} />
      </dd>
    </div>
  )
}

/** The old and the new KR of an entry; a unit that only the translation changed has one KR, so they are the same. */
export function entryKr(e: UpdateEntry): { old: string | undefined; now: string | undefined } {
  return {
    old: e.old_kr ?? (e.kind === 'zh-changed' ? e.new_kr : undefined),
    now: e.new_kr ?? (e.kind === 'zh-changed' ? e.old_kr : undefined),
  }
}

/** The origin before and after (either may be missing). */
export function Origins({ entry: e }: { entry: UpdateEntry }) {
  const oldOrigin = e.old_origin as Origin | undefined
  const newOrigin = e.new_origin as Origin | undefined
  return (
    <span class="uc-origins">
      {oldOrigin && <OriginBadge origin={oldOrigin} />}
      {oldOrigin && newOrigin && <span aria-label="改成">→</span>}
      {newOrigin && <OriginBadge origin={newOrigin} />}
    </span>
  )
}

export function CauseChip({ cause }: { cause: UpdateCause }) {
  return <span class="uc-cause">{CAUSE_NAMES[cause]}</span>
}

type Current = { status: 'loading' } | { status: 'error' } | { status: 'ready'; unit: Unit | undefined }

/**
 * The JP and EN of the unit as the file now has them; the file is loaded when
 * this is opened. The unit is found by its pointer, which an older Release may
 * no longer have at the same place: one whose KR is not `kr` is another unit.
 */
function CurrentJPEN({ file, ptr, kr }: { file: string; ptr: string; kr: string | undefined }) {
  const { langs, raw } = useContext(ShownText)
  const [state, setState] = useState<Current>()
  if (!langs.jp && !langs.en) return null
  const open = (e: Event) => {
    if (!(e.currentTarget as HTMLDetailsElement).open || state) return
    setState({ status: 'loading' })
    loadFile(file).then(
      (data) => setState({ status: 'ready', unit: data.units.find((u) => u.p === ptr) }),
      () => setState({ status: 'error' }),
    )
  }
  return (
    <details class="uc-jpen" onToggle={open}>
      <summary>JP／EN（目前的）</summary>
      {state?.status === 'loading' && <p role="status">載入中…</p>}
      {state?.status === 'error' && <p role="status">讀不到這個檔案</p>}
      {state?.status === 'ready' && !state.unit && <p role="status">目前的檔案裡找不到這個單位</p>}
      {state?.status === 'ready' && state.unit && kr !== undefined && state.unit.kr !== kr && (
        <p role="status">目前檔案的這個位置已是另一句</p>
      )}
      {state?.status === 'ready' && state.unit && (kr === undefined || state.unit.kr === kr) && (
        <dl class="uc-cur">
          {langs.jp && (
            <div class="uc-row">
              <dt>JP</dt>
              <dd lang={LANG_ATTR.jp}>{state.unit.jp === undefined ? '—' : <TmpText text={state.unit.jp} raw={raw} lang="jp" />}</dd>
            </div>
          )}
          {langs.en && (
            <div class="uc-row">
              <dt>EN</dt>
              <dd lang={LANG_ATTR.en}>{state.unit.en === undefined ? '—' : <TmpText text={state.unit.en} raw={raw} lang="en" />}</dd>
            </div>
          )}
        </dl>
      )}
    </details>
  )
}

/** One changed unit: what it was and what it is, old above new. */
export function UpdateCard({ entry: e }: { entry: UpdateEntry }) {
  const { langs } = useContext(ShownText)
  const kind = KIND_NAMES[e.kind]
  const deleted = e.kind === 'deleted'
  const kr = entryKr(e)
  return (
    <article class="uc-card">
      <div class="uc-head">
        <span class={`uc-kind ${kind.cls}`}>{kind.name}</span>
        {e.cause && <CauseChip cause={e.cause} />}
        <code class="uc-where">
          {e.field} {e.ptr}
        </code>
        <Origins entry={e} />
      </div>
      <dl class="uc-texts">
        <Pair label="KR" lang="kr" old={kr.old} now={kr.now} show={langs.kr} />
        <Pair label="TW" lang="zh" old={e.old_zh} now={e.new_zh} show={langs.zh} />
      </dl>
      {!deleted && (
        <div class="uc-foot">
          <CurrentJPEN file={e.file} ptr={e.ptr} kr={e.new_kr} />
          <a class="uc-open" href={fileHref(e.file, e.ptr)}>
            在檔案中開啟 →
          </a>
        </div>
      )}
    </article>
  )
}
