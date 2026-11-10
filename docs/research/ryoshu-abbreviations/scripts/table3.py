# 報告第 3 節的表（第 3 輪）：依類別分 4 張表；KR 相同而 JP／EN／繁中不同的各占一行，附四種語言的完整說法與來源
# 輸入：rows4.json（rows4.py）、full_forms.json（完整說法，人工判讀）；輸出 table3.md
import json, collections, re
rows = json.load(open('rows4.json'))
full = {o['id']: o for o in json.load(open('full_forms.json'))}

def cell(x): return str(x).replace('|', '｜').replace('\n', ' ')
def fullcell(o, lang, r):
    x = (o or {}).get(lang) or {}
    s, f = x.get('source', '不明'), re.sub(r'<[^>]+>', '', x.get('full', ''))
    if s == '遊戲內':
        here = x['loc'].split('#')[0] in {l.split('#')[0] for l in r['locs']}
        out = f'{f}（遊戲內{"" if here else "・他處"}：{x["who"]}，`{x["loc"]}`）'
    elif s == '推測': out = f'{f}（推測）'
    elif s == '未縮': out = f'{f}（未縮）' if f else '（未縮）'
    else: out = f'{f}（不明）' if f else '不明'
    return out
def forms(main, all_):
    rest = [f for f in all_ if f != main]
    return main + ('（另 ' + '、'.join(rest) + '）' if rest else '')
def krkey(a): return re.sub(r'\s', '', a).rstrip('.')
def krcell(r, rs):
    seen = {krkey(r['kr'])}; alts = []
    for a in r['kr_forms']:
        k = krkey(a)
        if k not in seen: seen.add(k); alts.append(a)
    s = forms(r['kr'], [r['kr']] + alts)
    extra = []
    if r['table'] == '中文不一' and r['class'] != '中文不一': extra.append('另屬' + r['class'].replace('＋中文不一', ''))
    if r['note']: extra.append(r['note'])
    if r['proposal']: extra.append('提案 ' + r['proposal'])
    return s + ('（' + '；'.join(extra) + '）' if extra else '')
def where(r, o):
    n = len(r['locs'])
    s = (f'{n} 處：' if n > 1 else '') + '、'.join(f'`{l}`' for l in r['locs'])
    rm = ((o or {}).get('remark') or '').strip()
    return s + (f'（附註：{rm}）' if rm else '')

HEAD = ['| KR 縮略 | KR 完整 | JP 縮略 | JP 完整 | EN 縮略 | EN 完整 | TW 縮略 | TW 完整 | 位置 |',
        '|---|---|---|---|---|---|---|---|---|']
byk = collections.OrderedDict()
for r in rows: byk.setdefault(r['g'], []).append(r)
out, stats = [], {}
for t in ('一致', '同義異寫', '異義', '中文不一'):
    ks = [(g, rs) for g, rs in byk.items() if rs[0]['table'] == t]
    ks.sort(key=lambda x: (-x[1][0]['n_kind'], x[0]))
    L = list(HEAD)
    for g, rs in ks:
        for r in sorted(rs, key=lambda r: -len(r['locs'])):
            o = full.get(r['id'])
            L.append('| ' + ' | '.join(cell(x) for x in (
                krcell(r, rs), fullcell(o, 'kr', r), forms(r['jp'], r['jp_forms']), fullcell(o, 'jp', r),
                forms(r['en'], r['en_forms']), fullcell(o, 'en', r), r['tw'], fullcell(o, 'tw', r), where(r, o))) + ' |')
    stats[t] = (len(ks), len(L) - 2)
    out.append((t, L))
with open('table3.md', 'w') as f:
    for t, L in out:
        f.write(f'### {t}（{stats[t][0]} 種、{stats[t][1]} 行）\n\n' + '\n'.join(L) + '\n\n')
src = collections.Counter(((full.get(r['id']) or {}).get(l) or {}).get('source', '缺') for r in rows for l in ('kr', 'jp', 'en', 'tw'))
by = {l: collections.Counter(((full.get(r['id']) or {}).get(l) or {}).get('source', '缺') for r in rows) for l in ('kr', 'jp', 'en', 'tw')}
print(stats); print({l: dict(c) for l, c in by.items()})
