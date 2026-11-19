"""usage: python3 -I ctx.py <repo> <group> [start] [count] [ctx]
Prints findings of a group (main-05, dungeon-05, event-e5) with context from the current raw/tw worktrees."""
import json, sys, os
repo=sys.argv[1]; group=sys.argv[2]
start=int(sys.argv[3]) if len(sys.argv)>3 else 0
count=int(sys.argv[4]) if len(sys.argv)>4 else 999
C=int(sys.argv[5]) if len(sys.argv)>5 else 3
F=json.load(open(repo+'/docs/research/2026-11-13-storydata-accuracy/data/findings.json'))
items=[f for f in F if f['group']==group][start:start+count]
cache={}
def load(f):
    if f not in cache:
        b=os.path.basename(f)
        d={}
        for l in ('kr','jp','en'):
            p='%s/.work/raw/%s/StoryData/%s_%s'%(repo,l,l.upper(),b)
            d[l]=json.load(open(p,encoding='utf-8-sig'))['dataList']
        d['tw']=json.load(open('%s/.work/tw/LimbusCompany_Data/Lang/zh-TW/%s'%(repo,f),encoding='utf-8-sig'))['dataList']
        cache[f]=d
    return cache[f]
def s(x): return (x or '').replace('\n','⏎')
for f in items:
    d=load(f['file']); idx=int(f['key'].split('/dataList/')[1].split('/')[0]); fld=f['field']
    cur_kr=d['kr'][idx].get(fld); cur_tw=d['tw'][idx].get(fld)
    flag=[]
    if cur_kr!=f['kr']: flag.append('KR已變')
    if cur_tw!=f['tw']: flag.append('TW已變:'+s(cur_tw))
    print('=== %s %s [%s|%s|信心%s] %s 來源%s %s'%(f['seq'],f['ref'],f['category'],f['severity'],f['confidence'],f['speaker'],f['origin'],' '.join(flag)))
    for i in range(max(0,idx-C),min(len(d['kr']),idx+C+1)):
        k=d['kr'][i]; t=d['tw'][i]
        sp=k.get('teller') or k.get('model') or ''
        if i==idx:
            print(' >>#%d [%s]'%(i,sp))
            print('    KR: '+s(k.get(fld))); print('    JP: '+s(d['jp'][i].get(fld))); print('    EN: '+s(d['en'][i].get(fld))); print('    TW: '+s(t.get(fld)))
        else:
            if k.get('content'): print('   #%d [%s] %s ‖ %s'%(i,sp,s(k.get('content')),s(t.get('content'))))
    print('    建議: '+s(f['suggest']))
    print('    說明: '+s(f['explain']))
    if f.get('reason'): print('    複核: '+s(f['reason']))
