"""Write the StoryData accuracy-review chunks.

usage: python3 -I chunk.py <sd dir> <target chars per chunk>
writes <sd>/acc/aNN.txt and <sd>/acc/refs.json (ref -> unit key)
"""
import json, collections, os, sys
SD, TARGET = sys.argv[1], int(sys.argv[2])
units = json.load(open(SD + '/units.json'))
by_file = collections.OrderedDict()
for u in units:
    by_file.setdefault(u['file'], collections.defaultdict(dict))[u['i']][u['f']] = u
cl = lambda s: (s or '').replace('\n', '⏎')
refs = {}
blocks = []
for f, els in by_file.items():
    base = f.split('/')[-1][:-5]
    lines = ['## %s' % base]
    size = 0
    for i in sorted(els):
        el = els[i]
        if 'place' in el:
            u = el['place']; r = '%s#%dp' % (base, i); refs[r] = u['key']
            lines.append('%dp〈地點〉%s ⇒ %s' % (i, cl(u['kr']), cl(u['zh'])))
        c = el.get('content')
        if not c:
            continue
        r = '%s#%d' % (base, i); refs[r] = c['key']
        if el.get('teller'):
            refs['%s#%dn' % (base, i)] = el['teller']['key']
        if el.get('title'):
            refs['%s#%dt' % (base, i)] = el['title']['key']
        if c['model'] or el.get('teller'):
            tk = el.get('title', {}).get('kr'); tz = el.get('title', {}).get('zh')
            sp = '%s%s→%s%s' % (c['sp_kr'], '［%s］' % tk if tk else '', c['sp_zh'], '［%s］' % tz if tz else '')
        else:
            sp = '（旁白）'
        lines.append('%d %s：%s ⇒ %s' % (i, sp, cl(c['kr']), cl(c['zh'])))
        size += len(c['kr']) + len(c['zh'])
    blocks.append((f, lines, size))
os.makedirs(SD + '/acc', exist_ok=True)
chunks, cur, cs = [], [], 0
for b in blocks:
    if cur and cs + b[2] > TARGET:
        chunks.append(cur); cur, cs = [], 0
    cur.append(b); cs += b[2]
if cur:
    chunks.append(cur)
index = []
for k, ch in enumerate(chunks, 1):
    name = 'a%02d' % k
    with open(SD + '/acc/%s.txt' % name, 'w', encoding='utf-8') as w:
        for f, lines, _ in ch:
            w.write('\n'.join(lines) + '\n\n')
    index.append({'chunk': name, 'files': [f for f, _, _ in ch], 'chars': sum(s for _, _, s in ch),
                  'lines': sum(len(l) - 1 for _, l, _ in ch)})
json.dump(refs, open(SD + '/acc/refs.json', 'w', encoding='utf-8'), ensure_ascii=False)
json.dump(index, open(SD + '/acc/index.json', 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
print(len(chunks), [(x['chunk'], x['lines'], x['chars']) for x in index])
