import sys; sys.path.insert(0,".")
from occ import *
from trim import trim
run=re.compile(r'(?<![가-힣])[가-힣]\.(?: ?[가-힣]\.)* ?[가-힣]\.?')
susp=set('쳇 하 훗 흥 쯧 흠 허 후 핫 음 어 뭐 아 오 응 네 예 큼 헹 훅 켁 씁 봐 돼 야 것 놈 뿐 거 님 개 자 척 속 곰 씨 서 분 둬 옷 칼 팔 혀 눈 칫 호 마 지 다 군 걸 너 해'.split())
out=[]
for r in rows:
    if r['f'] in SKIPF: continue
    if who(r): continue
    for m in run.finditer(r['kr']):
        for a in trim(m.group().strip(), r['kr'][m.end():]):
            z=zre.findall(r['zh']); j=jre.findall(r['jp']); e=[x for x in ere.findall(r['en']) if x not in ('E.G.O','EGO','LCB','LCCB','GMT','OK','HP')]
            if not (z or j or e): continue
            if len(syl(a))==2 and syl(a)[0] in susp and re.match(r'^[가-힣]\. [가-힣]$', a): continue
            out.append((r['model'] or r['speaker'] or 'FILE:'+r['path'].split('/')[-1], r['key'], a, z, r['kr'], r['zh']))
import collections
c=collections.Counter(x[0] for x in out)
print(c.most_common())
for x in out:
    print(x[0], x[1].replace('StoryData/','').replace('/dataList/','#'), '['+x[2]+']', x[3][:3], '|KR', x[4][:80].replace('\n','⏎'), '|ZH', x[5][:60].replace('\n','⏎'))
