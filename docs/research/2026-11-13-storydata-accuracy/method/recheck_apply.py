"""Merge the context re-check results into data/findings.json and render items/*.md again.

usage: python3 -I recheck_apply.py <docs dir> <date> <raw tag> [--adjustments <rules/adjustments dir>] [recheck json ...]

Each recheck file is a JSON array of
  {"seq", "result": "direct"|"adjust", "suggest", "kr_meaning", "context", "problem",
   "also": [{"ref", "kr", "tw", "suggest", "why"}], "note"}
(see method/RECHECK.md). A record replaces the `recheck` field of the finding with the same seq.
With no recheck file the script only renders items/*.md from data/findings.json.

With --adjustments, a finding whose unit already has a manual adjustment is checked against that
adjustment instead of the suggestion of the report: "direct" then means that the adjustment needs
no further change, and the adjustment is recorded in recheck.applied.

The item layout is the one of acc_assemble.py; a finding with a `recheck` field gets the
「再檢視」 lines after it. Groups without any re-checked finding are rendered byte for byte as before.
"""
import json, os, re, sys, collections, difflib

OUT, DATE, RAW = sys.argv[1], sys.argv[2], sys.argv[3]
FILES = sys.argv[4:]
ADJ = {}
if FILES[:1] == ['--adjustments']:
    import glob, yaml
    for _p in sorted(glob.glob(FILES[1] + '/*.yaml')):
        for _e in yaml.safe_load(open(_p, encoding='utf-8')) or []:
            ADJ[_e['key']] = _e['zh']
    FILES = FILES[2:]
N = lambda t: (t or '').replace('\n', '⏎')   # the report writes line breaks as ⏎
P = OUT + '/data/findings.json'
kept = json.load(open(P, encoding='utf-8'))
by_seq = {r['seq']: r for r in kept}

for f in FILES:
    for x in json.load(open(f, encoding='utf-8')):
        r = by_seq[x['seq']]
        if x['result'] not in ('direct', 'adjust'):
            sys.exit('%s: result %r' % (x['seq'], x['result']))
        applied = ADJ.get(r['key'])
        base = N(applied) if applied is not None else r['suggest']
        if x['result'] == 'direct' and N(x['suggest']) != base:
            sys.exit('%s: direct, but the text is not the %s' % (x['seq'], 'adjustment' if applied is not None else 'suggestion of the report'))
        if x['result'] == 'adjust' and not (x.get('kr_meaning') and x.get('context') and x.get('problem')):
            sys.exit('%s: adjust needs kr_meaning, context and problem' % x['seq'])
        if x['result'] == 'adjust' and N(x['suggest']) == base:
            sys.exit('%s: adjust, but the text is the %s' % (x['seq'], 'adjustment' if applied is not None else 'suggestion of the report'))
        rc = collections.OrderedDict([('date', DATE), ('raw', RAW), ('result', x['result'])])
        if applied is not None:
            rc['applied'] = collections.OrderedDict([('zh', N(applied)), ('same_as_report', N(applied) == r['suggest'])])
        if x['result'] == 'adjust':
            rc['suggest'] = x['suggest']
            rc['kr_meaning'], rc['context'], rc['problem'] = x['kr_meaning'], x['context'], x['problem']
        rc['also'] = [collections.OrderedDict((k, a[k]) for k in ('ref', 'kr', 'tw', 'suggest', 'why')) for a in x.get('also') or []]
        if x.get('note'):
            rc['note'] = x['note']
        r['recheck'] = rc
if FILES:
    json.dump(kept, open(P, 'w', encoding='utf-8'), ensure_ascii=False, indent=1)

CAT = collections.OrderedDict([('A1', '錯譯'), ('A2', '漏譯'), ('A3', '多譯'), ('A4', '指涉錯'), ('A5', '反轉'),
                               ('A6', '數字、名稱、術語'), ('A7', '語氣、稱謂'), ('A8', '說話者名、稱號'), ('A9', '其他')])
SEV = collections.OrderedDict([('high', '高'), ('medium', '中'), ('low', '低')])
SINNER = {'01': '李箱', '02': '浮士德', '03': '唐吉訶德', '04': '良秀', '05': '莫梭', '06': '鴻璐', '07': '希斯克利夫',
          '08': '以實瑪利', '09': '羅佳', '10': '辛克萊', '11': '奧提斯', '12': '格里高爾'}

def group_title(g):
    k, _, n = g.partition('-')
    if k == 'main':
        return '主線第 %d 章（S%s…）' % (int(n), int(n)) if n not in ('00', '99') else ('主線序章（S0…）' if n == '00' else '主線其他（S99…）')
    if k == 'dungeon':
        return '第 %d 章副本與章節活動（%dD…）' % (int(n), int(n))
    if k == 'identity':
        return '人格故事：%s（P1%s…）' % (SINNER.get(n, n), n)
    if g == 'event-es':
        return '活動 ES…'
    if g == 'event-pc':
        return '合作活動 PC…（明日方舟）'
    if k == 'event':
        return '活動與間章 E%s…' % n[1:]
    return '其他'

def vis(t):
    return t.replace(' ', '␣')

def diff(frm, to, ctx=8):
    sm = difflib.SequenceMatcher(None, frm, to, autojunk=False)
    parts = []
    for op, a1, a2, b1, b2 in sm.get_opcodes():
        if op == 'equal':
            continue
        parts.append('%s%s〔%s→%s〕%s%s' % ('…' if a1 > ctx else '', frm[max(0, a1 - ctx):a1], vis(frm[a1:a2]) or '∅',
                                            vis(to[b1:b2]) or '∅', frm[a2:a2 + ctx], '…' if a2 + ctx < len(frm) else ''))
    return parts

def code(s):
    if s is None or s == '':
        return '（空）'
    s = str(s).replace('\n', '⏎')
    tick = '``' if '`' in s else '`'
    return tick + s + tick

def esc(s):
    return '' if s is None else str(s).replace('\n', ' ')

def diff_line(frm, to):
    sm = difflib.SequenceMatcher(None, frm, to, autojunk=False)
    if sm.ratio() < 0.6:
        return '整句改寫較多，請看建議全文'
    return '；'.join(code(p) for p in diff(frm, to))

def recheck_md(r):
    rc = r['recheck']
    out = []
    ap = rc.get('applied')
    if ap:
        state = '已寫成人工調整（與報告建議相同）' if ap['same_as_report'] else '已寫成人工調整，譯法與報告建議不同：%s' % code(ap['zh'])
    if rc['result'] == 'direct':
        if ap:
            out.append('- **再檢視**（%s）：%s；不需要再改%s' % (rc['date'], state, ('。' + esc(rc['note'])) if rc.get('note') else ''))
        else:
            out.append('- **再檢視**（%s）：可以直接照建議修改%s' % (rc['date'], ('。' + esc(rc['note'])) if rc.get('note') else ''))
    else:
        out.append('- **再檢視**（%s）：%s建議再調整' % (rc['date'], (state + '；') if ap else ''))
        out.append('  - KR 原意：%s' % esc(rc['kr_meaning']))
        out.append('  - 前後文：%s' % esc(rc['context']))
        out.append('  - %s：%s' % ('現在的譯文的問題' if ap else '報告建議的問題', esc(rc['problem'])))
        out.append('  - **調整後的建議**：%s' % code(rc['suggest']))
        out.append('  - 差異（對現行）：%s' % diff_line(ap['zh'] if ap else r['tw'], rc['suggest']))
        if rc.get('note'):
            out.append('  - 備註：%s' % esc(rc['note']))
    for a in rc['also']:
        out.append('- **前後文一起改**：`%s`' % a['ref'])
        out.append('  - KR：%s' % code(a['kr']))
        out.append('  - 現行：%s' % code(a['tw']))
        out.append('  - **建議**：%s' % code(a['suggest']))
        out.append('  - 說明：%s' % esc(a['why']))
    return out

def item_md(r):
    out = ['### %s　`%s`　%s｜%s　（信心%s）' % (r['seq'], r['ref'], CAT.get(r['category'], r['category']),
                                          SEV.get(r['severity'], r['severity']), SEV.get(r['confidence'], r['confidence'] or '—')), '']
    out.append('- 說話者：%s；來源：%s；單位：`%s`' % (r['speaker'], r['origin'], r['key']))
    out.append('- KR：%s' % code(r['kr']))
    out.append('- JP：%s' % code(r['jp']))
    out.append('- EN：%s' % code(r['en']))
    out.append('- 現行：%s' % code(r['tw']))
    out.append('- **建議**：%s' % code(r['suggest']))
    out.append('- 差異：%s' % diff_line(r['tw'], r['suggest']))
    out.append('- 說明：%s' % esc(r['explain']))
    if r.get('reason'):
        out.append('- 複核附註：%s' % esc(r['reason']))
    if r.get('recheck'):
        out += recheck_md(r)
    out.append('')
    return out

groups = collections.OrderedDict()
for r in kept:
    groups.setdefault(r['group'], []).append(r)
summary = []
for g, rs in groups.items():
    lines = ['# %s：%d 處（%s–%s）' % (group_title(g), len(rs), rs[0]['seq'], rs[-1]['seq']), '',
             '[回報告](../README.md)。格式見報告「怎麼讀」。', '']
    rc = [r['recheck'] for r in rs if r.get('recheck')]
    if rc:
        nd = sum(1 for x in rc if x['result'] == 'direct')
        na = sum(1 for x in rc if x['result'] == 'adjust')
        nl = sum(len(x['also']) for x in rc)
        nap = sum(1 for x in rc if x.get('applied'))
        dates = sorted(set(x['date'] for x in rc))
        adj_list = ('建議再調整的是 %s。' % '、'.join(r['seq'] for r in rs if r.get('recheck') and r['recheck']['result'] == 'adjust')) if na else ''
        if nap:
            nda = sum(1 for x in rc if x['result'] == 'direct' and x.get('applied'))
            ndu = nd - nda
            lines += ['再檢視（%s，見報告「[再檢視](../README.md#再檢視)」）：%d 項中 %d 項已寫成人工調整。%d 項不需要再改%s、%d 項建議再調整；另有 %d 句前後文建議一起改。%s' % (
                '、'.join(dates), len(rc), nap, nda, ('、%d 項還沒寫入但可以直接照建議修改' % ndu) if ndu else '', na, nl, adj_list), '']
        else:
            lines += ['再檢視（%s，見報告「[再檢視](../README.md#再檢視)」）：%d 項中 %d 項可以直接照建議修改、%d 項建議再調整；另有 %d 句前後文建議一起改。%s' % (
                '、'.join(dates), len(rc), nd, na, nl, adj_list), '']
        summary.append((g, len(rs), len(rc), nd, na, nl, nap))
    cur = None
    for r in rs:
        f = r['file'].split('/')[-1][:-5]
        if f != cur:
            lines += ['## %s' % f, '']
            cur = f
        lines += item_md(r)
    open(OUT + '/items/%s.md' % g, 'w', encoding='utf-8').write('\n'.join(lines).rstrip() + '\n')

for title, rows, head in (('還沒處理的範圍', [x for x in summary if not x[6]], '可以直接照建議'),
                          ('已經處理過的範圍', [x for x in summary if x[6]], '不需要再改')):
    if not rows:
        continue
    print('%s：' % title)
    print()
    print('| 範圍 | 項數 | %s | 建議再調整 | 前後文一起改 | 清單 |' % head)
    print('|---|---:|---:|---:|---:|---|')
    for g, n, nr, nd, na, nl, nap in rows:
        print('| %s | %d | %d | %d | %d | [items/%s.md](items/%s.md) |' % (group_title(g), nr, nd, na, nl, g, g))
    print('| 合計 | %d | %d | %d | %d | |' % (sum(x[2] for x in rows), sum(x[3] for x in rows), sum(x[4] for x in rows), sum(x[5] for x in rows)))
    print()
