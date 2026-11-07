import re, sys
sys.path.insert(0,'.')
import load
MODELS=['료슈','료슈요리사','탐정료슈','혈귀료슈','료슈다침','료슈F','료슈2','료슈그림자','료슈3','료슈9장','료슈9장다침','료슈9장이후담배','료슈9장이후노담배','료슈9장이후반담배','료슈만우','뉴료슈노담다침','뉴료슈반담다침','뉴료슈담배다침']
OTHER=['과거료슈1','과거료슈2','과거료슈3']
NOT=['료슈딸','료슈딸연구실','료슈딸2','그렉료슈']
SKIPF={'model','teller','speaker','id','key','title','place'}
# run of >=2 single-syllable+dot tokens (optional space), or dotted w/o trailing dot
PA=re.compile(r'(?<![가-힣A-Za-z0-9.])(?:[가-힣]\. ?){2,}')
PB=re.compile(r'(?<![가-힣A-Za-z0-9.])(?:[가-힣]\.){1,}[가-힣](?![가-힣.])')
def abbrs(kr):
    out=[]
    for m in PA.finditer(kr):
        out.append((m.start(), m.group().strip()))
    for m in PB.finditer(kr):
        if not any(s<=m.start()<s+len(g)+1 for s,g in out):
            out.append((m.start(), m.group().strip()))
    out.sort()
    return [g for _,g in out]
def norm(a):
    return re.sub(r'[\s.]','',a)
def who(r):
    if r['model'] in MODELS: return 'ryoshu'
    if r['model'] in OTHER: return 'ryoshu-past'
    if r['speaker'] in MODELS: return 'ryoshu'
    fn=r['path'].split('/')[-1]
    if re.match(r'(AbDlg_Ryoshu|Voice_Ryoshu_|Voice_EGO_Ryoshu_|Announcer_Ryoshu_)', fn): return 'ryoshu'
    return None
