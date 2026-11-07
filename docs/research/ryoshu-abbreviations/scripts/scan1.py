import sys, re, collections, json
sys.path.insert(0,'.')
import load
rows=load.load()
TEXTF={'content','dlg','dialog','text','desc','name','title','message','result','summary','flavor','simpleDesc','place','displayName','nickName','undefined'}
pat=re.compile(r'(?<![가-힣])[가-힣](?:\.\s?[가-힣](?![가-힣])){1,}\.?')
hits=[]
for r in rows:
    if r['f'] in ('model','teller','speaker','id','key'): continue
    kr=r['kr']
    for m in pat.finditer(kr):
        hits.append((m.group(), r))
c=collections.Counter()
spk=collections.defaultdict(collections.Counter)
for g,r in hits:
    s=r['model'] or r['speaker'] or ('FILE:'+r['path'].split('/')[-1])
    spk[s][g]+=1
tot=collections.Counter({s:sum(v.values()) for s,v in spk.items()})
for s,n in tot.most_common(80):
    print(n, s, dict(spk[s].most_common(8)))
