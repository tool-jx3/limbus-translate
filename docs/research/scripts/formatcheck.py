import json, sys, collections, re
S = sys.argv[1]
H = re.compile(r"[가-힣]"); HAN = re.compile(r"[一-鿿]")
IDF = {"key","songWriter","usage","imgStr","model","id","codeName"}
CJKP = "，。：；！？、「」『』（）《》〈〉【】…—～"
c = collections.Counter(); ex = collections.defaultdict(list)
def add(k, u, s=None):
    c[k] += 1
    if len(ex[k]) < 3: ex[k].append(f"{u['file']}#{u['eid']} :: {(s or u['para'])[:90]!r}")
skillfam = re.compile(r"^(Skills|Passive|Bufs|BattleKeywords|EGOgift|Enemies|PanicInfo)")
for line in open(f"{S}/units.jsonl", encoding="utf-8"):
    u = json.loads(line); t = u["para"]; kr = u["kr"]
    if u["field"] in IDF or not t or t == u["en"] or not HAN.search(t): continue
    sk = bool(skillfam.match(u["file"]))
    c["units"] += 1; c["skill_units"] += sk
    # bracket ids preserved
    for k in set(re.findall(r"\[([A-Za-z][A-Za-z0-9_]*)\]", kr)):
        c["kr_bracket_ids"] += 1
        if f"[{k}]" in t: c["bracket_kept"] += 1
        else: add("bracket_replaced", u)
    for k in set(re.findall(r"<([A-Za-z]+)[=>]", kr)):
        c["kr_tags"] += 1
        if not re.search(rf"<{k}[=> ]", t): add("tag_missing", u)
    if "&lt;" in t or "&gt;" in t: add("escaped_lt_gt", u)
    # spacing around [] (rule 6)
    for m in re.finditer(r"\[[A-Za-z][A-Za-z0-9_]*\]", t):
        i, j = m.start(), m.end(); tok = m.group()
        if tok == "[TabExplain]":
            if i > 0 and t[i-1] == " ": add("tabexplain_space_before", u)
            continue
        if i > 0 and t[i-1] not in " \n" + CJKP + "<>\"'": add("no_space_before_bracket", u)
        else: c["space_before_ok"] += 1
        if j < len(t) and t[j] not in " \n" + CJKP + "<>\"'.,:;!?)": add("no_space_after_bracket", u)
        else: c["space_after_ok"] += 1
    if re.search(r"\]\[", t): add("adjacent_brackets_no_space", u)
    # brackets width (rule 7)
    if "（" in t or "）" in t: add("fullwidth_paren", u) if sk else c.__setitem__("fullwidth_paren_nonskill", c["fullwidth_paren_nonskill"] + 1)
    if re.search(r"[一-鿿][()]|[()][一-鿿]", t): c["halfwidth_paren_near_han"] += 1
    # punctuation width (rule 8)
    if re.search(r"[一-鿿],|,[一-鿿]", t): add("halfwidth_comma", u)
    if re.search(r"[一-鿿]:[一-鿿]", t): add("halfwidth_colon", u)
    if re.search(r"[一-鿿]\.(?:\s|$)", t): add("halfwidth_period", u)
    # potency/count wording (rule 9)
    if sk:
        for pat in [r"\d+級", r"強度", r"\d+回", r"\d+層", r"層數", r"回數", r"次數", r"威力"]:
            if re.search(pat, t): c[f"rule9:{pat}"] += 1
    # general punctuation
    for k, rx in {"「": "「", "“": "“", "……": "……", "...": r"\.\.\.", "～": "～", "~": "~", "——": "——"}.items():
        if re.search(rx, t): c[f"punct:{k}"] += 1
print({k: v for k, v in c.items() if not k.startswith(("rule9", "punct"))})
print("rule9:", {k: v for k, v in c.items() if k.startswith("rule9")})
print("punct:", {k: v for k, v in c.items() if k.startswith("punct")})
for k, v in ex.items(): print(f"\n[{k}] x{c[k]}"); [print("   ", e) for e in v]
