import sys,re,json,collections; sys.path.insert(0,'.')
from common import *
rows=load.load()
byfile=collections.defaultdict(dict)
for r in rows:
    if r['f'] in ('content','text','dlg','desc','name','title'):
        if r['i'] is not None:
            byfile[r['path']].setdefault(r['i'],[]).append(r)
rel=re.compile(r'(?<![가-힣])[가-힣]\.(?: ?[가-힣]\.)* ?[가-힣](?![.가-힣])|(?<![가-힣])[가-힣]\.[가-힣]')
INTERJ=set('쳇 하 훗 흥 쯧 흠 허 후 핫 음 어 뭐 아 오 응 네 예 큼 헹 훅 켁 씁'.split())
zre=re.compile(r'[^\s．，。！？…「」（）、<>]．(?:[^\s．，。！？…「」（）、<>]．)*[^\s．，。！？…「」（）、<>]')
jre=re.compile(r'[ぁ-んァ-ヶー一-龥（）()]{1,4}(?:・[ぁ-んァ-ヶー一-龥]{1,5})+')
ere=re.compile(r'\b(?:[A-Z]\.){1,}[A-Z]?\.?(?![a-z])|\b[A-Z]{2,}\b')
def syl(a): return [c for c in a if '가'<=c<='힣']
def eoj(text):
    t=re.sub(r'<[^>]+>','',text)
    return [w for w in re.split(r'[\s,.!?…~\-"\'“”‘’()（）/]+',t) if w]
def find_exp(sy, text, skip_abbr=True):
    ws=eoj(text)
    n=len(sy)
    for k in range(len(ws)-n+1):
        seg=ws[k:k+n]
        if all(seg[j][0]==sy[j] for j in range(n)) and any(len(w)>1 for w in seg):
            return ' '.join(seg)
    return None
