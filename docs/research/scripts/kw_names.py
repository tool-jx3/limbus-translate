import json, sys, re, collections
U = sys.argv[1]; HAN = re.compile(r"[一-鿿]")
names = {}; rows = []
for line in open(U, encoding="utf-8"):
    u = json.loads(line)
    if re.match(r"(Bufs|BattleKeywords)", u["file"]) and u["field"] == "name" and isinstance(u["eid"], str) and u["para"] and u["para"] != u["en"]:
        names.setdefault(u["eid"], collections.Counter())[u["para"]] += 1
    t = u["para"]
    if t and t != u["en"] and HAN.search(t) and "[" in u["kr"]: rows.append(u)
off = {k: c.most_common(1)[0][0] for k, c in names.items()}
ok = bad = 0; badex = collections.defaultdict(collections.Counter)
for u in rows:
    t = u["para"]
    for k in set(re.findall(r"\[([A-Za-z][A-Za-z0-9_]*)\]", u["kr"])):
        if f"[{k}]" in t or k not in off: continue
        if off[k] in t: ok += 1
        else:
            bad += 1
            m = re.findall(r"[一-鿿]{2,8}(?= )", t); badex[k][" / ".join(m[:3])] += 1
print(f"replaced IDs whose official name appears: {ok}, not found: {bad}")
for k, c in sorted(badex.items(), key=lambda x: -sum(x[1].values()))[:10]:
    print(f"  [{k}] official「{off[k]}」 -> text has: {c.most_common(2)}")
