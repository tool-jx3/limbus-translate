import json, os, glob, re, pickle
# The site data written by `lct site --out DIR` (DIR/files/<path>.json); the default is the repo's site/public/data.
D=os.path.join(os.environ.get('LCT_SITE_DATA', 'site/public/data'), 'files')
CACHE='rows.pkl'  # in the working directory
def load():
    if os.path.exists(CACHE):
        return pickle.load(open(CACHE,'rb'))
    rows=[]
    for f in glob.glob(D+'/**/*.json', recursive=True):
        d=json.load(open(f,encoding='utf-8'))
        path=d['path']; units=d['units']
        # context: story model/teller by i; rpg speaker by prefix
        ctx={}
        for u in units:
            p=u['p']
            if u['f'] in ('model','teller','title','speaker'):
                base=p.rsplit('/',1)[0]
                ctx.setdefault(base,{})[u['f']]=u.get('kr')
                if u['f']=='teller': ctx[base]['teller_zh']=u.get('zh')
        for u in units:
            base=u['p'].rsplit('/',1)[0]
            c=ctx.get(base,{})
            rows.append(dict(key=path+'|'+u['p'], path=path, p=u['p'], f=u['f'], i=u.get('i'),
                kr=u.get('kr') or '', jp=u.get('jp') or '', en=u.get('en') or '', zh=u.get('zh') or '', o=u.get('o'),
                model=c.get('model'), teller=c.get('teller'), teller_zh=c.get('teller_zh'), speaker=c.get('speaker'), family=d.get('family')))
    pickle.dump(rows, open(CACHE,'wb'))
    return rows
