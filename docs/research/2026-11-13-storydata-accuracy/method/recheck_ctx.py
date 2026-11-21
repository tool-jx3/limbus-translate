"""usage: python3 -I ctx.py <repo> <group> [start] [count] [ctx]
Prints findings of a group (main-05, dungeon-05, event-e5) with context from the current raw/tw worktrees."""
import json, sys, os
repo=sys.argv[1]; group=sys.argv[2]
start=int(sys.argv[3]) if len(sys.argv)>3 else 0
count=int(sys.argv[4]) if len(sys.argv)>4 else 999
C=int(sys.argv[5]) if len(sys.argv)>5 else 3
F=json.load(open(repo+'/docs/research/2026-11-13-storydata-accuracy/data/findings.json'))
items=[f for f in F if f['group']==group][start:start+count]
import glob
try:
    import yaml
    ADJ={}
    for _p in sorted(glob.glob(repo+'/rules/adjustments/*.yaml')):
        for _e in yaml.safe_load(open(_p,encoding='utf-8')) or []:
            ADJ[_e['key']]=(_e['zh'],_e.get('reason',''),os.path.basename(_p))
except ImportError:
    ADJ={}
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
    adj=ADJ.get(f['key'])
    if s(cur_kr)!=s(f['kr']): flag.append('KR已變')   # the report writes line breaks as ⏎
    if adj: flag.append('已寫成人工調整')
    elif s(cur_tw)!=s(f['tw']): flag.append('TW已變:'+s(cur_tw))
    print('=== %s %s [%s|%s|信心%s] %s 來源%s %s'%(f['seq'],f['ref'],f['category'],f['severity'],f['confidence'],f['speaker'],f['origin'],' '.join(flag)))
    for i in range(max(0,idx-C),min(len(d['kr']),idx+C+1)):
        k=d['kr'][i]; t=d['tw'][i]
        sp=k.get('teller') or k.get('model') or ''
        if i==idx:
            print(' >>#%d [%s]'%(i,sp))
            print('    KR: '+s(k.get(fld))); print('    JP: '+s(d['jp'][i].get(fld))); print('    EN: '+s(d['en'][i].get(fld))); print('    TW: '+s(t.get(fld)))
        else:
            if k.get('content'): print('   #%d [%s] %s ‖ %s'%(i,sp,s(k.get('content')),s(t.get('content'))))
    if adj:
        print('    （上面的 TW 是已套用人工調整後的現行譯文）')
        print('    報告當時的現行: '+s(f['tw']))
        print('    人工調整: '+s(adj[0])+('　〔與報告建議相同〕' if s(adj[0])==s(f['suggest']) else '　〔與報告建議不同〕')+('　〔注意：與現行 TW 不同〕' if s(adj[0])!=s(cur_tw) else ''))
        print('    調整理由: '+s(adj[1]))
    print('    報告建議: '+s(f['suggest']))
    print('    說明: '+s(f['explain']))
    if f.get('reason'): print('    複核: '+s(f['reason']))
    rc=f.get('recheck')
    if rc:
        if rc['result']=='direct':
            print('    前次再檢視(%s): direct（可以直接照建議）%s'%(rc['date'],('；備註：'+s(rc['note'])) if rc.get('note') else ''))
        else:
            print('    前次再檢視(%s): adjust → %s'%(rc['date'],s(rc['suggest'])))
            print('      前次指出的問題: '+s(rc['problem']))
            if rc.get('note'): print('      前次備註: '+s(rc['note']))
        for a in rc.get('also') or []:
            print('    前次建議一起改 %s: %s → %s（%s）'%(a['ref'],s(a['tw']),s(a['suggest']),s(a['why'])))
