import sys; sys.path.insert(0,".")
from occ import *
import pickle
run=re.compile(r'(?<![가-힣])[가-힣]\.(?: ?[가-힣]\.)* ?[가-힣]\.?')
manual_undotted={
 'StoryData/P10402.json|/dataList/1/content':['별, 없.'],
 'StoryData/P10402.json|/dataList/12/content':['검만예극'],
 'StoryData/S201B.json|/dataList/78/content':['내머.'],
 'StoryData/S101B.json|/dataList/45/content':['모분조도.'],
 'StoryData/S111B.json|/dataList/28/content':['로드','엔러쉬'],
 'StoryData/S951B.json|/dataList/24/content':['안심안지','유잼'],
 'BattleAnnouncerDlg/Announcer_Ryoshu_42.json|/dataList/11/dlg':['완소'],
 'PersonalityVoiceDlg/Voice_Ryoshu_KumoPinky_10415.json|/dataList/20/dlg':['감다뒤'],
 'PersonalityVoiceDlg/Voice_Ryoshu_Yurodivy_10409.json|/dataList/9/dlg':['무료지비'],
}
occ=[]
for r in rows:
    if r['f'] in SKIPF: continue
    w=who(r)
    if not w: continue
    found=[]
    for m in run.finditer(r['kr']):
        a=m.group().strip()
        sy=syl(a)
        # strip leading interjection
        while len(sy)>2 and sy[0] in INTERJ and re.match('^'+sy[0]+r'\.\s', a):
            a=a[a.index(' ')+1:]; sy=syl(a)
        found.append(('dotted',a,m.start()))
    for a in manual_undotted.get(r['key'],[]):
        found.append(('undotted',a,r['kr'].find(a)))
    for kind,a,pos in found:
        sy=syl(a)
        occ.append(dict(kind=kind, abbr=a, sy=''.join(sy), who=w, key=r['key'], path=r['path'], i=r['i'], pos=pos, model=r['model'] or r['speaker'] or '', kr=r['kr'], zh=r['zh'], jp=r['jp'], en=r['en'], o=r['o'],
            zseg=zre.findall(r['zh']), jseg=jre.findall(r['jp']), eseg=[x for x in ere.findall(r['en']) if x not in ('E.G.O','EGO','LCB','LCCB','GMT','OK')]))
pickle.dump(occ, open('occ2.pkl','wb'))
print(len(occ))
with open('occ2.txt','w') as f:
    for n,o in enumerate(occ):
        k=o['key'].replace('StoryData/','').replace('/dataList/','#').replace('/content','').replace('PersonalityVoiceDlg/','').replace('BattleAnnouncerDlg/','').replace('RPGSystem/rpg-loc-dialogue-','RPG:').replace('/texts/','/')
        aid=' Z'+str(o['zseg']) if o['zseg'] else ''
        aid+=' J'+str(o['jseg']) if o['jseg'] else ''
        aid+=' E'+str(o['eseg']) if o['eseg'] else ''
        f.write(f"{n}\t{k}\t[{o['abbr']}]\t{aid}\n")
