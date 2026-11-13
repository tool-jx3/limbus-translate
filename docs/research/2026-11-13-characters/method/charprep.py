"""Per-character material: every StoryData line a character speaks, with context and stats.

usage: python3 -I charprep.py <sd dir>
writes <sd>/chars/<id>.lines.tsv, <id>.stats.md
"""
import json, collections, re, os, sys
SD = sys.argv[1]
units = json.load(open(SD + '/units.json')); meta = json.load(open(SD + '/meta.json'))
names, guides = meta['names'], meta['guides']
EXTRA = {  # variants found in StoryData but not in the guide's models (to be judged by the agent)
    'heathcliff': ['후디히스', '뿅망히스', '곰스클리프'],
    'sinclair': ['기사싱클', '기사싱클2', '퀴즈싱클'],
    'gregor': ['말레고르'],
    'donquixote': ['키호테', '가면키호테', '기사돈키', '키호테빛'],
}
# unguided characters: group of model codes per character (by display name), >= 60 lines
UNGUIDED = collections.OrderedDict([
    ('gasichun', ['가시춘']), ('hohenheim', ['호엔하임', '호엔하임2', '뒤틀린호엔하임']), ('donglang', ['동랑']),
    ('index-yuen', ['검지아비']), ('gachiu', ['가치우']), ('kimsatgat', ['뉴김삿갓']), ('thumb-valencina', ['엄지아비']),
    ('nelly', ['넬리']), ('ezra', ['에즈라']), ('ahab', ['에이해브']), ('samson', ['산손']), ('kira', ['뉴키라', '중지제자']),
    ('sancho', ['산초', '옛산초', '옛돈키']), ('lin-daiyu', ['임대옥']), ('amiya', ['아미야']), ('carmen', ['카르멘', '카르멘진짜']),
    ('jeanne', ['잔느', '잔느변신2']), ('xue-baochai', ['설보차']), ('catherine', ['캐서린']), ('jiamu', ['과거가모', '가모']),
    ('zigong', ['자공']), ('sord', ['뉴소드', '소드']), ('yuri', ['유리']), ('jia-yuanchun', ['가원춘']), ('samjo', ['삼조']),
    ('aengdu', ['뉴앵두', '앵두']), ('ring-callisto', ['약지아비']), ('pinky-jihyeseong', ['소지아비', '소지아비2']),
    ('t-corp-rep', ['T사대표']), ('chwisu', ['취수야객']), ('wei', ['웨이']), ('linton', ['린튼']), ('jia-huan', ['에고가환']),
    ('meyers', ['메이어스', '주교메이어스']), ('irene', ['아이린']), ('sonya', ['소냐']), ('noehoeng', ['뇌횡']),
    ('hindley', ['힌들리']), ('alisa', ['알리사']), ('herman', ['헤르만']), ('wang-zhao', ['왕자우']), ('zilu', ['자로사복']),
    ('queequeg', ['퀴케그']), ('mc-grandview', ['사회자']), ('bari', ['바리']), ('kyungdo', ['경도']),
    ('middle-matthias', ['중지아비']), ('kaltsit', ['켈시']), ('dongbaek-past', ['과거동백']), ('cesara', ['체사라']),
    ('demian', ['데미안']), ('xue-pan', ['설반']), ('moses', ['모제스2', '모제스4']), ('allen', ['앨런']), ('doctor', ['박사']),
    ('asia', ['아세아']), ('kromer-sister', ['크로머언니']), ('marten', ['마튼']), ('time-killer', ['시간살인마']),
    ('si-yihua', ['사이화']), ('vespa', ['베스파']), ('deadrabbits-boss', ['데드레빗츠보스']), ('effie', ['에피']),
    ('pilot', ['파일럿']), ('sangi', ['상이']),
])
# index lines by file for context
by_file = collections.defaultdict(list)
for u in units:
    if u['f'] == 'content':
        by_file[u['file']].append(u)
pos = {}
for f, us in by_file.items():
    for j, u in enumerate(us):
        pos[u['key']] = (f, j)

END = [('합쇼체', r'(습니다|ㅂ니다|니다|습니까|니까|십시오|시오|읍시다|십시다)$'),
       ('하오체', r'(하오|이오|구려|소|시오|겠소|었소|았소|오|게나)$'),
       ('해요체', r'요$'),
       ('반말', r'.')]
def strip(k):
    k = re.sub(r'<[^>]*>', '', k)
    k = re.sub(r'[\s\.…!?~,♪♡’‘"”“\'\-—)(\]\[]+$', '', k)
    return k
def ending(k):
    s = strip(k)
    if not s or not re.search(r'[가-힣]$', s):
        return '其他'
    for name, pat in END:
        if re.search(pat, s):
            return name
SELF = {'나/내': r'(^|[^가-힣])(나는|나도|나를|나의|나한테|나에게|나랑|나와|내가|내|난|날)(?=[^가-힣]|$)',
        '저/제': r'(^|[^가-힣])(저는|저도|저를|저의|저한테|저에게|저랑|제가|제|전|절)(?=[^가-힣]|$)',
        '본인': r'본인', '소인': r'소인', '이 몸/이몸': r'이 ?몸', '우리': r'우리', '저희': r'저희', '짐': r'(^|[^가-힣])짐(이|은|을|의)'}
SECOND = {'너/네': r'(^|[^가-힣])(너는|너도|너를|너의|너한테|너에게|너랑|너와|네가|니가|넌|널|너)(?=[^가-힣]|$)',
          '당신': r'당신', '자네': r'자네', '그대': r'그대', '너희': r'너희', '여러분': r'여러분', '귀하': r'귀하', '댁': r'(^|[^가-힣])댁'}
TWSELF = {'我': '我', '本小姐': '本小姐', '本大爺': '本大爺', '在下': '在下', '吾': '吾', '俺': '俺', '人家': '人家', '咱': '咱', '小的': '小的', '老子': '老子', '本人': '本人'}
TWSECOND = {'你': '你', '妳': '妳', '您': '您', '汝': '汝', '閣下': '閣下', '你們': '你們', '各位': '各位', '爾': '爾'}
def has(p, s):
    return re.search(p, s) is not None

def material(cid, models, extra, label, guide_path):
    rows = []
    for u in units:
        if u['f'] == 'content' and u['model'] in models + extra:
            rows.append(u)
    if not rows:
        return 0
    os.makedirs(SD + '/chars', exist_ok=True)
    with open(SD + '/chars/%s.lines.tsv' % cid, 'w', encoding='utf-8') as w:
        w.write('ref\tmodel\tprev_speaker\tprev_kr\tkr\ttw\ten\n')
        for u in rows:
            f, j = pos[u['key']]
            prev = by_file[f][j - 1] if j > 0 else None
            nxt = by_file[f][j + 1] if j + 1 < len(by_file[f]) else None
            ref = '%s#%d' % (f.split('/')[-1][:-5], u['i'])
            cl = lambda s: (s or '').replace('\t', ' ').replace('\n', '⏎')
            w.write('\t'.join([ref, u['model'], cl(prev['sp_kr']) if prev else '', cl(prev['kr'])[:60] if prev else '',
                               cl(u['kr']), cl(u['zh']), cl(u['en'])]) + '\n')
    st = ['# %s 的 StoryData 統計' % label, '',
          '- 行數：%d（model：%s）' % (len(rows), '、'.join('%s %d' % (m, n) for m, n in collections.Counter(u['model'] for u in rows).most_common())),
          '- 檔案數：%d' % len({u['file'] for u in rows}),
          '- 單位來源：%s' % '、'.join('%s %d' % (o, n) for o, n in collections.Counter(u['o'] for u in rows).most_common()),
          '- 角色指引：%s' % (guide_path or '（沒有）'), '']
    st.append('## KR 句尾語體（只看每行最後一句的句尾，粗略分類）')
    st.append('')
    ec = collections.Counter(ending(u['kr']) for u in rows)
    tot = sum(ec.values())
    for k, n in ec.most_common():
        st.append('- %s：%d（%.0f%%）' % (k, n, 100 * n / tot))
    st.append('')
    def cross(title, krd, twd):
        st.append('## ' + title)
        st.append('')
        st.append('| KR \\ TW | 行數 | ' + ' | '.join(twd) + ' |')
        st.append('|---|---:|' + '---:|' * len(twd))
        for kname, kp in krd.items():
            ls = [u for u in rows if has(kp, u['kr'])]
            if not ls:
                continue
            st.append('| %s | %d | ' % (kname, len(ls)) + ' | '.join(str(sum(1 for u in ls if tp in u['zh'])) for tp in twd.values()) + ' |')
        st.append('')
    cross('自稱（KR 有該詞的行，TW 出現各寫法的行數）', SELF, TWSELF)
    cross('第二人稱（同上）', SECOND, TWSECOND)
    # name self-reference
    nm = names.get(models[0], [None])[0] if models else None
    if nm:
        ls = [u for u in rows if re.search(re.escape(nm) + r'(는|가|는요|도|를|의|한테)', u['kr'])]
        st.append('- KR 用自己的名字「%s」+ 助詞的行：%d' % (nm, len(ls)))
        st.append('')
    # address terms: X 씨 / 님 / 양반 / 군 / 양 / 공
    st.append('## 稱呼他人（KR「名詞＋씨／님／양반／선생／아저씨／언니／오빠／누나／형」，前 40 名；군、양 太雜不列）')
    st.append('')
    ac = collections.Counter()
    for u in rows:
        for m in re.finditer(r'([가-힣A-Za-z]{1,8}) ?(씨|님|양반|선생님|선생|아저씨|언니|오빠|누나|형님|형)(?=[^가-힣]|$|[이가은는을를의께에한랑과도요])', u['kr']):
            ac[m.group(1) + ' ' + m.group(2)] += 1
    st.append('、'.join('%s %d' % (k, n) for k, n in ac.most_common(40)))
    st.append('')
    # previous speakers (who they answer)
    pc = collections.Counter()
    for u in rows:
        f, j = pos[u['key']]
        if j > 0:
            pc[by_file[f][j - 1]['sp_kr'] or '（旁白）'] += 1
    st.append('## 前一行的說話者（大致是對話對象，前 25 名）')
    st.append('')
    st.append('、'.join('%s %d' % (k, n) for k, n in pc.most_common(25)))
    st.append('')
    # TW sentence-final particles
    st.append('## TW 句末語氣詞（每行最後一個字）')
    st.append('')
    fc = collections.Counter()
    for u in rows:
        z = re.sub(r'<[^>]*>', '', u['zh'])
        z = re.sub(r'[\s。！？…，、「」『』\.!?~～—\-）)]+$', '', z)
        if z:
            fc[z[-1]] += 1
    st.append('、'.join('%s %d' % (k, n) for k, n in fc.most_common(30)))
    st.append('')
    # frequent KR phrases (2-3 eojeol n-grams)
    st.append('## KR 常見片語（2～3 個語節，出現 ≥5 次，前 40 名）')
    st.append('')
    ng = collections.Counter()
    for u in rows:
        ws = re.findall(r'[가-힣]+', u['kr'])
        for n in (2, 3):
            for i in range(len(ws) - n + 1):
                ng[' '.join(ws[i:i + n])] += 1
    st.append('、'.join(['%s %d' % (k, c) for k, c in ng.most_common(80) if c >= 5][:40]))
    st.append('')
    open(SD + '/chars/%s.stats.md' % cid, 'w', encoding='utf-8').write('\n'.join(st) + '\n')
    return len(rows)

out = {}
for gid, g in guides.items():
    n = material(gid, g['models'] + g['other'], EXTRA.get(gid, []), '%s（%s）' % (g['name'], gid), 'guides/characters/%s.md' % gid)
    out[gid] = n
for cid, ms in UNGUIDED.items():
    out[cid] = material(cid, ms, [], cid, None)
json.dump(out, open(SD + '/chars/index.json', 'w'), ensure_ascii=False, indent=1)
print(out)
