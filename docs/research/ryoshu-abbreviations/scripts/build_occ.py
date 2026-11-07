import sys; sys.path.insert(0,".")
from occ import *
import pickle
occ=[]
manual_undotted={ # key -> list of raw abbr strings (verified by reading)
 'StoryData/P10402.json|/dataList/1/content':['별, 없'],
 'StoryData/P10402.json|/dataList/12/content':['검만예극'],
 'StoryData/S201B.json|/dataList/78/content':['내머'],
 'StoryData/S101B.json|/dataList/45/content':['모분조도'],
 'StoryData/S111B.json|/dataList/28/content':['로드','엔러쉬'],
 'StoryData/S951B.json|/dataList/24/content':['안심안지','유잼'],
 'BattleAnnouncerDlg/Announcer_Ryoshu_42.json|/dataList/11/dlg':['완소'],
 'PersonalityVoiceDlg/Voice_Ryoshu_KumoPinky_10415.json|/dataList/20/dlg':['감다뒤'],
 'PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json|/dataList/21/dlg':['모. 분'],
 'PersonalityVoiceDlg/Voice_Ryoshu_Yurodivy_10409.json|/dataList/9/dlg':['무료지비'],
}
for r in rows:
    if r['f'] in SKIPF: continue
    w=who(r)
    if not w: continue
    found=[]
    for a in abbrs(r['kr']): found.append(('strict',a))
    for m in rel.finditer(r['kr']):
        g=m.group()
        if not any(norm(g) in norm(a) for _,a in found):
            found.append(('relaxed',g))
    for a in manual_undotted.get(r['key'],[]):
        if not any(norm(a)==norm(x) for _,x in found): found.append(('undotted',a))
    for kind,a in found:
        sy=syl(a)
        # strip leading interjection followed by '. ' when rest has >=2 syllables
        while len(sy)>2 and sy[0] in INTERJ and re.match(r'^'+sy[0]+r'\.\s', a):
            a=a[a.index(' ')+1:]; sy=syl(a)
        occ.append(dict(kind=kind, abbr=a, sy=''.join(sy), who=w, key=r['key'], path=r['path'], i=r['i'], model=r['model'] or r['speaker'] or '', kr=r['kr'], zh=r['zh'], jp=r['jp'], en=r['en'], o=r['o'],
            zseg=zre.findall(r['zh']), jseg=jre.findall(r['jp']), eseg=[x for x in ere.findall(r['en']) if x not in ('E.G.O','EGO','LCB','LCCB','GMT')]))
pickle.dump(occ, open('occ.pkl','wb'))
print(len(occ), collections.Counter(o['kind'] for o in occ))
