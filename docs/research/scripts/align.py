import json, os, sys
S = sys.argv[1]
src = {"kr": (f"{S}/raw/kr/LocalizeTemp_kr", "KR_"), "en": (f"{S}/raw/en/LocalizeTemp_en", "EN_"),
       "jp": (f"{S}/raw/jp/LocalizeTemp_jp", "JP_"), "para": (f"{S}/tw/para", ""), "ltm": (f"{S}/tw/ltm", "")}
def index(base, pfx):
    out = {}
    for dp, _, fs in os.walk(base):
        for f in fs:
            if not f.endswith(".json"): continue
            rel = os.path.relpath(os.path.join(dp, f), base); d, n = os.path.split(rel)
            if pfx and n.startswith(pfx): n = n[len(pfx):]
            out[os.path.join(d, n)] = os.path.join(dp, f)
    return out
F = {k: index(*v) for k, v in src.items()}
def load(p):
    try: return json.loads(open(p, "rb").read().decode("utf-8-sig"))
    except Exception: return None
def flat(v, path, out, ctx):
    if isinstance(v, dict):
        c = dict(ctx)
        for k in ("model", "teller", "title", "id"):
            if isinstance(v.get(k), (str, int)): c[k] = v[k]
        for k, x in v.items(): flat(x, path + [k], out, c)
    elif isinstance(v, list):
        for i, x in enumerate(v): flat(x, path + [i], out, ctx)
    elif isinstance(v, str): out[json.dumps(path, ensure_ascii=False)] = (v, ctx)
w = open(f"{S}/units.jsonl", "w", encoding="utf-8")
n = 0
for rel in sorted(F["kr"]):
    d = {}
    for k in src:
        p = F[k].get(rel); data = load(p) if p else None
        out = {}
        if data is not None: flat(data, [], out, {})
        d[k] = out
    for path, (kr, ctx) in d["kr"].items():
        if not kr.strip() or kr.strip() == "-": continue
        g = lambda k: d[k].get(path, (None, None))[0]
        rec = {"file": rel, "path": path, "field": next((x for x in reversed(json.loads(path)) if isinstance(x, str)), ""),
               "kr": kr, "en": g("en"), "jp": g("jp"), "para": g("para"), "ltm": g("ltm"),
               "model": ctx.get("model"), "teller_kr": ctx.get("teller"), "eid": ctx.get("id")}
        w.write(json.dumps(rec, ensure_ascii=False) + "\n"); n += 1
print("units", n)
