import sys; sys.path.insert(0,'.')
import pickle
from overrides import OV
occ=pickle.load(open('occ2.pkl','rb'))
out=[]
for o in occ:
    ov=OV.get(o['key'],{})
    if o['abbr'] in ov:
        new=ov[o['abbr']]
        if new is None: continue
        news=new if isinstance(new,list) else [new]
        for a in news:
            p=dict(o); p['abbr']=a; p['sy']=''.join(c for c in a if '가'<=c<='힣'); out.append(p)
    else: out.append(o)
pickle.dump(out, open('occ3.pkl','wb')); print(len(out))
# --- round 1: add occurrences the pipeline missed (reviewer)
import load
from occ import zre, jre, ere, syl
rows={r['key']:r for r in load.load()}
ADD={'StoryData/E030X.json|/dataList/94/content':'알. 못',
     'StoryData/E908B.json|/dataList/45/content':'흥. 이.',
     'PersonalityVoiceDlg/Voice_Ryoshu_Butler_10408.json|/dataList/21/dlg':'모. 분'}
have={(o['key'],o['abbr']) for o in out}
for k,a in ADD.items():
    if (k,a) in have: continue
    r=rows[k]
    out.append(dict(kind='dotted', abbr=a, sy=''.join(syl(a)), who='ryoshu', key=k, path=r['path'], i=r['i'], pos=r['kr'].find(a), model=r['model'] or r['speaker'] or '', kr=r['kr'], zh=r['zh'], jp=r['jp'], en=r['en'], o=r['o'],
        zseg=zre.findall(r['zh']), jseg=jre.findall(r['jp']), eseg=[x for x in ere.findall(r['en']) if x not in ('E.G.O','EGO','LCB','LCCB','GMT','OK','HP')]))
pickle.dump(out, open('occ3.pkl','wb')); print('after add', len(out))
