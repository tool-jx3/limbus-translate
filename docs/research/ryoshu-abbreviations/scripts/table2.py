# 報告第 3 節的表（第 2 輪）：每個縮略語列出 JP／EN／繁中的所有寫法與次數、劇中展開與類別
# 輸入：tok.pkl（tokens.py）、exp.pkl（expand.py）；輸出 table2.md（3.1 出現兩次以上、3.3 只出現一次）
import sys, re, collections, pickle
sys.path.insert(0, '.')
from tokens import letters, short
toks = pickle.load(open('tok.pkl', 'rb'))
res = pickle.load(open('exp.pkl', 'rb'))
import load
byk = {r['key']: r for r in load.load()}

def cu(sk):
    """單位的短寫：E512B#1、Announcer#26、Voice_Butler#21、RPG:floor-b2#22/9"""
    s = sk.replace('|', '')
    s = s.replace('Announcer_Ryoshu_42', 'Announcer').replace('Voice_EGO_Ryoshu_', 'EGO_')
    s = re.sub(r'Voice_Ryoshu_(\w+?)_\d{5}', r'Voice_\1', s)
    return s
def jn(j): return j.replace('（不加點）', '').replace('・', '')   # 只差「・」的（KR 不加點）算同一種
def cell(x): return str(x).replace('|', '｜').replace('\n', ' ')

# 類別（人工讀過場景後判定）。沒列的：各欄都一致 → 一致
# 值：(類別, 中文不一, 提案編號)
CLS = {
 '모분': ('同義異寫', True, '10'), '시대': ('', True, '1'), '알못': ('同義異寫', False, ''),
 '거거': ('同義異寫', False, ''), '그그': ('異義', True, '34'), '노잼': ('同義異寫', False, ''),
 '관없': ('同義異寫', True, '2、3'), '어떠': ('同義異寫', True, '8'), '쓸없': ('同義異寫', True, '35'),
 '재없': ('同義異寫', True, '5、30'), '자정': ('同義異寫', False, ''), '황가': ('異義', False, '37'),
 '모불아위': ('異義', False, ''), '내머': ('同義異寫', True, '33'), '무상': ('同義異寫', True, '32'),
 '맘들않': ('同義異寫', True, '28'), '버안': ('同義異寫', True, '31'), '상없': ('同義異寫', True, '4'),
 '믿없': ('同義異寫', True, '27'), '엔팔': ('同義異寫', True, '9'), '말안어': ('異義', False, ''),
 '어이': ('異義', False, '21'), '알바': ('同義異寫', False, ''), '아닥': ('同義異寫', False, ''),
 '동감': ('同義異寫', True, '6、7'), '삼대': ('', True, '36'),
}
# 只差點、空格或末點的 EN 寫法算同一種（SYNC = S.Y.N.C.）；說到一半被打斷的算原形
ALIAS = {'SYN': 'SYNC'}
# 劇中展開（出現兩次以上的，人工核對；含自動找不到的）
EXP = {
 '모분': '모가지를 분질러버리면（싱클레어，ES130B#10）；S210B#5、P10413#19 同',
 '시대': '시계 대가리（히스클리프，RPG:common-a1c10p1#50/0）',
 '거거': '거기서 거기（료슈2，E903A#7；료슈，E507A#45）',
 '그그': '그게 그거라니요（싱클레어，E302B#33）；그게 그거（키라，E903A#8）。Announcer#19 無展開，JP 案・定、EN knew it：그럼 그렇지',
 '성업': '성능 업그레이드라니（싱클레어，E064X#15）',
 '자정': '자색 정오를（이상，P10410#44）；#55 이상 把 #54 讀成 자정（午夜）',
 '황가': '황금가지라도（뫼르소，S1013B#11）→ 良秀：황금가지가 아니라 황금가죽（S1013B#12）。E914B#88 無展開，JP 黄・枝：황금가지',
 '모불아위': '同句括號：…아름다움을 위해（#0）／…아라야를 위해（#1）',
 '내머': '내려라, 머저리들（同句，Announcer#0）；내리자, 머저리들（료슈，S201B#80）',
 '버안': '버스 안이에요（홍루，E512B#2）。Announcer#26 無展開，JP バ・中、EN O.T.B.（on the bus）',
 '알단이예': '알겠나 단테? 이것이 예술이다（同句括號）',
 '검만예극': '검으로 만들어내는 예술의 극치（旁白，P10402#13）',
 '모분조도': '모가지를 분질러야 조용해지지 도야지（료슈，S101B#46；Announcer#28 同句）',
 '나시': '나갈 시간이다（旁白，RPG:common-a1c10p1#18/2）',
 '실분': '실로 분해되는（뫼르소，RPG:floor-b2#51/2）',
 '좁형': '좁은 형님이（싱클레어，S532B#10）',
 '말안어': '말 안 듣는 어른（싱클레어，ES130B#10）／말 안듣는 어린아이（同，#11）',
 '아닥': '아가리를 닥치고（호엔하임，E914B#38）',
 '대박': '대가리가 박살（旁白，S504B#111）',
 '머절': '머리를 절단해（료슈2，E902B#71）',
 '사분': '사지를 분질러（旁白，P10404#10）',
 '알바': '알 바는（탐정로쟈，E614A#12）',
 '재없': '재미없는 대화라（싱클레어，E724B#75；良秀 #78 認可）',
 '삼대': '삼조씨 대답하시래요（싱클레어，S430B#59）→ 대가리를 깨버린다고（同，#62）→ 良秀：지금 대가리를 깬다고 했다（#63）',
 '쓸없': '—（RPG 兩處是 `쓸.없.` 後接道具名 파하하，繁中寫成 無．用．噗哈哈）',
 '큰형': '—（同一行兩次；EN 合成一個 G.B.）',
 '단테': '—（後句「시계… 너를 부른 게 아니다」）',
 '어이': '—（5D205B#3 對魁魁格；S445B#0 叫李箱，李箱答「부른 거요?」，JP お、イ、EN E.Y. 都讀成李箱）',
}
# 只出現一次的：沿用初版人工改過的展開
SEXP = {'감다뒤*': '감이… 다… 뒤졌다아（阿賴耶，`S940B｜#34`）',
        '안심안지*': '안 심심하고 안 지루…（싱클레어，`S951B｜#25`）',
        '완소*': '—（阿賴耶的流行語，S947B#224）'}

def sk(s): return s.replace('Announcer_Ryoshu_42', 'Announcer_Ryoshu').replace('Voice_Ryoshu_', 'Voice_').replace('|', '｜')
def forms(l, field, varied):
    """依次數列出寫法；欄內有變化時，次數 ≤ 3 的寫法附上單位"""
    if field == 'e':
        grp = collections.OrderedDict()
        for t in sorted(l, key=lambda t: t['e']):
            k = letters(t['e']); k = ALIAS.get(k, k)
            grp.setdefault(k, []).append(t)
        items = []
        for k, ts in grp.items():
            c = collections.Counter(t['e'] for t in ts)
            main = max(c, key=lambda f: (c[f], -f.count(' '), '.' in f))
            alt = '、'.join(f'{f}×{n}' if n > 1 else f for f, n in c.items() if f != main)
            items.append((main, ts, f'（含 {alt}）' if alt else ''))
    else:
        grp = collections.OrderedDict()
        for t in l: grp.setdefault(t[field], []).append(t)
        items = [(f, ts, '') for f, ts in grp.items()]
    items.sort(key=lambda x: -len(x[1]))
    out = []
    for f, ts, alt in items:
        s = f + (f'×{len(ts)}' if len(ts) > 1 else '') + alt
        if varied and len(ts) <= 3:
            s += '（' + '、'.join(cu(t['sk']) for t in ts) + '）'
        out.append(s)
    return '、'.join(out)
def krcol(l):
    c = collections.Counter(t['kr'] for t in l)
    rep = c.most_common(1)[0][0]
    key = lambda a: re.sub(r'\s', '', a).rstrip('.')
    alts = collections.Counter()
    for a, n in c.items():
        if key(a) != key(rep): alts[key(a) + ('.' if a.endswith('.') else '')] += n
    s = rep
    if alts: s += '（另 ' + '、'.join(f'`{a}`' + (f'×{n}' if n > 1 else '') for a, n in alts.items()) + '）'
    return s

G = collections.OrderedDict()
for t in toks: G.setdefault(t['g'], []).append(t)
rec = sorted([(g, l) for g, l in G.items() if len(l) >= 2], key=lambda x: (-len(x[1]), x[0]))
one = sorted([(g, l) for g, l in G.items() if len(l) == 1], key=lambda x: x[0])
summary = collections.Counter(); nflag = 0
L = ['| KR | 次數 | JP | EN | 既有繁中 | 劇中展開 | 類別 | 出處 |', '|---|---|---|---|---|---|---|---|']
for g, l in rec:
    vj = len({jn(t['j']) for t in l}) > 1
    ve = len({ALIAS.get(letters(t['e']), letters(t['e'])) for t in l}) > 1
    vz = len({t['z'] for t in l}) > 1
    cls, zf, prop = CLS.get(g, ('一致', False, ''))
    if g not in CLS:
        assert not (vj or ve or vz), ('needs a class', g)
    if zf: assert vz, g
    lab = '＋'.join(x for x in (cls, '中文不一' if zf else '') if x)
    summary[cls or '中文不一'] += 1; nflag += zf
    if prop: lab += f'（提案 {prop}）'
    keys = [sk(t['sk']) for t in l]
    src = '、'.join(f'`{k}`' for k in keys[:3]) + (f' 等 {len(keys)} 處' if len(keys) > 3 else '')
    L.append('| ' + ' | '.join(cell(x) for x in (krcol(l), len(l), forms(l, 'j', vj), forms(l, 'e', ve),
                                                   forms(l, 'z', vz), EXP.get(g, '—'), lab, src)) + ' |')
S = ['| KR | JP | EN | 既有繁中 | 劇中展開 | 出處 |', '|---|---|---|---|---|---|']
for g, l in one:
    t = l[0]
    exp = SEXP.get(g)
    if exp is None:
        exp = '—'
        if t['n'] in res:
            x = res[t['n']][0]; r = byk[x[1]]
            spk = '同句' if x[0] == 'same' else (r['model'] or r['speaker'] or '旁白')
            exp = f"{x[2]}（{spk}，`{sk(short(x[1]))}`）"
    S.append('| ' + ' | '.join(cell(x) for x in (t['kr'], t['j'], t['e'], t['z'], exp, f"`{sk(t['sk'])}`")) + ' |')
summary['一致'] += len(one)
with open('table2.md', 'w') as f:
    f.write('\n'.join(L) + '\n\n' + '\n'.join(S) + '\n')
print('groups', len(G), 'recurring', len(rec), sum(len(l) for g, l in rec), 'single', len(one))
print('class', dict(summary), 'zh-flags', nflag)
