import json, sys, collections, re
S = sys.argv[1]
smc = {}
for line in open(f"{S}/units.jsonl", encoding="utf-8"):
    u = json.loads(line)
    if u["file"].startswith("ScenarioModelCodes") and u["field"] in ("name", "nickName"):
        smc.setdefault(u["eid"], {})[u["field"]] = (u["kr"], u["para"], u["ltm"])
names = ["이상","파우스트","돈키호테","료슈","뫼르소","홍루","히스클리프","이스마엘","로쟈","싱클레어","오티스","그레고르","단테","베르길리우스","카론"]
for n in names:
    keys = [k for k in smc if isinstance(k, str) and n in k]
    print(n, "->", smc.get(n, {}).get("name"), "| nick:", smc.get(n, {}).get("nickName"), "| variant keys:", len(keys), keys[:12])
