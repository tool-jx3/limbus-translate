import json, os, sys, collections, re
root = sys.argv[1]
base = os.path.join(root, "kr", "LocalizeTemp_kr")
def flat(v, path, out):
    if isinstance(v, dict):
        for k, x in v.items(): flat(x, path + (k,), out)
    elif isinstance(v, list):
        for i, x in enumerate(v): flat(x, path + (i,), out)
    elif isinstance(v, str) and v.strip() not in ("", "-"): out.append((path, v))
fam = collections.defaultdict(lambda: [0, 0, 0, []])  # files, strings, chars, examples
for dp, _, fs in os.walk(base):
    for f in fs:
        rel = os.path.relpath(os.path.join(dp, f), base)
        d, n = os.path.split(rel)
        n = n[3:] if n.startswith("KR_") else n
        stem = n[:-5]
        if d:
            key = d + "/" + re.split(r"[_\-]", stem)[0]
        else:
            key = re.split(r"[_\-]", stem)[0]
            key = re.sub(r"\d+$", "", key)
        try: data = json.loads(open(os.path.join(dp, f), "rb").read().decode("utf-8-sig"))
        except Exception: continue
        out = []; flat(data, (), out)
        e = fam[key]; e[0] += 1; e[1] += len(out); e[2] += sum(len(v) for _, v in out)
        if len(e[3]) < 3: e[3].append(stem)
rows = sorted(fam.items(), key=lambda x: -x[1][2])
print(f"{len(rows)} families; top by KR characters:")
tot = sum(v[2] for _, v in rows)
for k, (nf, ns, nc, ex) in rows[:45]:
    print(f"  {k:32s} files={nf:4d} strings={ns:6d} chars={nc:8d} ({nc*100/tot:4.1f}%)  e.g. {', '.join(ex[:2])}")
print("... remaining", len(rows) - 45, "families,", sum(v[2] for _, v in rows[45:]) * 100 / tot, "% of chars")
