import json, sys, re, collections
L = json.load(open(sys.argv[1], encoding="utf-8"))
sys.path.insert(0, sys.argv[2]); from honorific import klass  # reuse
HON = re.compile(r"(님|께서|께|[하가오보주되계]시|으시|셨|세요|셔요|십시오|십니까|드리|드릴|드립|드려)")
tab = collections.Counter(); ex = collections.defaultdict(list)
for key, us in L.items():
    for u in us:
        z = u["para"]; nin = "您" in z; ni = bool(re.search(r"你|妳", z))
        if nin == ni or klass(u["kr"]) != "존댓말": continue
        kr = u["kr"]; hon = bool(HON.search(kr)); ds = "당신" in kr
        g = ("敬語標記" if hon else "無標記") + ("+당신" if ds else "")
        zz = "您" if nin else "你"; tab[(g, zz)] += 1
        if len(ex[(g, zz)]) < 2: ex[(g, zz)].append((key, u["file"] + "#" + str(u["eid"]), kr[:60], z[:50]))
for g in ["敬語標記", "敬語標記+당신", "無標記", "無標記+당신"]:
    a, b = tab[(g, "您")], tab[(g, "你")]
    print(f"{g:10s} 您 {a:4d} / 你 {b:4d}  ({a*100/max(1,a+b):.0f}% 您)")
for k, v in ex.items():
    if k[0] in ("敬語標記", "無標記+당신"):
        print(k); [print("   ", *e, sep=" | ") for e in v]
