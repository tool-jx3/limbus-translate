import sys,re,collections,pickle; sys.path.insert(0,'.')
exec(open('group.py').read().split("if __name__")[0])
import load
byk={r['key']:r for r in load.load()}
res=pickle.load(open('exp.pkl','rb'))
byline=collections.OrderedDict()
for n,o in good: byline.setdefault(o['key'],[]).append((n,o))
def pair(o, segs, lst, idx):
    if len(segs)==len(lst): return segs[idx]
    if len(segs)==1: return segs[0]
    return '／'.join(segs) if segs else ''
seginfo={}
for k,l in byline.items():
    for idx,(n,o) in enumerate(l):
        js=[x for x in o['jseg'] if len(x)<16]
        seginfo[n]=(pair(o,o['zseg'],l,idx), pair(o,js,l,idx), pair(o,o['eseg'],l,idx))
def sk(k):
    s=short(k)
    return s.replace('Announcer_Ryoshu_42','Announcer_Ryoshu').replace('Voice_Ryoshu_','Voice_')
rowsout=[]
g=collections.OrderedDict()
for n,o in good: g.setdefault(o['sy']+('*' if o['kind']!='dotted' else ''),[]).append((n,o))
for sy,lst in sorted(g.items(), key=lambda x:(-len(x[1]),x[0])):
    forms=collections.Counter(o['abbr'] for n,o in lst)
    rep=forms.most_common(1)[0][0]
    zc=collections.Counter(seginfo[n][0] or '（無）' for n,o in lst)
    jc=collections.Counter(seginfo[n][1] or '—' for n,o in lst)
    ec=collections.Counter(seginfo[n][2] or '—' for n,o in lst)
    exp=''
    for n,o in lst:
        if n in res:
            x=res[n][0]
            r=byk[x[1]]
            spk='同句' if x[0]=='same' else (r['model'] or r['speaker'] or '旁白')
            exp=f"{x[2]}（{spk}，`{sk(x[1])}`）"
            break
    keys=[sk(o['key']) for n,o in lst]
    ks=', '.join(f'`{k}`' for k in keys[:5]) + (f' 等 {len(keys)} 處' if len(keys)>5 else '')
    def fmt(c): return '、'.join(f'{k}×{v}' if v>1 else k for k,v in c.most_common())
    jl=[k for k in jc if k!='—']
    jrep=min(jl, key=len) if jl else '—'
    el=[k for k,v in ec.most_common() if k!='—']
    erep=el[0] if el else '—'
    rowsout.append((rep, len(lst), exp or '—', jrep, erep, fmt(zc), ks))
pickle.dump(rowsout, open('table.pkl','wb'))
with open('table.md','w') as f:
    f.write('| KR | 次數 | 劇中展開 | JP | EN | 既有繁中 | 出處 |\n|---|---|---|---|---|---|---|\n')
    for r in rowsout:
        f.write('| '+' | '.join(str(x).replace('|','｜').replace('\n',' ') for x in r)+' |\n')
print(len(rowsout))
