import sys; sys.path.insert(0,'.')
import pickle,re,collections
occ=pickle.load(open('occ3.pkl','rb'))
susp=set('쳇 하 훗 흥 쯧 흠 허 후 핫 음 어 뭐 아 오 응 네 예 큼 헹 훅 켁 씁 봐 돼 야 것 놈 뿐 거 님 개 자 척 속 곰 씨 서 분 둬 옷 칼 팔 혀 눈 칫 호 마 지 다 군 걸 너 해'.split())
FPIDX=set()
for n,o in enumerate(occ):
    if not (o['zseg'] or o['jseg'] or o['eseg']): FPIDX.add(n); continue
    if len(o['sy'])==2 and o['sy'][0] in susp and re.match(r'^[가-힣]\. [가-힣]$', o['abbr']): FPIDX.add(n)
extra_fp=set(map(int,open('fp_extra.txt').read().split())) if __import__('os').path.exists('fp_extra.txt') else set()
FPIDX|=extra_fp
good=[(n,o) for n,o in enumerate(occ) if n not in FPIDX]
g=collections.OrderedDict()
for n,o in good:
    g.setdefault(o['sy'],[]).append((n,o))
def short(k):
    return k.replace('StoryData/','').replace('/dataList/','#').replace('/content','').replace('PersonalityVoiceDlg/','').replace('BattleAnnouncerDlg/','').replace('EGOVoiceDig/','').replace('RPGSystem/rpg-loc-dialogue-','RPG:').replace('/texts/','/').replace('.json','').replace('/dlg','').replace('/text','')
if __name__=='__main__':
    print(len(good), len(g))
    for sy,lst in sorted(g.items(), key=lambda x:-len(x[1])):
        print(f"== {sy} ({len(lst)})")
        for n,o in lst:
            print(f"  {n} {short(o['key'])} [{o['abbr']}] Z{o['zseg'][:3]} J{[x for x in o['jseg'] if len(x)<14][:3]} E{o['eseg'][:3]}")
