# 第 3 輪：依 KR 縮略語與 JP／EN／繁中的寫法分行，附上下文，供補完整說法
# 輸出 rows4.json：每行一個（縮略語, JP, EN, 繁中）的組合與它出現的位置、上下文
import sys, re, json, pickle, collections, io, contextlib
sys.path.insert(0, '.')
from tokens import letters
with contextlib.redirect_stdout(io.StringIO()):
    import table2 as T
toks, res, byk = T.toks, T.res, T.byk
import load
rows = load.load()
bypath = collections.defaultdict(list)
for r in rows: bypath[r['path']].append(r)
pos = {}
for p, l in bypath.items():
    for i, r in enumerate(l): pos[r['key']] = (p, i)

def loc(t):
    path = t['key'].split('|')[0]
    return path + t['sk'].split('|')[1]          # StoryData/S1005B.json#35、RPGSystem/…#22/9
def who(r): return r['model'] or r['speaker'] or r['teller'] or '—'
def line(r): return {'who': who(r), 'loc': r['path'] + '#' + str(r['i']), 'kr': r['kr'], 'jp': r['jp'], 'en': r['en'], 'tw': r['zh']}
def ctx(key, n=3):
    p, i = pos[key]; l = bypath[p]
    return [dict(line(l[j]), here=(j == i)) for j in range(max(0, i - n), min(len(l), i + n + 1))]

def table(g):
    cls, zf, prop = T.CLS.get(g, ('一致', False, ''))
    if zf: return '中文不一'
    return cls or '一致'

G = collections.OrderedDict()
for t in toks: G.setdefault(t['g'], []).append(t)
out = []
for g, l in G.items():
    cls, zf, prop = T.CLS.get(g, ('一致', False, ''))
    var = collections.OrderedDict()
    for t in l:
        e = letters(t['e']); e = T.ALIAS.get(e, e)
        var.setdefault((T.jn(t['j']), e, t['z']), []).append(t)
    exp = T.EXP.get(g) or T.SEXP.get(g)
    if exp is None and len(l) == 1 and l[0]['n'] in res:
        x = res[l[0]['n']][0]; r = byk[x[1]]
        exp = f"{x[2]}（{'同句' if x[0] == 'same' else who(r)}，{r['path']}#{r['i']}）"
    for (jk, ek, z), ts in var.items():
        ec = collections.Counter(t['e'] for t in ts)
        out.append({
            'id': len(out), 'g': g, 'table': table(g), 'class': '＋'.join(x for x in (cls, '中文不一' if zf else '') if x) or '一致',
            'proposal': prop, 'note': T.NOTE.get(g, ''), 'n_kind': len(l),
            'kr': collections.Counter(t['kr'] for t in ts).most_common(1)[0][0],
            'kr_forms': sorted(set(t['kr'] for t in ts)),
            'jp': collections.Counter(t['j'] for t in ts).most_common(1)[0][0], 'jp_forms': sorted(set(t['j'] for t in ts)),
            'en': ec.most_common(1)[0][0], 'en_forms': sorted(ec), 'tw': z,
            'locs': [loc(t) for t in ts], 'expansion_known': exp or '',
            'context': [ctx(t['key']) for t in ts[:3]],
        })
json.dump(out, open('rows4.json', 'w'), ensure_ascii=False, indent=1)
c = collections.Counter(r['table'] for r in out)
print(len(out), 'rows', dict(c), 'kinds', len(G))
print('kinds per table', dict(collections.Counter(table(g) for g in G)))
