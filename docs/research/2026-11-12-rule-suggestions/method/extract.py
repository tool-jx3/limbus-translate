import json, sys, os, re, glob, collections
A = sys.argv[1]            # adjclone root
OUT = sys.argv[2]          # output dir
D = A + '/site/public/data/review'
RAW = A + '/.work/raw'
TW = A + '/.work/tw'

# records: key -> origin
origin = {}
for f in glob.glob(TW + '/.meta/units/**/*.jsonl', recursive=True):
    for line in open(f, encoding='utf-8'):
        r = json.loads(line); origin[r['key']] = r['origin']

kr_docs = {}
def kr_doc(file):
    if file not in kr_docs:
        cand = glob.glob(RAW + '/**/KR_' + os.path.basename(file), recursive=True)
        p = None
        for c in cand:
            if c.replace('\\', '/').endswith(('KR_' + os.path.basename(file))) and os.path.dirname(file) in c:
                p = c; break
        if p is None and cand: p = cand[0]
        kr_docs[file] = json.load(open(p, encoding='utf-8-sig')) if p else None
    return kr_docs[file]

def speaker(key):
    file, ptr = key.split('|', 1)
    m = re.match(r'/dataList/(\d+)', ptr)
    d = kr_doc(file)
    if not m or not d: return ''
    i = int(m.group(1)); dl = d.get('dataList') or []
    if i >= len(dl) or not isinstance(dl[i], dict): return ''
    el = dl[i]
    sp = el.get('model') or ''
    m2 = re.match(r'/dataList/\d+/texts/(\d+)', ptr)
    if m2 and isinstance(el.get('texts'), list):
        t = el['texts'][int(m2.group(1))]
        if isinstance(t, dict): sp = t.get('speaker') or sp
    return sp

TAG = re.compile(r'<[^>]*>')
PUNCT = str.maketrans({'，': ',', '。': '.', '！': '!', '？': '?', '：': ':', '；': ';', '（': '(', '）': ')', '「': '"', '」': '"', '『': '"', '』': '"', '“': '"', '”': '"', '‘': "'", '’': "'", '～': '~', '—': '-', '―': '-', '－': '-', '…': '.', '⋯': '.', '・': '.', '．': '.', '　': ' '})
def norm(z):
    z = TAG.sub('', z).translate(PUNCT)
    z = re.sub(r'\.+', '.', z)
    return re.sub(r'\s+', '', z)

s = json.load(open(D + '/suggestions.json', encoding='utf-8'))['items']
groups = []
for x in s:
    vs = []
    for v in x['variants']:
        keys = v['keys']
        vs.append({'zh': v['zh'], 'count': v['count'],
                   'origins': dict(collections.Counter(origin.get(k, '?') for k in keys)),
                   'speakers': dict(collections.Counter(speaker(k) for k in keys[:30])) if x['family'].startswith(('story', 'rpg', 'ally', 'enemy', 'identity', 'mirror', 'announcer')) else {},
                   'keys': keys[:6], 'more_keys': max(0, len(keys) - 6),
                   'jpen': [{'jp': j.get('jp'), 'en': j.get('en'), 'count': j['count']} for j in v.get('jpen', [])][:4]})
    mech = len({norm(v['zh']) for v in x['variants']}) == 1
    groups.append({'id': x['id'], 'family': x['family'], 'field': x['field'], 'kr': x['kr'],
                   'jpen_class': x.get('jpen_class'), 'units': sum(v['count'] for v in x['variants']),
                   'mechanical': mech, 'variants': vs})
n = json.load(open(D + '/notes.json', encoding='utf-8'))['items']
notes = [{'id': x['id'], 'key': x['key'], 'family': x['family'], 'batch': x['batch'], 'kr': x['kr'], 'zh': x['zh'],
          'note': x['note'], 'origin': x['o'], 'speaker': speaker(x['key']) if x['family'].startswith(('story', 'rpg')) else ''}
         for x in n if not x.get('declined')]
json.dump(groups, open(OUT + '/groups.json', 'w', encoding='utf-8'), ensure_ascii=False, indent=0)
json.dump(notes, open(OUT + '/notes.json', 'w', encoding='utf-8'), ensure_ascii=False, indent=0)
print('groups', len(groups), 'mechanical', sum(g['mechanical'] for g in groups), 'notes', len(notes))
print(collections.Counter(g['jpen_class'] for g in groups if not g['mechanical']))
