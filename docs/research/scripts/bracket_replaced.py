import json, sys, re, collections
U = sys.argv[1]
HAN = re.compile(r"[一-鿿]")
kept = collections.Counter(); repl = collections.Counter(); ex = collections.defaultdict(list); fam = collections.Counter()
for line in open(U, encoding="utf-8"):
    u = json.loads(line); t = u["para"]
    if not t or t == u["en"] or not HAN.search(t): continue
    for k in set(re.findall(r"\[([A-Za-z][A-Za-z0-9_]*)\]", u["kr"])):
        if f"[{k}]" in t: kept[k] += 1
        else:
            repl[k] += 1
            f = u["file"]; fam[re.split(r"[_\-]", f.split("/")[-1][:-5])[0]] += 1
            if len(ex[k]) < 2: ex[k].append((f"{u['file']}#{u['eid']}", u["kr"][:80], t[:80]))
print("total replaced", sum(repl.values()), "kept", sum(kept.values()))
print("by family:", fam.most_common(10))
print("\nmost replaced IDs (replaced / kept):")
for k, v in repl.most_common(25): print(f"  [{k}] {v} / {kept[k]}")
print()
for k, _ in repl.most_common(8):
    for e in ex[k]: print(f"[{k}] {e[0]}\n   KR: {e[1]}\n   ZH: {e[2]}")
