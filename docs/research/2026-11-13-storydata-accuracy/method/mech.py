"""Mechanical checks over StoryData TW vs KR. usage: python3 -I mech.py <sd dir>"""
import json, re, collections, sys
SD = sys.argv[1]
units = json.load(open(SD + '/units.json'))
TAG = re.compile(r'<[^>]*>')
HANGUL = re.compile(r'[가-힣ㄱ-ㅎㅏ-ㅣ]')
NUM = re.compile(r'\d+(?:[.,]\d+)?')
KNUM = {'하나': 1, '둘': 2, '셋': 3, '넷': 4, '다섯': 5}
out = []
D = {'零': 0, '〇': 0, '一': 1, '二': 2, '兩': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
U = {'十': 10, '百': 100, '千': 1000, '萬': 10000}
def cn_numbers(z):
    """Arabic values of the Chinese numerals in z (simple forms up to 99999, and digit runs like 一九)."""
    vals = set()
    for m in re.finditer(r'[零〇一二兩三四五六七八九十百千萬]+', z):
        t = m.group(0)
        total, sec, num = 0, 0, 0
        ok = True
        for ch in t:
            if ch in D:
                num = num * 10 + D[ch] if num and not any(c in U for c in t) else D[ch]
            elif ch in U:
                if U[ch] == 10000:
                    total += (sec + (num or 0)) * 10000; sec = num = 0
                else:
                    sec += (num or 1) * U[ch]; num = 0
        vals.add(str(total + sec + num))
        if not any(c in U for c in t):
            vals.add(''.join(str(D[c]) for c in t if c in D))
    return vals
def ref(u):
    base = u['file'].split('/')[-1][:-5]
    return '%s#%d%s' % (base, u['i'], {'content': '', 'place': 'p', 'title': 't', 'teller': 'n'}[u['f']])
for u in units:
    kr, zh = u['kr'], u['zh']
    if u['f'] != 'content':
        continue
    kz = TAG.sub('', zh); kk = TAG.sub('', kr)
    # residual hangul in TW (outside ruby/noparse content is hard to tell; report all)
    h = HANGUL.findall(kz.replace('하나', ''))
    if h and HANGUL.search(kk) and u['o'] != 'copy':
        out.append({'ref': ref(u), 'check': 'hangul', 'detail': ''.join(h)[:20]})
    # numbers
    nk = collections.Counter(n.replace(',', '') for n in NUM.findall(kk))
    nz = collections.Counter(n.replace(',', '') for n in NUM.findall(kz))
    if nk and nk != nz:
        miss = nk - nz
        cn = cn_numbers(kz)
        miss = collections.Counter({n: c for n, c in miss.items() if n not in cn})
        if miss:
            out.append({'ref': ref(u), 'check': 'number', 'detail': 'KR 有 %s，TW 沒有' % '、'.join(sorted(miss))})
    # very short TW
    lk = len(re.sub(r'[\s\W]', '', kk)); lz = len(re.sub(r'[\s\W]', '', kz))
    if lk >= 25 and lz < lk * 0.35:
        out.append({'ref': ref(u), 'check': 'short', 'detail': 'KR %d 字，TW %d 字' % (lk, lz)})
json.dump(out, open(SD + '/mech.json', 'w', encoding='utf-8'), ensure_ascii=False, indent=0)
print(len(out), collections.Counter(x['check'] for x in out))
