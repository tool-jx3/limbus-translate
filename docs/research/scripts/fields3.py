import json, os, sys, collections, re
root = sys.argv[1]
L = {"kr": "KR_", "en": "EN_", "jp": "JP_"}
def files(lang):
    base = os.path.join(root, lang, "LocalizeTemp_" + lang); out = {}
    for dp, _, fs in os.walk(base):
        for f in fs:
            rel = os.path.relpath(os.path.join(dp, f), base); d, n = os.path.split(rel)
            if n.startswith(L[lang]): n = n[len(L[lang]):]
            out[os.path.join(d, n)] = os.path.join(dp, f)
    return out
F = {l: files(l) for l in L}
common = set(F["kr"]) & set(F["en"]) & set(F["jp"])
print("files kr/en/jp:", len(F["kr"]), len(F["en"]), len(F["jp"]), "common to all 3:", len(common))
load = lambda p: json.loads(open(p, "rb").read().decode("utf-8-sig"))
def flat(v, path, out):
    if isinstance(v, dict):
        for k, x in v.items(): flat(x, path + (k,), out)
    elif isinstance(v, list):
        for i, x in enumerate(v): flat(x, path + (i,), out)
    elif isinstance(v, str): out[path] = v
# stats per field name (last non-index key)
allsame = collections.Counter(); differs = collections.Counter(); emptyish = collections.Counter()
structmismatch = 0; hangul_in_en = collections.Counter()
H = re.compile(r"[가-힣]")
group = lambda rel: re.split(r"[-_.]", os.path.basename(rel))[0]
pergroup = collections.defaultdict(lambda: [0, 0])
for rel in sorted(common):
    try: d = {l: {} for l in L}; [flat(load(F[l][rel]), (), d[l]) for l in L]
    except Exception as e: print("bad", rel, e); continue
    ks = set(d["kr"]) & set(d["en"]) & set(d["jp"])
    if set(d["kr"]) != set(d["en"]) or set(d["kr"]) != set(d["jp"]): structmismatch += 1
    for p in ks:
        field = next((x for x in reversed(p) if isinstance(x, str)), "?")
        a, b, c = d["kr"][p], d["en"][p], d["jp"][p]
        if a.strip() in ("", "-"): emptyish[field] += 1; continue
        if a == b == c:
            allsame[field] += 1
        else:
            differs[field] += 1
        if field == "desc" or field == "dlg":
            pergroup[(group(rel), field)][0 if a == b == c else 1] += 1
print("files whose string-path sets differ between languages:", structmismatch)
print("\nfield: translatable(differs) / identical in all 3  [ratio identical]")
fields = sorted(set(allsame) | set(differs), key=lambda k: -(allsame[k] + differs[k]))
ident, mixed = [], []
for k in fields:
    t = allsame[k] + differs[k]; r = allsame[k] / t
    if differs[k] == 0: ident.append((k, t))
    elif r > 0.5: mixed.append((k, differs[k], allsame[k]))
print("ALWAYS identical across KR/EN/JP (never translate):")
print("  " + ", ".join(f"{k}({t})" for k, t in ident))
print("MOSTLY identical (>50%) but sometimes differ — need per-file rule:")
for k, dd, s in mixed: print(f"  {k}: differ {dd} / same {s}")
print("\nTop translatable fields:")
for k in fields[:25]:
    if differs[k]: print(f"  {k}: differ {differs[k]} / same {allsame[k]}")
print("\n'desc'/'dlg' by file group where mostly identical:")
for (g, f), (s, dd) in sorted(pergroup.items(), key=lambda x: -x[1][0]):
    if s > dd and s > 20: print(f"  {g}.{f}: same {s} / differ {dd}")
