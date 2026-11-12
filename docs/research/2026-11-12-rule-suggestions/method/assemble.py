"""Assemble the item lists and data files of the two research reports.

usage: python3 -I assemble.py <rv dir> <adjclone> <docs/research dir>

Reads <rv>/groups.json, <rv>/notes.json (extract.py), <rv>/out/*.json (the
per-chunk judgements), <rv>/fixcur.json (current TW of the keys that note
fixes touch), <rv>/kwspace.json (units the keyword-space rule would change)
and the review data of <adjclone>. Writes, under the docs dir:

  2026-11-12-rule-suggestions/items/<category>.md, data/judgements.json
  2026-11-12-translator-notes/items/<category>.md, items/keyword-space-*.md,
  data/judgements.json, data/keyword-space.json
and prints the summary tables for the two README files to <rv>/tables-*.md.
"""
import json, sys, os, glob, collections, re

RV, A, OUT = sys.argv[1], sys.argv[2], sys.argv[3]
SUG = json.load(open(A + '/site/public/data/review/suggestions.json', encoding='utf-8'))['items']
full_keys = {x['id']: {v['zh']: v['keys'] for v in x['variants']} for x in SUG}
sug_jpen = {x['id']: (x.get('jp'), x.get('en')) for x in SUG}
groups = {g['id']: g for g in json.load(open(RV + '/groups.json', encoding='utf-8'))}
notes = {n['id']: n for n in json.load(open(RV + '/notes.json', encoding='utf-8'))}
fixcur = json.load(open(RV + '/fixcur.json', encoding='utf-8'))
kwspace = json.load(open(RV + '/kwspace.json', encoding='utf-8'))

def load(prefix):
    out = collections.OrderedDict()
    for f in sorted(glob.glob(RV + '/out/%s*.json' % prefix)):
        for r in json.load(open(f, encoding='utf-8')):
            r['_chunk'] = os.path.basename(f)[:-5]
            out[r['id']] = r
    return out

gj, nj = load('g'), load('n')

# current TW of every unit, to find other texts that use a keyword name
TWZH = {}
for f in glob.glob(A + '/.work/tw/.meta/units/**/*.jsonl', recursive=True):
    for line in open(f, encoding='utf-8'):
        rec = json.loads(line)
        TWZH[rec['key']] = rec['zh']

def other_refs(r, c):
    """Units outside the group whose TW contains the old keyword name."""
    if groups[r['id']]['family'] != 'keywords' or groups[r['id']]['field'] != 'name':
        return None
    own = set(k for ks in full_keys[r['id']].values() for k in ks)
    return sorted(k for k, z in TWZH.items() if c['from'] in z and k not in own)
assert set(gj) == set(groups) and set(nj) == set(notes)

# Corrections made while assembling: a unify that left one variant unchanged.
CORRECTIONS = []
def correct(sid, frm, why):
    r = gj[sid]
    assert frm in {v['zh'] for v in groups[sid]['variants']}
    assert frm not in {c['from'] for c in r['changes']}
    r['changes'].append({'from': frm, 'to': r['target']})
    CORRECTIONS.append((sid, frm, r['target'], why))
correct('inconsistent/ui/content/9cd88aa5', '進入關卡時，您可以選擇所擁有的人格與E.G.O來組建隊伍。',
        '判讀為「統一」，但只列了另一種譯法的變更；補上這一種，兩個單位才會一致')
correct('inconsistent/ui/content/a4bdca8d', '選擇站點並進入戰鬥。',
        '判讀為「統一」，但只列了另一種譯法的變更；補上這一種，兩個單位才會一致')

# ---------- formatting helpers ----------
def esc(s):
    return '' if s is None else str(s).replace('|', '\\|')

def code(s, table=False):
    """Inline code for an exact text: newlines shown as ⏎, a leading or
    trailing space shown as ␣ so it cannot be missed. In a table cell a | is
    escaped; elsewhere it is not (a code span shows the backslash)."""
    if s is None or s == '':
        return '（空）'
    s = str(s)
    lead = len(s) - len(s.lstrip(' '))
    trail = len(s) - len(s.rstrip(' '))
    core = s.strip(' ')
    shown = ('␣' * lead + core + '␣' * trail) if core else '␣' * len(s)
    shown = shown.replace('\n', '⏎')
    if table:
        shown = shown.replace('|', '\\|')
    tick = '``' if '`' in shown else '`'
    pad = ' ' if shown.startswith('`') or shown.endswith('`') else ''
    return tick + pad + shown + pad + tick

def diff(frm, to, ctx=8):
    """The changed places of frm → to, each as …before〔old→new〕after…, with
    spaces shown as ␣ and newlines as ⏎."""
    import difflib
    def vis(t):
        return t.replace(' ', '␣').replace('\n', '⏎')
    sm = difflib.SequenceMatcher(None, frm, to, autojunk=False)
    parts = []
    for op, a1, a2, b1, b2 in sm.get_opcodes():
        if op == 'equal':
            continue
        before = frm[max(0, a1 - ctx):a1]
        after = frm[a2:a2 + ctx]
        parts.append('%s%s〔%s→%s〕%s%s' % ('…' if a1 - ctx > 0 else '', vis(before), vis(frm[a1:a2]) or '∅',
                                            vis(to[b1:b2]) or '∅', vis(after), '…' if a2 + ctx < len(frm) else ''))
    return parts

def diff_line(frm, to, indent):
    ps = diff(frm, to)
    return '%s差異：%s' % (indent, '；'.join(code(p) for p in ps))

def files_of(keys):
    c = collections.Counter(k.split('|', 1)[0] for k in keys)
    return '、'.join('%s×%d' % (f, n) if n > 1 else f for f, n in sorted(c.items()))

def counter(d):
    return '、'.join('%s %d' % (k or '（無）', v) for k, v in sorted(d.items(), key=lambda t: -t[1]))

GCAT = collections.OrderedDict([
    ('I1', '格式差異'), ('I2', '依 JP/EN 本來就不同'), ('I3', '說話者、語氣不同'), ('I4', '同義措辭不同'),
    ('I5', '名稱或術語不一致'), ('I6', '有一種譯法有誤'), ('I7', 'KR 相同但語境不同'), ('I9', '其他')])
NCAT = collections.OrderedDict([
    ('N1', '沿用既有譯名或譯文的說明'), ('N2', '新名稱或新詞的譯法'), ('N3', '詞彙表或指引的例外說明'),
    ('N4', '指出別的單位或檔案也該一併修正'), ('N5', '譯者不確定、建議人工決定'), ('N6', '原文問題'),
    ('N7', '依規則、hint 或草稿的處理說明'), ('N8', '角色語氣、稱謂的判斷說明'), ('N9', '其他')])
GACT = collections.OrderedDict([('unify', '統一'), ('fix', '只修錯的譯法'), ('glossary', '加詞條並統一'),
                                ('decide', '交給你決定'), ('keep', '保留')])
NACT = collections.OrderedDict([('fix', '修改譯文'), ('glossary', '加詞條'), ('decide', '交給你決定'),
                                ('source', '原文問題，只記錄'), ('none', '確認即可')])
JPEN = {'explained': '每種 JP/EN 只對到一種譯法', 'partial': '部分譯法對到不同 JP/EN', 'same': 'JP/EN 全同'}
CONF = {'high': '高', 'medium': '中', 'low': '低'}

def render_options(opts):
    lines = []
    if isinstance(opts, str):
        return ['  - ' + esc(opts)]
    for i, o in enumerate(opts, 1):
        if isinstance(o, str):
            lines.append('  %d. %s' % (i, esc(o)))
            continue
        label = o.get('label') or o.get('option') or ''
        does = o.get('does') or o.get('do') or o.get('effect') or ''
        lines.append('  %d. %s%s' % (i, ('**%s**：' % esc(label)) if label else '', esc(does)))
        for c in o.get('changes') or []:
            if 'key' in c:
                lines.append('     - `%s`：%s → %s' % (c['key'], code(c['from']), code(c['to'])))
            else:
                lines.append('     - %s → %s' % (code(c.get('from')), code(c.get('to'))))
    return lines

# ---------- rule suggestions ----------
GORDER = list(GACT)
gitems = sorted(gj.values(), key=lambda r: (list(GCAT).index(r['category']), GORDER.index(r['action']),
                                            groups[r['id']]['family'], groups[r['id']]['field'],
                                            -groups[r['id']]['units'], r['id']))
for i, r in enumerate(gitems, 1):
    r['_seq'] = 'G%03d' % i

def g_units_changed(r):
    fk = full_keys[r['id']]
    return sum(len(fk[c['from']]) for c in r.get('changes') or [])

def g_item(r):
    g = groups[r['id']]
    fk = full_keys[r['id']]
    vs = {v['zh']: v for v in g['variants']}
    flags = []
    if r['confidence'] == 'low':
        flags.append('⚠ 低信心')
    if r['action'] == 'unify' and r.get('target') not in vs:
        flags.append('⚠ 統一後是新寫法（不是現有任何一種）')
    if any(other_refs(r, c) for c in r.get('changes') or []):
        flags.append('⚠ 改關鍵字名稱，別處有引用')
    if any(c[0] == r['id'] for c in CORRECTIONS):
        flags.append('⚠ 整理時補上一項變更（見報告「整理時的更正」）')
    out = ['### %s　%s｜%s %s｜信心%s%s' % (r['_seq'], GACT[r['action']], r['category'], GCAT[r['category']],
                                         CONF[r['confidence']], ('　' + '　'.join(flags)) if flags else ''), '']
    out.append('`%s`　%s／%s，%d 個單位，%s' % (r['id'], g['family'], g['field'], g['units'],
                                         JPEN.get(g['jpen_class'], g['jpen_class'] or '')))
    out.append('')
    out.append('- KR：%s' % code(g['kr']))
    jp, en = sug_jpen[r['id']]
    if g['jpen_class'] == 'same' and (jp or en):
        out.append('- JP：%s　EN：%s' % (code(jp), code(en)))
    out.append('- 現有譯法：')
    for n, v in enumerate(g['variants'], 1):
        extra = ['%d 個單位' % v['count'], '來源 ' + counter(v['origins'])]
        sp = {k: c for k, c in v['speakers'].items() if k}
        if sp:
            extra.append('說話者 ' + counter(sp))
        out.append('  %d. %s（%s）' % (n, code(v['zh']), '；'.join(extra)))
        if g['jpen_class'] != 'same':
            for j in v['jpen']:
                out.append('     - JP %s／EN %s（%d）' % (code(j['jp']), code(j['en']), j['count']))
    out.append('- 判讀：%s' % esc(r['reason']))
    act = r['action']
    if act == 'keep':
        out.append('- 建議：各譯法都保留，不改譯文')
        out.append('- 同意後寫入：`rules/declined.yaml` 一筆（`id` 為上面的 id，`reason` 為判讀，`date` 為同意當天）')
    elif act == 'decide':
        out.append('- 建議：交給你決定，選項：')
        out.extend(render_options(r.get('options') or []))
        out.append('- 同意後寫入：依你選的選項；選「保留」寫 `rules/declined.yaml`，選改寫則每個改動寫一個 `rules/rewrite/` 改寫檔（寫法同「統一」）；選項寫「須擬稿」的，先給你看擬好的譯文')
    else:
        if act == 'unify':
            out.append('- 建議：統一為 %s' % code(r['target']))
        elif act == 'fix':
            out.append('- 建議：只改下列譯法，其他保留')
        elif act == 'glossary':
            gl = r['glossary']
            out.append('- 建議：在 `glossary/terms.yaml` 加詞條 KR %s → %s（%s），並統一' % (
                code(gl.get('kr')), code(gl.get('zh')), esc(gl.get('note') or '')))
        out.append('- 會改：')
        for c in r['changes']:
            keys = fk[c['from']]
            out.append('  - %s → %s：%d 個單位（%s）' % (code(c['from']), code(c['to']), len(keys), files_of(keys)))
            out.append(diff_line(c['from'], c['to'], '    - '))
            refs = other_refs(r, c)
            if refs:
                out.append('    - ⚠ 這是關鍵字名稱；另有 %d 個單位的譯文含 %s，不在本組，**不會被改**（同意時要另外決定）：%s%s' % (
                    len(refs), code(c['from']), '、'.join('`%s`' % k for k in refs[:12]), '…' if len(refs) > 12 else ''))
        kept = [v for v in g['variants'] if v['zh'] not in {c['from'] for c in r['changes']}]
        if kept:
            out.append('  - 不改：' + '、'.join('%s（%d）' % (code(v['zh']), v['count']) for v in kept))
        out.append('- 同意後寫入：每個「會改」一個 `rules/rewrite/` 改寫檔（`files` 為上列檔案，`pattern` 與 `kr_pattern` 逐字比對整句，`replace` 為新譯文）%s；這個 id 不再列出' % (
            '，並在 `glossary/terms.yaml` 加上述詞條' if act == 'glossary' else ''))
    out.append('')
    return out

LIMIT = 300 * 1024

def write_cat(d, c, title, sections, link):
    """Write items/<c>.md, or, when it would be larger than LIMIT, one file per
    section (items/<c>-<action>.md, further cut into -1, -2 … when needed) and
    an index at items/<c>.md."""
    head = ['# ' + title, '', link, '']
    whole = head + [l for _, _, ls in sections for l in ls]
    if len('\n'.join(whole).encode()) <= LIMIT:
        write(d + '/items/%s.md' % c, whole)
        return
    index = head + ['這一類較長，分成下列檔案：', '']
    for act, label, ls in sections:
        # cut at item boundaries
        items, cur = [], []
        for l in ls[2:]:
            if l.startswith('### ') and cur:
                items.append(cur); cur = []
            cur.append(l)
        if cur:
            items.append(cur)
        parts, part, size = [], [], 0
        for it in items:
            n = len('\n'.join(it).encode())
            if part and size + n > LIMIT:
                parts.append(part); part, size = [], 0
            part.append(it); size += n
        if part:
            parts.append(part)
        for k, part in enumerate(parts, 1):
            name = '%s-%s%s.md' % (c, act, ('-%d' % k) if len(parts) > 1 else '')
            first = part[0][0].split('　')[0][4:]; last = part[-1][0].split('　')[0][4:]
            write(d + '/items/' + name, ['# %s：%s（%s–%s）' % (title.split('：')[0], label, first, last), '',
                                         '[回這一類的目錄](%s.md)。' % c, ''] + [l for it in part for l in it])
            index.append('- [%s（%s–%s）](%s)' % (label, first, last, name))
    write(d + '/items/%s.md' % c, index)

def write(path, lines):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines).rstrip('\n') + '\n')

GD = OUT + '/2026-11-12-rule-suggestions'
bycat = collections.OrderedDict((c, [r for r in gitems if r['category'] == c]) for c in GCAT)
for c, rs in bycat.items():
    if not rs:
        continue
    sections = []
    for a in GACT:
        sub = [r for r in rs if r['action'] == a]
        if not sub:
            continue
        label = '%s：%d 組（%s–%s）' % (GACT[a], len(sub), sub[0]['_seq'], sub[-1]['_seq'])
        sections.append((a, GACT[a], ['## ' + label, ''] + [l for r in sub for l in g_item(r)]))
    write_cat(GD, c, '%s %s：%d 組（%s–%s）' % (c, GCAT[c], len(rs), rs[0]['_seq'], rs[-1]['_seq']), sections,
              '[回報告](../README.md#%s)。每組的格式見報告「怎麼讀」。' % c.lower())

gdata = []
for r in gitems:
    g = groups[r['id']]; fk = full_keys[r['id']]
    gdata.append({'seq': r['_seq'], 'id': r['id'], 'family': g['family'], 'field': g['field'], 'kr': g['kr'],
                  'units': g['units'], 'category': r['category'], 'action': r['action'], 'confidence': r['confidence'],
                  'reason': r['reason'], 'target': r.get('target'),
                  'changes': [{'from': c['from'], 'to': c['to'], 'keys': fk[c['from']]} for c in r.get('changes') or []],
                  'glossary': r.get('glossary'), 'options': r.get('options'),
                  'other_refs': {c['from']: other_refs(r, c) for c in r.get('changes') or [] if other_refs(r, c)} or None,
                  'variants': [{'zh': v['zh'], 'count': v['count'], 'origins': v['origins']} for v in g['variants']]})
write(GD + '/data/judgements.json', [json.dumps(gdata, ensure_ascii=False, indent=1)])

# tables for README
T = []
T.append('| 類別 | ' + ' | '.join(GACT.values()) + ' | 合計 | 單位 | 會改的單位 |')
T.append('|---|' + '---:|' * (len(GACT) + 3))
tot = collections.Counter()
for c, rs in bycat.items():
    cnt = collections.Counter(r['action'] for r in rs)
    units = sum(groups[r['id']]['units'] for r in rs)
    ch = sum(g_units_changed(r) for r in rs)
    tot.update(cnt); tot['_u'] += units; tot['_c'] += ch
    T.append('| [%s %s](items/%s.md) | ' % (c, GCAT[c], c) + ' | '.join(str(cnt.get(a, '')) for a in GACT) +
             ' | %d | %d | %d |' % (len(rs), units, ch))
T.append('| 合計 | ' + ' | '.join(str(tot.get(a, 0)) for a in GACT) + ' | %d | %d | %d |' % (len(gitems), tot['_u'], tot['_c']))
T.append('')
fams = collections.OrderedDict()
for r in gitems:
    fams.setdefault(groups[r['id']]['family'], collections.Counter())[r['action']] += 1
T.append('| 族群 | ' + ' | '.join(GACT.values()) + ' | 合計 |')
T.append('|---|' + '---:|' * (len(GACT) + 1))
for f, cnt in sorted(fams.items(), key=lambda t: -sum(t[1].values())):
    T.append('| %s | ' % f + ' | '.join(str(cnt.get(a, '')) for a in GACT) + ' | %d |' % sum(cnt.values()))
T.append('')
# per category x family x action
for c, rs in bycat.items():
    cf = collections.OrderedDict()
    for r in rs:
        cf.setdefault(groups[r['id']]['family'], collections.Counter())[r['action']] += 1
    T.append('%s: ' % c + '；'.join('%s %s' % (f, '、'.join('%s %d' % (GACT[a], n) for a, n in cnt.most_common()))
                                    for f, cnt in sorted(cf.items(), key=lambda t: -sum(t[1].values()))))
T.append('')
T.append('new targets: ' + '、'.join(r['_seq'] for r in gitems if r['action'] == 'unify' and r['target'] not in {v['zh'] for v in groups[r['id']]['variants']}))
T.append('low: ' + '、'.join(r['_seq'] for r in gitems if r['confidence'] == 'low'))
T.append('decide: ' + '、'.join(r['_seq'] for r in gitems if r['action'] == 'decide'))
T.append('glossary: ' + '、'.join(r['_seq'] for r in gitems if r['action'] == 'glossary'))
T.append('keyword refs: ' + '、'.join(r['_seq'] for r in gitems if any(other_refs(r, c) for c in r.get('changes') or [])))
T.append('corrections: ' + '、'.join(gj[c[0]]['_seq'] for c in CORRECTIONS))
write(RV + '/tables-groups.md', T)

# ---------- translator notes ----------
NORDER = list(NACT)
nitems = sorted(nj.values(), key=lambda r: (list(NCAT).index(r['category']), NORDER.index(r['action']),
                                            notes[r['id']]['family'], notes[r['id']]['key'], r['id']))
for i, r in enumerate(nitems, 1):
    r['_seq'] = 'N%03d' % i
kw = {x['Key']: x for x in kwspace}

def ws_only(c):
    """The change is what the keyword-space rule (NormalizeKeywordSpace) makes of the unit."""
    k = kw.get(c['key'])
    return k is not None and k['From'] == c['from'] and k['To'] == c['to']

def n_item(r):
    n = notes[r['id']]
    flags = []
    if r['confidence'] == 'low':
        flags.append('⚠ 低信心')
    if r['action'] == 'fix' and all(ws_only(c) for c in r['changes']):
        flags.append('只補狀態名後的空格（見報告「狀態名後的空格」）')
    out = ['### %s　%s｜%s %s｜信心%s%s' % (r['_seq'], NACT[r['action']], r['category'], NCAT[r['category']],
                                         CONF[r['confidence']], ('　' + '　'.join(flags)) if flags else ''), '']
    out.append('`%s`　%s，批次 %s，來源 %s%s' % (r['id'], n['family'], n['batch'], n['origin'],
                                          ('，說話者 ' + n['speaker']) if n['speaker'] else ''))
    out.append('')
    out.append('- KR：%s' % code(n['kr']))
    out.append('- 現行譯文：%s' % code(n['zh']))
    out.append('- 註記：%s' % esc(n['note']).replace('\n', ' ⏎ '))
    out.append('- 判讀：%s' % esc(r['summary']))
    if r.get('reason'):
        out.append('- 理由：%s' % esc(r['reason']))
    if r.get('checked'):
        out.append('- 查證：%s' % esc(r['checked']))
    act = r['action']
    accept = '`bin/lct accept` 把這則註記記進 `rules/declined.yaml`（`reason` 為判讀），審核頁不再列出'
    if act == 'none':
        out.append('- 同意後寫入：' + accept + '；不改譯文')
    elif act == 'source':
        out.append('- 同意後寫入：' + accept + '；不改譯文（原文問題只記在這份報告）')
    elif act == 'glossary':
        gl = r['glossary']
        out.append('- 建議：在 `glossary/terms.yaml` 加詞條 KR %s → %s（%s）' % (code(gl.get('kr')), code(gl.get('zh')), esc(gl.get('note') or '')))
        out.append('- 同意後寫入：`glossary/terms.yaml` 一個條目（`status: approved`），並' + accept)
    elif act == 'decide':
        out.append('- 建議：交給你決定，選項：')
        out.extend(render_options(r.get('options') or []))
        out.append('- 同意後寫入：依你選的選項，並' + accept)
    elif act == 'fix':
        out.append('- 會改（`from` 已與 tw 5f25d25 的現行譯文逐字核對）：')
        for c in r['changes']:
            out.append('  - `%s`：%s → %s' % (c['key'], code(c['from']), code(c['to'])))
            out.append(diff_line(c['from'], c['to'], '    - '))
        out.append('- 同意後寫入：每個要改的單位一個 `rules/rewrite/` 改寫檔（逐字比對整句），並' + accept)
    out.append('')
    return out

ND = OUT + '/2026-11-12-translator-notes'
nbycat = collections.OrderedDict((c, [r for r in nitems if r['category'] == c]) for c in NCAT)
for c, rs in nbycat.items():
    if not rs:
        continue
    sections = []
    for a in NACT:
        sub = [r for r in rs if r['action'] == a]
        if not sub:
            continue
        label = '%s：%d 則（%s–%s）' % (NACT[a], len(sub), sub[0]['_seq'], sub[-1]['_seq'])
        sections.append((a, NACT[a], ['## ' + label, ''] + [l for r in sub for l in n_item(r)]))
    write_cat(ND, c, '%s %s：%d 則（%s–%s）' % (c, NCAT[c], len(rs), rs[0]['_seq'], rs[-1]['_seq']), sections,
              '[回報告](../README.md#%s)。每則的格式見報告「怎麼讀」。' % c.lower())

ndata = []
for r in nitems:
    n = notes[r['id']]
    ndata.append({'seq': r['_seq'], 'id': r['id'], 'key': n['key'], 'family': n['family'], 'batch': n['batch'],
                  'origin': n['origin'], 'kr': n['kr'], 'zh': n['zh'], 'note': n['note'],
                  'category': r['category'], 'action': r['action'], 'confidence': r['confidence'],
                  'summary': r['summary'], 'reason': r.get('reason'), 'checked': r.get('checked'),
                  'changes': r.get('changes'), 'glossary': r.get('glossary'), 'options': r.get('options')})
write(ND + '/data/judgements.json', [json.dumps(ndata, ensure_ascii=False, indent=1)])

# keyword space: every unit NormalizeKeywordSpace would change
in_notes = {c['key']: r['_seq'] for r in nitems if r['action'] == 'fix' for c in r['changes'] if ws_only(c)}
def spots(frm, to):
    """The places a space is inserted, each with some text around it."""
    out, i, j = [], 0, 0
    while j < len(to):
        if i < len(frm) and frm[i] == to[j]:
            i += 1; j += 1
            continue
        assert to[j] == ' ', (frm, to)
        out.append((frm[max(0, i - 10):i], frm[i:i + 6]))
        j += 1
    return out
kwrows = sorted(kwspace, key=lambda x: (x['Origin'] != 'claude', x['Key']))
for x in kwrows:
    x['Spots'] = spots(x['From'], x['To'])
write(ND + '/data/keyword-space.json', [json.dumps([{'key': x['Key'], 'origin': x['Origin'], 'batch': x['Batch'],
                                                       'from': x['From'], 'to': x['To'], 'in_note': in_notes.get(x['Key'])}
                                                      for x in kwrows], ensure_ascii=False, indent=1)])
for origin, title in (('claude', '本專案新譯（2026-11-01 的批次）'), ('paratranz', '社群舊譯（Paratranz）')):
    rows = [x for x in kwrows if x['Origin'] == origin]
    lines = ['# 狀態名後的空格：%s，%d 個單位' % (title, len(rows)), '',
             '[回報告](../README.md#狀態名後的空格)。每列是一個單位；「補空格處」列出每個要補半形空格的位置〔␣〕與前後文字（原有的空格也顯示為 ␣），其餘文字不變。完整的改前、改後全文在 [`data/keyword-space.json`](../data/keyword-space.json)。', '',
             '| # | 單位 | 補空格處 | 註記 |', '|---:|---|---|---|']
    for i, x in enumerate(rows, 1):
        sp = '<br>'.join(code(a.replace(' ', '␣') + '〔␣〕' + b.replace(' ', '␣'), table=True) for a, b in x['Spots'])
        lines.append('| %d | %s | %s | %s |' % (i, code(x['Key'], table=True), sp, in_notes.get(x['Key'], '')))
    write(ND + '/items/keyword-space-%s.md' % origin, lines)

T = []
T.append('| 類別 | ' + ' | '.join(NACT.values()) + ' | 合計 | 會改的單位 |')
T.append('|---|' + '---:|' * (len(NACT) + 2))
tot = collections.Counter()
for c, rs in nbycat.items():
    cnt = collections.Counter(r['action'] for r in rs)
    ch = sum(len(r.get('changes') or []) for r in rs if r['action'] == 'fix')
    tot.update(cnt); tot['_c'] += ch
    T.append('| [%s %s](items/%s.md) | ' % (c, NCAT[c], c) + ' | '.join(str(cnt.get(a, '')) for a in NACT) + ' | %d | %d |' % (len(rs), ch))
T.append('| 合計 | ' + ' | '.join(str(tot.get(a, 0)) for a in NACT) + ' | %d | %d |' % (len(nitems), tot['_c']))
T.append('')
fams = collections.OrderedDict()
for r in nitems:
    fams.setdefault(notes[r['id']]['family'], collections.Counter())[r['category']] += 1
T.append('| 族群 | ' + ' | '.join(NCAT) + ' | 合計 |')
T.append('|---|' + '---:|' * (len(NCAT) + 1))
for f, cnt in sorted(fams.items(), key=lambda t: -sum(t[1].values())):
    T.append('| %s | ' % f + ' | '.join(str(cnt.get(c, '')) for c in NCAT) + ' | %d |' % sum(cnt.values()))
T.append('')
for c, rs in nbycat.items():
    T.append('%s: ' % c + '；'.join('%s %d' % (NACT[a], n) for a, n in collections.Counter(r['action'] for r in rs).most_common()))
T.append('ws fixes: ' + '、'.join(r['_seq'] for r in nitems if r['action'] == 'fix' and all(ws_only(c) for c in r['changes'])))
T.append('content fixes: ' + '、'.join(r['_seq'] for r in nitems if r['action'] == 'fix' and not all(ws_only(c) for c in r['changes'])))
T.append('low: ' + '、'.join(r['_seq'] for r in nitems if r['confidence'] == 'low'))
T.append('decide: ' + '、'.join(r['_seq'] for r in nitems if r['action'] == 'decide'))
T.append('glossary: ' + '、'.join(r['_seq'] for r in nitems if r['action'] == 'glossary'))
T.append('source: ' + '、'.join(r['_seq'] for r in nitems if r['action'] == 'source'))
T.append('kw claude %d paratranz %d in_notes %d' % (sum(x['Origin'] == 'claude' for x in kwrows), sum(x['Origin'] == 'paratranz' for x in kwrows), len(in_notes)))
write(RV + '/tables-notes.md', T)
print('groups', len(gitems), 'notes', len(nitems), 'corrections', len(CORRECTIONS))
