#!/usr/bin/env python3
"""把譯文與 KR 逐欄位比對，找出報告 04-other.md 列的結構問題。

  python3 -I packcheck.py <KR資料夾> <譯文資料夾> [每類列出幾個例子]

  例：python3 -I packcheck.py .work/raw/kr .work/tw/LimbusCompany_Data/Lang/zh-TW 5

檢查項目：檔案與 id 是否齊全、欄位集合、非字串的值、{n} 佔位符的集合、格式字串是否合法、
劇情檔的列數與 id 順序與 model、teller／title 是否為空、content 的 //、選擇事件的 \n\n 頁數、
但丁筆記的條列符號、敵人名稱的半形 ? 個數、欄位是否由空變非空。
輸出是「類別  數量」加上幾個例子；kr_ 開頭的類別是 KR 自己的狀況，供對照。
"""
import collections
import json
import os
import re
import sys

if len(sys.argv) < 3:
    sys.exit(__doc__)
KR, TW = sys.argv[1], sys.argv[2]
SHOW = int(sys.argv[3]) if len(sys.argv) > 3 else 3


def load(p):
    try:
        return json.load(open(p, encoding='utf-8-sig'))
    except Exception as e:
        return None

def pairs():
    for root, _, files in os.walk(KR):
        for f in files:
            if not f.endswith('.json') or not f.startswith('KR_'):
                continue
            rel = os.path.relpath(os.path.join(root, f), KR)
            twp = os.path.join(TW, os.path.dirname(rel), f[3:])
            yield rel, os.path.join(root, f), twp

def walk(o, path=()):
    if isinstance(o, dict):
        for k, v in o.items():
            yield from walk(v, path + (k,))
    elif isinstance(o, list):
        for i, v in enumerate(o):
            yield from walk(v, path + (i,))
    else:
        yield path, o

PH = re.compile(r'\{(\d+)(?:[,:][^{}]*)?\}')
def braces_ok(s):
    # .NET composite format validity: every { must start {{ or a valid item, every } must be }} or close an item
    i = 0; n = len(s)
    while i < n:
        c = s[i]
        if c == '{':
            if i + 1 < n and s[i+1] == '{': i += 2; continue
            m = re.compile(r'\{\s*\d+\s*(,\s*-?\d+\s*)?(:[^{}]*)?\}').match(s, i)
            if not m: return False
            i = m.end(); continue
        if c == '}':
            if i + 1 < n and s[i+1] == '}': i += 2; continue
            return False
        i += 1
    return True

stat = collections.Counter()
ex = collections.defaultdict(list)
def note(k, msg):
    stat[k] += 1
    if len(ex[k]) < 6: ex[k].append(msg)

def fname_class(rel):
    b = os.path.basename(rel)[3:]
    return re.sub(r'[-_].*|\.json$', '', b)

for rel, kp, tp in pairs():
    k = load(kp)
    if k is None: note('kr_unparsable', rel); continue
    if not os.path.exists(tp): note('tw_missing_file', rel); continue
    t = load(tp)
    if t is None: note('tw_unparsable', rel); continue
    kl = k.get('dataList') if isinstance(k, dict) else None
    tl = t.get('dataList') if isinstance(t, dict) else None
    if not isinstance(kl, list) or not isinstance(tl, list): note('no_dataList', rel); continue
    story = rel.startswith('StoryData')
    cls = fname_class(rel)
    if story:
        stat['story_files'] += 1
        if len(kl) != len(tl): note('story_len_mismatch', '%s kr=%d tw=%d' % (rel, len(kl), len(tl)))
        for i, (a, b) in enumerate(zip(kl, tl)):
            if a.get('id') != b.get('id'): note('story_id_seq_mismatch', '%s idx %d kr=%r tw=%r' % (rel, i, a.get('id'), b.get('id'))); break
        for i, (a, b) in enumerate(zip(kl, tl)):
            for key in ('model',):
                if a.get(key) != b.get(key): note('story_model_changed', '%s idx %d %r -> %r' % (rel, i, a.get(key), b.get(key)))
            for key in ('teller', 'title', 'place', 'content'):
                if bool(a.get(key)) != bool(b.get(key)): note('story_%s_emptiness_changed' % key, '%s idx %d %r -> %r' % (rel, i, a.get(key), b.get(key)))
            if '//' in (a.get('content') or ''): note('story_kr_content_has_//', '%s idx %d %r' % (rel, i, a.get('content')[:60]))
            if '//' in (b.get('content') or ''): note('story_tw_content_has_//', '%s idx %d %r' % (rel, i, b.get('content')[:80]))
            if not a.get('model') and a.get('teller') and not a.get('title'): note('story_kr_teller_without_model_and_title', '%s idx %d %r' % (rel, i, a.get('teller')))
            if not b.get('model') and b.get('teller') and not b.get('title'): note('story_tw_teller_without_model_and_title', '%s idx %d %r' % (rel, i, b.get('teller')))
            ks = set(a.keys()); ts = set(b.keys())
            if ks != ts: note('story_keyset_diff', '%s idx %d kr-only=%s tw-only=%s' % (rel, i, sorted(ks - ts), sorted(ts - ks)))
        continue
    kd = {str(x.get('id')): x for x in kl if isinstance(x, dict)}
    td = {}
    for x in tl:
        if not isinstance(x, dict): continue
        i = str(x.get('id'))
        if i in td: note('tw_duplicate_id', '%s id=%s' % (rel, i))
        td[i] = x
    if len(kd) != len(kl): note('kr_duplicate_id', '%s %d dup' % (rel, len(kl) - len(kd)))
    for i in kd:
        if i not in td: note('tw_missing_id', '%s id=%s' % (rel, i))
    for i in td:
        if i not in kd: note('tw_extra_id', '%s id=%s' % (rel, i))
    for i, a in kd.items():
        b = td.get(i)
        if b is None: continue
        if type(a.get('id')) != type(b.get('id')): note('id_json_type_changed', '%s id=%r -> %r' % (rel, a.get('id'), b.get('id')))
        ka = dict(walk(a)); tb = dict(walk(b))
        for p, v in ka.items():
            if p not in tb:
                note('tw_missing_path', '%s id=%s %s' % (rel, i, p)); continue
            w = tb[p]
            if isinstance(v, str) and isinstance(w, str):
                stat['strings'] += 1
                kph = sorted(set(PH.findall(v))); tph = sorted(set(PH.findall(w)))
                if kph: stat['strings_with_placeholders'] += 1
                if kph != tph:
                    extra = [x for x in tph if x not in kph]
                    if extra: note('placeholder_extra_in_tw(%s)' % cls, '%s id=%s %s kr=%s tw=%s' % (rel, i, p, kph, tph))
                    else: note('placeholder_missing_in_tw(%s)' % cls, '%s id=%s %s kr=%s tw=%s' % (rel, i, p, kph, tph))
                if not braces_ok(v): note('kr_invalid_format_string(%s)' % cls, '%s id=%s %s %r' % (rel, i, p, v[:70]))
                if not braces_ok(w): note('tw_invalid_format_string(%s)' % cls, '%s id=%s %s %r' % (rel, i, p, w[:70]))
                if braces_ok(v) and not braces_ok(w): note('tw_invalid_format_string_where_kr_valid', '%s id=%s %s %r' % (rel, i, p, w[:70]))
                # line/paragraph structure
                if v.count('\n\n') != w.count('\n\n') and cls in ('AbEvents', 'ActionEvents'):
                    note('choice_event_paragraph_count_diff(%s)' % cls, '%s id=%s %s kr=%d tw=%d' % (rel, i, p, v.count('\n\n') + 1, w.count('\n\n') + 1))
                for tok in ('<style="highlight">', '</style>', '\\n', '<br>', '\r'):
                    if v.count(tok) != w.count(tok): note('token_count_diff %r' % tok, '%s id=%s %s kr=%d tw=%d' % (rel, i, p, v.count(tok), w.count(tok)))
                if cls.startswith('StoryTheaterDanteNote') and v.count('\n-') != w.count('\n-'):
                    note('dantenote_bullet_count_diff', '%s id=%s %s kr=%d tw=%d' % (rel, i, p, v.count('\n-'), w.count('\n-')))
                if cls == 'Enemies' and p == ('name',) and v.count('?') != w.count('?'):
                    note('enemy_name_halfwidth_qmark_count_diff', '%s id=%s %r -> %r' % (rel, i, v, w))
                if (v == '') != (w == ''): note('emptiness_changed(%s)' % cls, '%s id=%s %s %r -> %r' % (rel, i, p, v[:30], w[:30]))
                if bool(re.search(r'\[[^\]\[]*\]', v)) and sorted(re.findall(r'\[[^\]\[]*\]', v)) != sorted(re.findall(r'\[[^\]\[]*\]', w)):
                    stat['bracket_token_multiset_diff(any)'] += 1
            else:
                if v != w or type(v) != type(w): note('nonstring_value_changed(%s)' % cls, '%s id=%s %s %r -> %r' % (rel, i, p, v, str(w)[:40]))
        for p in tb:
            if p not in ka: note('tw_extra_path(%s)' % cls, '%s id=%s %s' % (rel, i, p))

for k in sorted(stat):
    print('%-60s %d' % (k, stat[k]))
    for m in ex.get(k, [])[:SHOW]:
        print('      ', m[:230])
