import sys; sys.path.insert(0,'.')
from common import *
import collections
rows=load.load()
AR=['료슈딸','료슈딸2','료슈딸연구실','아라야정장','아라야정장2','아라야10대','아라야20대','아라야공백','소지아비','소지아비2','소지아비3','소지아비B1','소지아비B2','소지아비얼굴']
cnt=collections.Counter(); hit=collections.Counter()
out=open('ara_dotted.txt','w')
for r in rows:
    if r['f']!='content' and r['f']!='text': continue
    m=r['model'] or r['speaker']
    if m not in AR: continue
    cnt[m]+=1
    a=abbrs(r['kr'])
    if a:
        hit[m]+=1
        out.write(f"{r['key'].replace('StoryData/','').replace('/dataList/','#').replace('/content','')} <{m}/{r['teller']}> [{' | '.join(a)}] KR: {r['kr']} || ZH: {r['zh']}\n")
print(cnt); print(hit)
