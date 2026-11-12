import json, glob, sys, collections, os
RV = os.path.dirname(os.path.abspath(__file__))
groups = {g['id']: g for g in json.load(open(RV + '/groups.json', encoding='utf-8'))}
notes = {n['id']: n for n in json.load(open(RV + '/notes.json', encoding='utf-8'))}
GC = {'I1','I2','I3','I4','I5','I6','I7','I9'}; GA = {'keep','unify','fix','glossary','guide','decide'}
NC = {'N%d' % i for i in range(1,10)}; NA = {'none','glossary','fix','guide','decide','source'}
probs = collections.Counter()
for f in sorted(glob.glob(RV + '/chunks/*.json')):
    name = os.path.basename(f)
    inp = json.load(open(f, encoding='utf-8'))
    out = json.load(open(RV + '/out/' + name, encoding='utf-8'))
    ids_in = [x['id'] for x in inp]; ids_out = [x['id'] for x in out]
    p = []
    if ids_in != ids_out:
        p.append('id order/count mismatch %d vs %d, missing %d, extra %d' % (len(ids_in), len(ids_out), len(set(ids_in)-set(ids_out)), len(set(ids_out)-set(ids_in))))
    for r in out:
        if name.startswith('g'):
            g = groups.get(r['id'])
            if not g: p.append('unknown id ' + r['id']); continue
            if r.get('category') not in GC: p.append('bad cat %s %s' % (r['id'], r.get('category')))
            a = r.get('action')
            if a not in GA: p.append('bad act %s %s' % (r['id'], a))
            zhs = {v['zh'] for v in g['variants']}
            for c in r.get('changes') or []:
                if c.get('from') not in zhs: p.append('from not variant %s: %r' % (r['id'][-8:], (c.get('from') or '')[:40]))
                if c.get('from') == c.get('to'): p.append('noop change %s' % r['id'][-8:])
            if a in ('unify','fix','glossary','guide') and not r.get('changes'): p.append('no changes for %s %s' % (a, r['id'][-8:]))
            if a == 'unify':
                t = r.get('target')
                if t is None: p.append('unify no target %s' % r['id'][-8:])
                else:
                    tos = {c['to'] for c in r['changes']}
                    if tos != {t}: p.append('unify to!=target %s' % r['id'][-8:])
                    if t not in zhs: probs['target not existing variant'] += 1
                    # every variant except target should be changed
                    changed = {c['from'] for c in r['changes']}
                    left = zhs - changed - {t}
                    if left: p.append('unify leaves variants %s n=%d' % (r['id'][-8:], len(left)))
            if a == 'decide' and not r.get('options'): p.append('decide no options %s' % r['id'][-8:])
            if a == 'glossary' and not r.get('glossary'): p.append('glossary missing %s' % r['id'][-8:])
            if a == 'guide' and not r.get('guide'): p.append('guide missing %s' % r['id'][-8:])
        else:
            n = notes.get(r['id'])
            if not n: p.append('unknown id ' + r['id']); continue
            if r.get('category') not in NC: p.append('bad cat %s %s' % (r['id'], r.get('category')))
            a = r.get('action')
            if a not in NA: p.append('bad act %s %s' % (r['id'], a))
            if a == 'fix' and not r.get('changes'): p.append('fix no changes %s' % r['id'])
            if a == 'decide' and not r.get('options'): p.append('decide no options %s' % r['id'])
            if a == 'glossary' and not r.get('glossary'): p.append('glossary missing %s' % r['id'])
            if a == 'guide' and not r.get('guide'): p.append('guide missing %s' % r['id'])
            for c in r.get('changes') or []:
                if not c.get('key') or c.get('from') is None or c.get('to') is None: p.append('bad change %s' % r['id'])
                elif c['from'] == c['to']: p.append('noop change %s' % r['id'])
        if r.get('confidence') not in ('high','medium','low'): p.append('bad conf %s' % r['id'][-12:])
    acts = collections.Counter(r.get('action') for r in out)
    print(name, len(out), dict(acts), ('PROBLEMS %d' % len(p)) if p else 'ok')
    for x in p[:8]: print('   ', x)
print(dict(probs))
