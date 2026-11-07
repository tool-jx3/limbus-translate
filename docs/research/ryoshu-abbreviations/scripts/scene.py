import sys; sys.path.insert(0,'.')
import load
rows=load.load()
path=sys.argv[1]; a=int(sys.argv[2]); b=int(sys.argv[3]); langs=sys.argv[4] if len(sys.argv)>4 else 'kz'
for r in rows:
    if r['path']!=path or r['f'] not in ('content','text','dlg'): continue
    i=r['i']
    if i is None or not (a<=i<=b): continue
    m=r['model'] or r['speaker'] or '-'
    s=f"#{i} {m}"
    out=[s]
    if 'k' in langs: out.append(' KR: '+r['kr'].replace('\n','⏎'))
    if 'j' in langs: out.append(' JP: '+r['jp'].replace('\n','⏎'))
    if 'e' in langs: out.append(' EN: '+r['en'].replace('\n','⏎'))
    if 'z' in langs: out.append(' ZH: '+r['zh'].replace('\n','⏎'))
    print('\n'.join(out))
