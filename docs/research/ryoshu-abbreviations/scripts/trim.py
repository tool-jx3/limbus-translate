import re
INTERJ=set('쳇 하 훗 흥 쯧 흠 후 핫 큼 헹 훅 켁 씁 칫 호'.split())
PART=set('은 는 이 가 을 를 의 에 도 로 와 과 나 만 처 하 할 해 했 한 인 라 란 으 랑 께 까 뿐 부 보 들'.split())
def segs(a):
    # list of (syllable, has_dot, space_before)
    out=[]; i=0; sp=False
    while i<len(a):
        c=a[i]
        if c==' ': sp=True; i+=1; continue
        if '가'<=c<='힣':
            dot = i+1<len(a) and a[i+1]=='.'
            out.append([c,dot,sp]); sp=False; i+= 2 if dot else 1
        else: i+=1
    return out
def fmt(ss):
    s=''
    for k,(c,d,sp) in enumerate(ss):
        if k and sp: s+=' '
        s+=c+('.' if d else '')
    return s
def trim(a, after):
    """a: run text; after: the text right after the run in the line. returns list of abbr strings"""
    ss=segs(a)
    # split at internal interjections preceded by space and followed by space
    parts=[[]]
    for k,s in enumerate(ss):
        if k>0 and s[2] and s[1] and s[0] in INTERJ and k+1<len(ss) and ss[k+1][2]:
            parts.append([]); continue
        parts[-1].append(s)
    res=[]
    for pi,p in enumerate(parts):
        while len(p)>2 and p[0][0] in INTERJ and p[0][1] and len(p)>1 and p[1][2]:
            p=p[1:]
        if p and not p[-1][1]:
            last=p[-1]
            nxt_hangul = (bool(after) and '가'<=after[0]<='힣') if pi==len(parts)-1 else False
            if last[2] and (nxt_hangul or last[0] in PART):
                p=p[:-1]
            elif (not last[2]) and last[0] in PART and len(p)>1 and p[-2][1]:
                p=p[:-1]
        if len(p)>=2 or (len(p)==1 and p[0][0]=='살' and p[0][1]):
            res.append(fmt(p))
    return res
