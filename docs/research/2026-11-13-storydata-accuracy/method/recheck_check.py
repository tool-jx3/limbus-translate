"""usage: python3 -I check_recheck.py <repo> <recheck json> <group[:start:count]> [group...]
Checks a recheck file against the report and the current worktrees; prints problems."""
import json, sys, subprocess, re, glob, os
repo, path, specs = sys.argv[1], sys.argv[2], sys.argv[3:]
F = json.load(open(repo + '/docs/research/2026-11-13-storydata-accuracy/data/findings.json'))
by = {f['seq']: f for f in F}
refs = {f['ref'] for f in F}
want = []
for s in specs:
    p = s.split(':'); g = p[0]; items = [f for f in F if f['group'] == g]
    if len(p) == 3: items = items[int(p[1]):int(p[1]) + int(p[2])]
    want += [f['seq'] for f in items]
adj = ''.join(open(p, encoding='utf-8').read() for p in glob.glob(repo + '/rules/adjustments/*.yaml'))
d = json.load(open(path))
bad = 0
def err(*a):
    global bad; bad += 1; print('!!', *a)
if [x['seq'] for x in d] != want:
    err('seq list differs: got %d want %d; missing %s extra %s' % (len(d), len(want), sorted(set(want) - {x['seq'] for x in d})[:5], sorted({x['seq'] for x in d} - set(want))[:5]))
cache = {}
def load(b):
    if b not in cache:
        cache[b] = (json.load(open('%s/.work/raw/kr/StoryData/KR_%s.json' % (repo, b), encoding='utf-8-sig'))['dataList'],
                    json.load(open('%s/.work/tw/LimbusCompany_Data/Lang/zh-TW/StoryData/%s.json' % (repo, b), encoding='utf-8-sig'))['dataList'])
    return cache[b]
def val(file, ptr, kr, zh):
    r = subprocess.run([repo + '/bin/lct', 'validate', '--kr', kr, '--zh', zh, '--file', file, '--pointer', ptr], capture_output=True, text=True, cwd=repo)
    out = (r.stdout + r.stderr).strip().splitlines()
    return r.returncode == 0 and out and 'no issues' in out[-1], (out[-3:] if out else [])
for x in d:
    f = by.get(x['seq'])
    if not f: err(x['seq'], 'unknown'); continue
    if x['result'] == 'direct':
        if x['suggest'] != f['suggest']: err(x['seq'], 'direct but text differs from report')
    elif x['result'] == 'adjust':
        if x['suggest'] == f['suggest']: err(x['seq'], 'adjust but same text')
        if x['suggest'] == f['tw']: err(x['seq'], 'adjust equals current tw')
        for k in ('kr_meaning', 'context', 'problem'):
            if not x.get(k): err(x['seq'], 'missing', k)
        ok, out = val(f['file'], f['key'].split('|')[1], f['kr'], x['suggest'])
        if not ok: err(x['seq'], 'validate', out)
    else:
        err(x['seq'], 'result', x['result'])
    for a in x.get('also') or []:
        m = re.match(r'^([A-Za-z0-9_]+)#(\d+)([ptn]?)$', a['ref'])
        if not m: err(x['seq'], 'also ref', a['ref']); continue
        b, i, suf = m.group(1), int(m.group(2)), m.group(3)
        fld = {'': 'content', 'p': 'place', 't': 'title', 'n': 'teller'}[suf]
        if a['ref'] in refs: err(x['seq'], 'also is a report item', a['ref'])
        try: kr, tw = load(b)
        except Exception as e: err(x['seq'], 'also file', a['ref'], e); continue
        if i >= len(kr): err(x['seq'], 'also index', a['ref']); continue
        if (kr[i].get(fld) or '') != a['kr']: err(x['seq'], 'also KR mismatch', a['ref'], '| file:', (kr[i].get(fld) or '')[:60], '| given:', a['kr'][:60])
        if (tw[i].get(fld) or '') != a['tw']: err(x['seq'], 'also TW mismatch', a['ref'], '| file:', (tw[i].get(fld) or '')[:60], '| given:', a['tw'][:60])
        if a['suggest'] == a['tw']: err(x['seq'], 'also no change', a['ref'])
        key = 'StoryData/%s.json|/dataList/%d/%s' % (b, i, fld)
        if key in adj: err(x['seq'], 'also already adjusted', a['ref'])
        ok, out = val('StoryData/%s.json' % b, '/dataList/%d/%s' % (i, fld), kr[i].get(fld) or '', a['suggest'])
        if not ok: err(x['seq'], 'also validate', a['ref'], out)
print('%s: %d items, %d adjust, %d also, %d problems' % (os.path.basename(path), len(d), sum(1 for x in d if x['result'] == 'adjust'), sum(len(x.get('also') or []) for x in d), bad))
