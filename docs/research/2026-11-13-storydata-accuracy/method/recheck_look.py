"""usage: python3 -I look.py <repo> <file base, e.g. S601B> <start> <end>
Prints KR/JP/EN/TW (content, plus place/teller/title when present) for lines start..end of a story file, from the current raw/tw worktrees."""
import json, sys
repo, base, a, b = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
d = {l: json.load(open('%s/.work/raw/%s/StoryData/%s_%s.json' % (repo, l, l.upper(), base), encoding='utf-8-sig'))['dataList'] for l in ('kr', 'jp', 'en')}
d['tw'] = json.load(open('%s/.work/tw/LimbusCompany_Data/Lang/zh-TW/StoryData/%s.json' % (repo, base), encoding='utf-8-sig'))['dataList']
s = lambda x: (x or '').replace('\n', '⏎')
for i in range(max(0, a), min(len(d['kr']), b + 1)):
    k = d['kr'][i]
    print('%s#%d [%s]%s' % (base, i, k.get('teller') or k.get('model') or '', (' 〔地點 KR:%s TW:%s〕' % (s(k.get('place')), s(d['tw'][i].get('place')))) if k.get('place') else ''))
    for l in ('kr', 'jp', 'en', 'tw'):
        print('   %s: %s' % (l.upper(), s(d[l][i].get('content'))))
