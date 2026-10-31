import json, sys, re
S = sys.argv[1]
L = json.load(open(f"{S}/charlines.json", encoding="utf-8"))
MARK = {"yisang": [r"女士", r"(오|소)[\.\?!…]*$"], "faust": [r"浮士德", r"您"], "donquixote": [r"吾", r"汝", r"老爺"], "ryoshu": [r"."],
        "meursault": [r"經理", r"您"], "honglu": [r"閣下", r"～"], "heathcliff": [r"."], "ishmael": [r"您", r"經理"],
        "rodion": [r"格雷格|以實(?!瑪)|希斯(?!克)|小辛|小浮|小唐|小莫", r"～"], "sinclair": [r"您", r"小姐|先生"], "outis": [r"經理", r"在下|您"],
        "gregor": [r"經理兄|老兄"], "dante": [r"."], "vergilius": [r"女士|但丁經理", r"."], "charon": [r"維吉", r"."]}
out = {}
for c, rxs in MARK.items():
    us = [u for k in ("story", "voice") for u in L.get(f"{c}|{k}", [])]
    picked, files = [], set()
    for rx in rxs:
        for u in us:
            if len(picked) >= 3: break
            if u["file"] in files or not (12 <= len(u["kr"]) <= 70) or "\n" in u["kr"]: continue
            if re.search(rx, u["para"]) or re.search(rx, u["kr"]):
                picked.append(u); files.add(u["file"]); break
    for u in us:
        if len(picked) >= 3: break
        if u["file"] not in files and 12 <= len(u["kr"]) <= 70 and "\n" not in u["kr"]: picked.append(u); files.add(u["file"])
    out[c] = [{"src": f"{u['file']}#{u['eid']}", "kr": u["kr"], "zh": u["para"]} for u in picked]
json.dump(out, open(f"{S}/examples.json", "w", encoding="utf-8"), ensure_ascii=False, indent=1)
for c, v in out.items():
    print(c); [print("  ", e["src"], "|", e["kr"], "|", e["zh"]) for e in v]
