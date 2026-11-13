"""Assemble the StoryData accuracy report items and data.

usage: python3 -I acc_assemble.py <sd dir> <docs out dir>

Reads <sd>/units.json, <sd>/vb/*-all.json (items with first-pass findings and
context) and <sd>/vb/out/*.json (verdicts). Writes, under the docs dir:
items/<group>.md, rejected.md, data/findings.json, data/rejected.json, and
prints the summary tables to <sd>/tables-acc.md.
"""
import json, glob, os, re, sys, collections, difflib

SD, OUT = sys.argv[1], sys.argv[2]
units = json.load(open(SD + '/units.json'))
by_key = {u['key']: u for u in units}

items = collections.OrderedDict()
for f in sorted(glob.glob(SD + '/vb/*-all.json')):
    for it in json.load(open(f)):
        if it['ref'] in items:  # same ref in two waves: merge findings
            items[it['ref']]['findings'] += it['findings']
        else:
            items[it['ref']] = it
verdicts = {}
for f in sorted(glob.glob(SD + '/vb/out/*.json')):
    for v in json.load(open(f)):
        verdicts[v['ref']] = v

CAT = collections.OrderedDict([('A1', '錯譯'), ('A2', '漏譯'), ('A3', '多譯'), ('A4', '指涉錯'), ('A5', '反轉'),
                               ('A6', '數字、名稱、術語'), ('A7', '語氣、稱謂'), ('A8', '說話者名、稱號'), ('A9', '其他')])
SEV = collections.OrderedDict([('high', '高'), ('medium', '中'), ('low', '低')])

def group(file):
    b = file.split('/')[-1][:-5]
    m = re.match(r'S(\d{1,2})(\d\d)', b)
    if b.startswith('S') and m:
        return 'main-%02d' % int(m.group(1))
    m = re.match(r'(\d+)D', b)
    if m:
        return 'dungeon-%02d' % int(m.group(1))
    m = re.match(r'P1(\d\d)', b)
    if m:
        return 'identity-%s' % m.group(1)
    if b.startswith('ES'):
        return 'event-es'
    if b.startswith('PC'):
        return 'event-pc'
    m = re.match(r'E(\d)', b)
    if m:
        return 'event-e%s' % m.group(1)
    return 'other'

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
def group_order(g):
    k, _, n = g.partition('-')
    return ({'main': 0, 'dungeon': 1, 'event': 2, 'identity': 3}.get(k, 9), n)

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
    s = str(s)
    tick = '``' if '`' in s else '`'
    return tick + s + tick

def esc(s):
    return '' if s is None else str(s).replace('\n', ' ')

kept, rejected, unverified, noop = [], [], [], []
for ref, it in items.items():
    v = verdicts.get(ref)
    u = by_key[it['key']]
    self_ctx = next((c for c in it['context'] if c['self']), None)
    rec = {'ref': ref, 'key': it['key'], 'file': u['file'], 'field': it['field'], 'origin': it['origin'],
           'speaker': self_ctx['speaker'] if self_ctx else '', 'kr': it['kr'], 'jp': self_ctx['jp'] if self_ctx else '',
           'en': self_ctx['en'] if self_ctx else '', 'tw': it['tw'], 'first_pass': it['findings']}
    if u['f'] != 'content':
        rec['jp'], rec['en'] = u['jp'].replace('\n', '⏎'), u['en'].replace('\n', '⏎')
    if not v:
        unverified.append(rec)
        continue
    rec.update({'verdict': v['verdict'], 'category': v.get('category') or it['findings'][0].get('category'),
                'severity': v.get('severity') or it['findings'][0].get('severity'), 'explain': v.get('explain'),
                'suggest': v.get('suggest'), 'reason': v.get('reason'), 'confidence': v.get('confidence')})
    if v['verdict'] == 'rejected':
        rejected.append(rec)
    elif not rec['suggest'] or rec['suggest'] == rec['tw']:
        noop.append(rec)
    else:
        kept.append(rec)

kept.sort(key=lambda r: (group_order(group(r['file'])), r['file'], int(re.search(r'#(\d+)', r['ref']).group(1)), r['ref']))
for i, r in enumerate(kept, 1):
    r['seq'] = 'S%04d' % i
    r['group'] = group(r['file'])

def item_md(r):
    out = ['### %s　`%s`　%s｜%s　（信心%s）' % (r['seq'], r['ref'], CAT.get(r['category'], r['category']),
                                          SEV.get(r['severity'], r['severity']), SEV.get(r['confidence'], r['confidence'] or '—')), '']
    out.append('- 說話者：%s；來源：%s；單位：`%s`' % (r['speaker'], r['origin'], r['key']))
    out.append('- KR：%s' % code(r['kr']))
    out.append('- JP：%s' % code(r['jp']))
    out.append('- EN：%s' % code(r['en']))
    out.append('- 現行：%s' % code(r['tw']))
    out.append('- **建議**：%s' % code(r['suggest']))
    sm = difflib.SequenceMatcher(None, r['tw'], r['suggest'], autojunk=False)
    if sm.ratio() < 0.6:
        out.append('- 差異：整句改寫較多，請看建議全文')
    else:
        out.append('- 差異：%s' % '；'.join(code(p) for p in diff(r['tw'], r['suggest'])))
    out.append('- 說明：%s' % esc(r['explain']))
    if r.get('reason'):
        out.append('- 複核附註：%s' % esc(r['reason']))
    out.append('')
    return out

os.makedirs(OUT + '/items', exist_ok=True)
os.makedirs(OUT + '/data', exist_ok=True)
groups = collections.OrderedDict()
for r in kept:
    groups.setdefault(r['group'], []).append(r)
for g, rs in groups.items():
    lines = ['# %s：%d 處（%s–%s）' % (group_title(g), len(rs), rs[0]['seq'], rs[-1]['seq']), '',
             '[回報告](../README.md)。格式見報告「怎麼讀」。', '']
    cur = None
    for r in rs:
        f = r['file'].split('/')[-1][:-5]
        if f != cur:
            lines += ['## %s' % f, '']
            cur = f
        lines += item_md(r)
    open(OUT + '/items/%s.md' % g, 'w', encoding='utf-8').write('\n'.join(lines).rstrip() + '\n')

rej = ['# 複核後不採用的 %d 處' % len(rejected), '',
       '[回報告](README.md#資料與方法)。初篩認為有錯、複核判斷不算錯的單位，列出來供參考；這些不會改。', '',
       '| 單位 | KR | 現行 TW | 初篩說法 | 複核理由 |', '|---|---|---|---|---|']
for r in sorted(rejected, key=lambda r: r['ref']):
    fp = r['first_pass'][0]
    t = lambda s: esc(s).replace('|', '\\|')
    rej.append('| `%s` | %s | %s | %s | %s |' % (r['ref'], t(r['kr'])[:80], t(r['tw'])[:80], t(fp.get('explain'))[:120], t(r.get('reason'))[:160]))
open(OUT + '/rejected.md', 'w', encoding='utf-8').write('\n'.join(rej) + '\n')
json.dump(kept, open(OUT + '/data/findings.json', 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
json.dump(rejected, open(OUT + '/data/rejected.json', 'w', encoding='utf-8'), ensure_ascii=False, indent=1)

# tables
content = [u for u in units if u['f'] == 'content']
T = []
T.append('items %d verdicts %d kept %d rejected %d noop %d unverified %d' % (len(items), len(verdicts), len(kept), len(rejected), len(noop), len(unverified)))
T.append('verdicts: %s' % dict(collections.Counter(v['verdict'] for v in verdicts.values())))
T.append('')
T.append('| 類別 | 高 | 中 | 低 | 合計 |')
T.append('|---|---:|---:|---:|---:|')
cs = collections.Counter((r['category'], r['severity']) for r in kept)
for c, name in CAT.items():
    row = [cs.get((c, s), 0) for s in SEV]
    if sum(row):
        T.append('| %s %s | %s | %d |' % (c, name, ' | '.join(str(x) for x in row), sum(row)))
T.append('| 合計 | %s | %d |' % (' | '.join(str(sum(cs.get((c, s), 0) for c in CAT)) for s in SEV), len(kept)))
T.append('')
oc = collections.Counter(u['o'] for u in content)
ok = collections.Counter(r['origin'] for r in kept if r['field'] == 'content')
T.append('| 來源 | 台詞行數 | 確認的錯誤 | 每千行 |')
T.append('|---|---:|---:|---:|')
for o in ('paratranz', 'claude', 'human', 'copy'):
    if oc.get(o):
        T.append('| %s | %d | %d | %.1f |' % (o, oc[o], ok.get(o, 0), 1000 * ok.get(o, 0) / oc[o]))
T.append('')
gl = collections.Counter(group(u['file']) for u in content)
T.append('| 範圍 | 台詞行數 | 錯誤 | 高 | 每千行 | 清單 |')
T.append('|---|---:|---:|---:|---:|---|')
for g in sorted(gl, key=group_order):
    rs = groups.get(g, [])
    T.append('| %s | %d | %d | %d | %.1f | %s |' % (group_title(g), gl[g], len(rs), sum(1 for r in rs if r['severity'] == 'high'),
                                                 1000 * len(rs) / gl[g], ('[items/%s.md](items/%s.md)' % (g, g)) if rs else '—'))
T.append('')
T.append('high: ' + '、'.join('%s %s' % (r['seq'], r['ref']) for r in kept if r['severity'] == 'high'))
T.append('noop: ' + '、'.join(r['ref'] for r in noop))
T.append('unverified: ' + '、'.join(r['ref'] for r in unverified))
open(SD + '/tables-acc.md', 'w', encoding='utf-8').write('\n'.join(T) + '\n')
print('\n'.join(T[:3]))
