import sys; sys.path.insert(0,'.')
from common import *
rows=load.load()
n=0; lines=0
out=open('ryo_dotted.tsv','w')
for r in rows:
    if r['f'] in SKIPF: continue
    w=who(r)
    if not w: continue
    lines+=1
    a=abbrs(r['kr'])
    if a:
        n+=1
        out.write('\t'.join([w, r['key'], r['model'] or r['speaker'] or '', ' | '.join(a), r['kr'].replace('\n','⏎'), r['zh'].replace('\n','⏎'), r['o'] or '', r['jp'].replace('\n','⏎'), r['en'].replace('\n','⏎')])+'\n')
print(lines, n)
