import json, sys, re, collections
U, C = sys.argv[1], sys.argv[2]
L = json.load(open(C, encoding="utf-8"))
dq = L["donquixote|story"] + L["donquixote|voice"]
print("DQ KR forms for Dante:", collections.Counter(m for u in dq for m in re.findall(r"(?:관리자|단테) ?(?:나리|님|씨|경|공)?", u["kr"])).most_common(6))
print("DQ KR forms for Faust:", collections.Counter(m for u in dq for m in re.findall(r"파우스트 ?(?:양|씨|님|공|아가씨|여사|경|선생)?", u["kr"])).most_common(6))
for u in dq:
    if re.search(r"파우스트", u["kr"]) and re.search(r"浮士德(女士|小姐)", u["para"]): print("  ", u["file"], "|", u["kr"][:50], "|", u["para"][:40])
print("\nWho uses which form for Faust (KR → zh) across all characters:")
tab = collections.Counter()
for k, us in L.items():
    for u in us:
        for m in re.findall(r"파우스트 ?(양|씨|님|아가씨)", u["kr"]):
            z = "女士" if "浮士德女士" in u["para"] else "小姐" if "浮士德小姐" in u["para"] else "其他"
            tab[(k.split("|")[0], m, z)] += 1
for k, v in sorted(tab.items(), key=lambda x: -x[1])[:14]: print("  ", k, v)
q = collections.Counter(); qe = []; p = collections.Counter(); pe = []
for line in open(U, encoding="utf-8"):
    u = json.loads(line); t = u["para"] or ""
    if "定向提取" in t:
        m = re.findall(r"[가-힣]+ ?추출|[가-힣]+ 추출", u["kr"]); q[tuple(m[:2])] += 1
        if len(qe) < 3: qe.append((u["file"], u["kr"][:60], t[:40]))
    if "人格碎片" in t:
        m = re.findall(r"[가-힣]+ ?파편", u["kr"]); p[tuple(m[:2])] += 1
        if len(pe) < 4: pe.append((u["file"], u["kr"][:60], t[:40]))
print("\n定向提取 ← KR:", q.most_common()); [print("  ", *e, sep=" | ") for e in qe]
print("人格碎片 ← KR:", p.most_common()); [print("  ", *e, sep=" | ") for e in pe]
