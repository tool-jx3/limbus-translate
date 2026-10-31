import json, os, sys, collections
S = sys.argv[1]
L = {"kr": "KR_", "en": "EN_", "jp": "JP_"}
def files(lang):
    base = os.path.join(S, "raw", lang, "LocalizeTemp_" + lang); out = {}
    for dp, _, fs in os.walk(base):
        for f in fs:
            rel = os.path.relpath(os.path.join(dp, f), base); d, n = os.path.split(rel)
            if n.startswith(L[lang]): n = n[len(L[lang]):]
            out[os.path.join(d, n)] = os.path.join(dp, f)
    return out
F = {l: files(l) for l in L}
load = lambda p: json.loads(open(p, "rb").read().decode("utf-8-sig"))
def items(d): return d.get("dataList") if isinstance(d, dict) and isinstance(d.get("dataList"), list) else None
def ident(x):
    if not isinstance(x, dict): return None
    return x.get("id") if "id" in x else None
res = {lang: collections.Counter() for lang in ("en", "jp")}; ex = {lang: collections.defaultdict(list) for lang in ("en", "jp")}
nested = {lang: collections.Counter() for lang in ("en", "jp")}
for rel in sorted(set(F["kr"]) & set(F["en"]) & set(F["jp"])):
    k = items(load(F["kr"][rel]))
    for lang in ("en", "jp"):
        o = items(load(F[lang][rel]))
        if k is None or o is None: res[lang]["no_dataList"] += 1; continue
        ki = [ident(x) for x in k]; oi = [ident(x) for x in o]
        if ki == oi:
            # same top-level order; check nested lists order by their ids
            for a, b in zip(k, o):
                if isinstance(a, dict) and isinstance(b, dict):
                    for key, va in a.items():
                        vb = b.get(key)
                        if isinstance(va, list) and isinstance(vb, list) and va and isinstance(va[0], dict):
                            ia = [ident(x) for x in va]; ib = [ident(x) for x in vb]
                            if ia != ib: nested[lang][key] += 1
            if all(i is None for i in ki):
                km = [x.get("model") for x in k if isinstance(x, dict)]; om = [x.get("model") for x in o if isinstance(x, dict)]
                res[lang]["same_len_no_id" if len(k) == len(o) else "diff_len_no_id"] += 1
                if km != om and len(k) == len(o): res[lang]["no_id_model_seq_differs"] += 1
            else: res[lang]["same_order"] += 1
            continue
        if collections.Counter(map(str, ki)) == collections.Counter(map(str, oi)):
            res[lang]["same_ids_diff_order"] += 1
            if len(ex[lang]["order"]) < 6: ex[lang]["order"].append((rel, len(ki), sum(a != b for a, b in zip(ki, oi))))
        else:
            sk, so = set(map(str, ki)), set(map(str, oi))
            res[lang]["diff_id_set"] += 1
            if len(ex[lang]["set"]) < 6: ex[lang]["set"].append((rel, len(ki), len(oi), len(sk - so), len(so - sk)))
for lang in ("en", "jp"):
    print(f"== KR vs {lang.upper()}:", dict(res[lang]))
    print("   reordered (file, items, positions differing):", ex[lang]["order"])
    print("   id set differs (file, kr items, other items, kr-only ids, other-only ids):", ex[lang]["set"])
    print("   nested list order differs:", dict(nested[lang]))
