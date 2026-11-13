"""Build verification batches from the first-pass findings.

usage: python3 -I vbuild.py <sd dir> <batch size> <chunks, comma-separated, or all> <out prefix> [mech]
reads acc/out/a*.json (and a*o.json re-reviews) and mech.json; writes vb/vNN.json and vb/all.json
"""
import json, glob, collections, os, sys, re
SD, N = sys.argv[1], int(sys.argv[2])
WANT = None if sys.argv[3] == 'all' else set(sys.argv[3].split(','))
PREFIX = sys.argv[4]
MECH = len(sys.argv) > 5 and sys.argv[5] == 'mech'
units = json.load(open(SD + '/units.json'))
refs = json.load(open(SD + '/acc/refs.json'))
by_key = {u['key']: u for u in units}
by_file = collections.defaultdict(list)
for u in units:
    if u['f'] == 'content':
        by_file[u['file']].append(u)
pos = {u['key']: (u['file'], j) for f, us in by_file.items() for j, u in enumerate(us)}
cl = lambda s: (s or '').replace('\n', '⏎')
def ctx(key, n=3):
    u = by_key[key]
    if u['f'] != 'content':
        # use the content line of the same element
        c = next((x for x in by_file[u['file']] if x['i'] == u['i']), None)
        if not c:
            return []
        key = c['key']
    f, j = pos[key]
    out = []
    for x in by_file[f][max(0, j - n):j + n + 1]:
        out.append({'ref': '%s#%d' % (f.split('/')[-1][:-5], x['i']), 'self': x['key'] == key,
                    'speaker': '%s→%s' % (x['sp_kr'] or '旁白', x['sp_zh'] or '旁白'), 'origin': x['o'],
                    'kr': cl(x['kr']), 'jp': cl(x['jp']), 'en': cl(x['en']), 'tw': cl(x['zh'])})
    return out
items = collections.OrderedDict()
problems = collections.Counter()
src_files = sorted(glob.glob(SD + '/acc/out/a*.json'))
for f in src_files:
    chunk = os.path.basename(f)[:-5]
    if WANT is not None and chunk not in WANT:
        continue
    for r in json.load(open(f)):
        ref = r.get('ref', '').strip()
        key = refs.get(ref)
        if not key:
            problems['unknown ref'] += 1
            continue
        u = by_key[key]
        if cl(u['kr']) != r.get('kr') or cl(u['zh']) != r.get('tw'):
            problems['kr/tw not exact (fixed from data)'] += 1
        rec = items.setdefault(ref, {'ref': ref, 'key': key, 'field': u['f'], 'origin': u['o'],
                                     'kr': cl(u['kr']), 'tw': cl(u['zh']), 'findings': [], 'context': ctx(key)})
        rec['findings'].append({'from': chunk, 'category': r.get('category'), 'severity': r.get('severity'),
                                'explain': r.get('explain'), 'suggest': r.get('suggest'), 'confidence': r.get('confidence')})
for m in (json.load(open(SD + '/mech.json')) if MECH else []):
    key = refs.get(m['ref'])
    if not key:
        continue
    u = by_key[key]
    rec = items.setdefault(m['ref'], {'ref': m['ref'], 'key': key, 'field': u['f'], 'origin': u['o'],
                                      'kr': cl(u['kr']), 'tw': cl(u['zh']), 'findings': [], 'context': ctx(key)})
    rec['findings'].append({'from': 'mech', 'category': 'M', 'severity': None,
                            'explain': '腳本檢查：%s' % m['detail'], 'suggest': None, 'confidence': None})
os.makedirs(SD + '/vb', exist_ok=True)
allv = list(items.values())
json.dump(allv, open(SD + '/vb/%s-all.json' % PREFIX, 'w', encoding='utf-8'), ensure_ascii=False)
for k in range(0, len(allv), N):
    json.dump(allv[k:k + N], open(SD + '/vb/%s%02d.json' % (PREFIX, k // N + 1), 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
print('items', len(allv), 'batches', (len(allv) + N - 1) // N, dict(problems),
      'multi', sum(1 for x in allv if len(x['findings']) > 1))
