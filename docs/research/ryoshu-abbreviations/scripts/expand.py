import sys; sys.path.insert(0,'.')
from occ import *
import pickle
exec(open('group.py').read().split("if __name__")[0])
# ordered text units per file
seq=collections.defaultdict(list)
for r in rows:
    if r['f'] in ('content','text','dlg'):
        seq[r['path']].append(r)
pos={}
for p,l in seq.items():
    for k,r in enumerate(l): pos[r['key']]=(p,k)
def initials_match(sy, text):
    t=re.sub(r'<[^>]+>','',text)
    ws=[w for w in re.split(r'[\s,.!?…~\-"\'“”‘’()（）/]+',t) if w and '가'<=w[0]<='힣']
    n=len(sy)
    for k in range(len(ws)-n+1):
        seg=ws[k:k+n]
        if all(seg[j][0]==sy[j] for j in range(n)) and sum(len(w)>1 for w in seg)>=max(1,n-1):
            # avoid matching the abbreviation itself (all words len1)
            return ' '.join(seg)
    return None
res={}
for n,o in good:
    sy=o['sy']
    if len(sy)<2: continue
    p,k=pos.get(o['key'],(None,None))
    cands=[]
    # same unit: text after abbr
    after=o['kr'][o['pos']+len(o['abbr']):] if o['pos']>=0 else o['kr']
    m=initials_match(sy, after)
    if m: cands.append(('same', o['key'], m))
    if p and (p.startswith('StoryData') or p.startswith('RPGSystem')):
        for d in list(range(1,7))+[-1,-2,-3]:
            kk=k+d
            if 0<=kk<len(seq[p]):
                r=seq[p][kk]
                m=initials_match(sy, r['kr'])
                if m: cands.append((d, r['key'], m, r['model'] or r['speaker']))
    if cands: res[n]=cands
pickle.dump(res, open('exp.pkl','wb'))
for n,c in res.items():
    o=occ[n]
    print(n, short(o['key']), '['+o['abbr']+']', '=>', c[:2])
