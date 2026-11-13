"""Show StoryData lines with KR, JP, EN and TW.

usage: python3 -I lookup.py <ref> [context lines]
  ref is <file>#<index>, e.g. S101B#3 (content), S101B#3p (place), S101B#3t (title), S101B#3n (teller)
"""
import json, sys, os, collections
SD = os.path.dirname(os.path.abspath(__file__))
units = json.load(open(SD + '/units.json'))
ref = sys.argv[1]
ctx = int(sys.argv[2]) if len(sys.argv) > 2 else 2
base, _, rest = ref.partition('#')
suf = rest[-1] if rest and rest[-1] in 'ptn' else ''
idx = int(rest[:-1] if suf else rest)
fld = {'': 'content', 'p': 'place', 't': 'title', 'n': 'teller'}[suf]
fu = [u for u in units if u['file'].split('/')[-1][:-5] == base]
if not fu:
    sys.exit('no such file: ' + base)
content = [u for u in fu if u['f'] == 'content']
target = [u for u in fu if u['i'] == idx and u['f'] == fld]
if not target:
    sys.exit('no such unit: ' + ref)
t = target[0]
pos = next((j for j, u in enumerate(content) if u['i'] == idx), None)
show = content[max(0, pos - ctx):pos + ctx + 1] if pos is not None else []
if fld != 'content':
    show = [t] + show
for u in show:
    mark = '>>' if u is t else '  '
    print('%s %s#%d%s [%s] %s→%s' % (mark, base, u['i'], {'content': '', 'place': 'p', 'title': 't', 'teller': 'n'}[u['f']], u['o'], u['sp_kr'] or '旁白', u['sp_zh'] or '旁白'))
    for k in ('kr', 'jp', 'en', 'zh'):
        print('     %s: %s' % (k.upper() if k != 'zh' else 'TW', u[k].replace('\n', '⏎')))
