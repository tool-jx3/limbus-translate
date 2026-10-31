import json, sys, re, collections
H = re.compile(r"[가-힣]"); KANA = re.compile(r"[぀-ヿ]"); HAN = re.compile(r"[一-鿿]")
IDF = {"key","songWriter","usage","imgStr","debugNodeId","debugNodeID","colorCode","outlineColorCode","timeline","chapterNumber","keywords","iconId","iconID","mainTextColor","mainTextGlowColor","model","id","codeName"}
def state(v, en, kr):
    if v is None: return "missing"
    if v == en and v != kr: return "en_copy"
    if v == kr: return "kr_copy"
    if H.search(v): return "hangul"
    if not HAN.search(v): return "no_han"
    return "zh"
c = collections.Counter(); cl = collections.Counter(); both = collections.Counter(); diffpl = 0; same = 0
fam = collections.defaultdict(collections.Counter)
for line in open(sys.argv[1], encoding="utf-8"):
    u = json.loads(line)
    if u["field"] in IDF: continue
    if u["kr"] == u["en"] == u["jp"]: continue
    sp, sl = state(u["para"], u["en"], u["kr"]), state(u["ltm"], u["en"], u["kr"])
    c[sp] += 1; cl[sl] += 1
    f = u["file"]; g = f.split("/")[0] if "/" in f else re.split(r"[_\-]", f[:-5])[0]
    fam[g][sp] += 1
    if sp == "zh" and sl == "zh":
        same += u["para"] == u["ltm"]; diffpl += u["para"] != u["ltm"]
tot = sum(c.values())
print("translatable units:", tot)
print("Paratranz:", {k: f"{v} ({v*100/tot:.1f}%)" for k, v in c.most_common()})
print("LTM:", {k: f"{v} ({v*100/tot:.1f}%)" for k, v in cl.most_common()})
print(f"both zh: identical {same}, different {diffpl}")
print("\nParatranz zh coverage by family (worst first, families with >300 units):")
rows = [(g, cc["zh"] / sum(cc.values()), sum(cc.values()), cc) for g, cc in fam.items() if sum(cc.values()) > 300]
for g, r, n, cc in sorted(rows, key=lambda x: x[1])[:25]:
    print(f"  {g:28s} zh {r*100:5.1f}%  of {n:6d}   en_copy {cc['en_copy']}, missing {cc['missing']}")
