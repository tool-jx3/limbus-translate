"""Prepare StoryData review material.

usage: python3 -I prep.py <adjclone> <repo> <out dir>
"""
import json, glob, collections, re, os, sys
A, REPO, OUT = sys.argv[1], sys.argv[2], sys.argv[3]
# speaker names
smc_kr = json.load(open(A + '/.work/raw/kr/KR_ScenarioModelCodes-AutoCreated.json', encoding='utf-8-sig'))['dataList']
smc_site = json.load(open(A + '/site/public/data/files/ScenarioModelCodes-AutoCreated.json'))['units']
zh_by_path = {u['p']: u.get('zh') for u in smc_site}
names = {}
for i, el in enumerate(smc_kr):
    names[el['id']] = (el.get('name'), zh_by_path.get('/dataList/%d/name' % i), el.get('nickName'), zh_by_path.get('/dataList/%d/nickName' % i))
# character guides
guides = {}
for g in sorted(glob.glob(REPO + '/guides/characters/*.md')):
    s = open(g, encoding='utf-8').read()
    fm = s.split('---')[1]
    def lst(k):
        m = re.search(r'^%s:\s*\[(.*?)\]' % k, fm, re.M)
        return [x.strip().strip('"\'') for x in m.group(1).split(',') if x.strip()] if m else []
    gid = re.search(r'^id:\s*(\S+)', fm, re.M).group(1)
    guides[gid] = {'models': lst('models'), 'other': lst('models_other_self'), 'not': lst('models_not_this'),
                   'name': re.search(r'^name:\s*(.+)', fm, re.M).group(1).strip()}
model_char = {}
for gid, g in guides.items():
    for m in g['models']: model_char[m] = (gid, 'self')
    for m in g['other']: model_char[m] = (gid, 'other')

units = []
for f in sorted(glob.glob(A + '/site/public/data/files/StoryData/*.json')):
    d = json.load(open(f))
    file = d['path']
    by_i = collections.defaultdict(dict)
    for u in d['units']:
        by_i[u['i']][u['f']] = u
    for i in sorted(by_i):
        el = by_i[i]
        model = el.get('model', {}).get('kr') or ''
        tel = el.get('teller')
        nm = names.get(model, (model, model, None, None))
        sp_kr = tel['kr'] if tel else (nm[0] or model)
        sp_zh = tel.get('zh') if tel else (nm[1] or model)
        for fld in ('title', 'place', 'teller', 'content'):
            u = el.get(fld)
            if not u: continue
            units.append({'file': file, 'i': i, 'f': fld, 'key': file + '|' + u['p'], 'model': model,
                          'sp_kr': sp_kr if model or tel else '', 'sp_zh': sp_zh if model or tel else '',
                          'kr': u['kr'], 'zh': u.get('zh') or '', 'jp': u.get('jp') or '', 'en': u.get('en') or '',
                          'o': u.get('o') or ''})
os.makedirs(OUT, exist_ok=True)
json.dump(units, open(OUT + '/units.json', 'w', encoding='utf-8'), ensure_ascii=False)
json.dump({'names': names, 'guides': guides}, open(OUT + '/meta.json', 'w', encoding='utf-8'), ensure_ascii=False)
c = collections.Counter(); unguided = collections.Counter()
for u in units:
    if u['f'] != 'content' or not u['model']: continue
    mc = model_char.get(u['model'])
    if mc: c[mc[0]] += 1
    else: unguided[u['model']] += 1
print('units', len(units))
print('guided', sorted(c.items(), key=lambda t: -t[1]))
print('unguided top', unguided.most_common(45))
