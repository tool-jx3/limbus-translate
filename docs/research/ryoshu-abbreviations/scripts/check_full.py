# 核對 full_forms.json：每個 id 都有、source 合法、「遊戲內」的 full 原樣出現在 loc 那個單位的該語言文字裡
import sys, json, glob, re, collections
sys.path.insert(0, '.')
import load
rows = load.load()
byloc = {}
for r in rows:
    if r['i'] is None: continue
    byloc.setdefault(r['path'] + '#' + str(r['i']), []).append(r)
F = {'kr': 'kr', 'jp': 'jp', 'en': 'en', 'tw': 'zh'}
want = {r['id'] for r in json.load(open('rows4.json'))}
got = {}; bad = []
for f in ['full_forms.json']:
    for o in json.load(open(f)):
        got[o['id']] = o
        for L, fld in F.items():
            x = o.get(L) or {}
            s = x.get('source')
            if s not in ('遊戲內', '推測', '未縮', '不明'): bad.append((f, o['id'], L, 'source', s)); continue
            if s == '遊戲內':
                loc = x.get('loc', ''); full = x.get('full', '')
                us = byloc.get(loc) or byloc.get(re.sub(r'#(\d+)/.*', r'#\1', loc))
                if not us: bad.append((f, o['id'], L, 'no such loc', loc)); continue
                norm = lambda t: re.sub(r'\s+', '', t)
                if not any(norm(full) and norm(full) in norm(u[fld]) for u in us):
                    bad.append((f, o['id'], L, 'not in text', full, loc))
miss = sorted(want - set(got)); extra = sorted(set(got) - want)
print('rows', len(want), 'got', len(got), 'missing', len(miss), miss[:10], 'extra', extra[:5])
print('bad', len(bad))
for b in bad: print(' ', b)
c = collections.Counter((L, (o.get(L) or {}).get('source')) for o in got.values() for L in F)
print(dict(sorted(c.items())))
