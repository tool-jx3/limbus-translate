# 第 3 節的表用：每處縮略語在 JP／EN／繁中裡對應的那一段
# 先自動抓（JP「・」連起來的片段、EN 大寫縮寫、繁中「．」連起來的片段，同一行有幾個縮略語就依序配對），
# 再以 tokens_ov.py（人工讀過整行的修正）覆蓋。輸出 tok.pkl
import sys, re, collections, pickle
sys.path.insert(0, '.')
exec(open('group.py').read().split("if __name"+"__")[0])   # occ, good, short
from tokens_ov import OV
# 接在縮略語後的 이다 的「다.」併入本體；阿賴耶式流行語（不加點）與良秀加點的同音節縮略語分開
MERGE = {'거거다': '거거', '그그다': '그그', '만단지예다': '만단지예'}
BORROWED = {'완소', '유잼', '감다뒤', '안심안지'}

SEP = r'\s・、。，,！？!?…「」『』（）()<>〈〉“”"\'.～~―—\-：:；;／/\[\]'
JRUN = re.compile(r'[^' + SEP + r']+(?:・[^' + SEP + r']+)+')
def script(c):
    if '一' <= c <= '鿿' or c in '々〆': return 'K'
    if '゠' <= c <= 'ヿ': return 'A'
    if '぀' <= c <= 'ゟ': return 'H'
    return 'O'
def tail(u):
    s = script(u[-1]); k = len(u)
    while k > 0 and script(u[k-1]) == s: k -= 1
    return u[k:]
def head(u):
    s = script(u[0]); k = 0
    while k < len(u) and script(u[k]) == s: k += 1
    return u[:k]
def jtoks(jp):
    out = []
    for m in JRUN.finditer(re.sub(r'<[^>]+>', '', jp)):
        units = m.group().split('・')
        units[0] = tail(units[0]); units[-1] = head(units[-1])
        out.append('・'.join(units))
    return out
ERE = re.compile(r'(?<![A-Za-z])(?:[A-Z]\.\s?){1,}[A-Z]?\.?(?![a-z])|(?<![A-Za-z])[A-Z]{2,}(?![a-z])')
ESKIP = {'E.G.O', 'E.G.O.', 'EGO', 'LCB', 'LCCB', 'GMT', 'OK', 'HP', 'I', 'A', 'LC', 'T'}
def enorm(e):
    """只差空格、末點的寫法視為同一種：S.H → S.H.、C. I. → C.I.、C.S.. → C.S."""
    t = e.strip()
    if re.fullmatch(r'[A-Z](?:\.\s?[A-Z])*\.*', t) and '.' in t:
        t = re.sub(r'\.\s+', '.', t); t = re.sub(r'\.+', '.', t)
        if not t.endswith('.'): t += '.'
    return t
def etoks(en):
    t = re.sub(r'<[^>]+>', '', en)
    out = []
    for x in ERE.findall(t):
        x = re.sub(r'\s+[A-Z]$', '', x.strip())       # 'S.H. I' → 'S.H.'（後面是代名詞 I）
        if x and x not in ESKIP: out.append(enorm(x))
    return out
ZRE = re.compile(r'[^\s．，。！？…「」（）、<>：；～—]．(?:[^\s．，。！？…「」（）、<>：；～—]．)*[^\s．，。！？…「」（）、<>：；～—]')
def ztoks(zh):
    return ZRE.findall(re.sub(r'<[^>]+>', '', zh))
def letters(e):
    return re.sub(r'[^A-Z]', '', e.split('（')[0])   # 只取大寫字母：D.O.N. and T.E. = D.O.N.T.E.
def syl(a): return ''.join(c for c in a if '가' <= c <= '힣')

byline = collections.OrderedDict()
for n, o in good: byline.setdefault(o['key'], []).append((n, o))
toks = []
for k, l in byline.items():
    l = sorted(l, key=lambda x: x[1]['pos'])            # 依 KR 中的位置排序再配對
    o0 = l[0][1]
    J, E, Z = jtoks(o0['jp']), etoks(o0['en']), ztoks(o0['zh'])
    for idx, (n, o) in enumerate(l):
        pick = lambda L: L[idx] if len(L) == len(l) else None
        t = dict(n=n, key=o['key'], sk=short(o['key']), abbr=o['abbr'], kr=o['abbr'], kind=o['kind'],
                 j=pick(J), e=pick(E), z=pick(Z), auto=True)
        ov = OV.get((t['sk'], o['abbr']))
        if ov:
            t.update(ov); t['auto'] = False
        sy = syl(t['kr'])
        g = MERGE.get(sy, sy)
        if o['kind'] == 'undotted' and sy in BORROWED: g += '*'
        t['g'] = g
        toks.append(t)
pickle.dump(toks, open('tok.pkl', 'wb'))
if __name__ == '__main__':
    miss = [t for t in toks if not (t['j'] and t['e'] and t['z'])]
    print(len(toks), 'occurrences;', len({t['g'] for t in toks}), 'groups; unresolved:', len(miss))
    for t in miss: print('  ', t['n'], t['sk'], t['kr'], t['j'], t['e'], t['z'])
