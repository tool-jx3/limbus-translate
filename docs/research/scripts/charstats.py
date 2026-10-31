import json, sys, collections, re
S = sys.argv[1]
H = re.compile(r"[가-힣]"); HAN = re.compile(r"[一-鿿]")
CH = {"이상":"yisang","파우스트":"faust","돈키호테":"donquixote","료슈":"ryoshu","뫼르소":"meursault","홍루":"honglu","히스클리프":"heathcliff",
      "이스마엘":"ishmael","로쟈":"rodion","싱클레어":"sinclair","오티스":"outis","그레고르":"gregor","단테":"dante","베르길리우스":"vergilius","카론":"charon"}
FILECH = {"yisang":"yisang","faust":"faust","donquixote":"donquixote","ryoshu":"ryoshu","meursault":"meursault","merusault":"meursault","meursalut":"meursault",
          "honglu":"honglu","heathcliff":"heathcliff","ishmael":"ishmael","rodion":"rodion","sinclair":"sinclair","outis":"outis","outist":"outis","gregor":"gregor"}
def ok(u): v = u["para"]; return v and v != u["en"] and v != u["kr"] and HAN.search(v) and not H.search(v)
lines = collections.defaultdict(list)
for line in open(f"{S}/units.jsonl", encoding="utf-8"):
    u = json.loads(line); f = u["file"]
    if f.startswith("StoryData/") and u["field"] == "content" and u["model"] in CH and ok(u):
        lines[(CH[u["model"]], "story")].append(u)
    elif f.startswith(("PersonalityVoiceDlg/", "EGOVoiceDig/")) and u["field"] == "dlg" and ok(u):
        m = re.match(r".*Voice_(?:EGO_)?([A-Za-z]+)_", f); c = FILECH.get(m.group(1).lower()) if m else None
        if c: lines[(c, "voice")].append(u)
    elif f.startswith("AbDlg_") and u["field"] in ("dlg", "content", "desc") and ok(u):
        c = FILECH.get(f[6:-5].lower())
        if c: lines[(c, "voice")].append(u)
json.dump({f"{k[0]}|{k[1]}": v for k, v in lines.items()}, open(f"{S}/charlines.json", "w", encoding="utf-8"), ensure_ascii=False)
ZH = {"self": ["我","吾","本小姐","人家","俺","在下","本人","咱","老子","浮士德"],
      "you": ["你","您","汝","閣下","妳"],
      "addr": ["先生","小姐","女士","老爺","經理兄","老兄","但丁經理","經理","維吉","大人","前輩"],
      "nick": ["格雷格","以實","希斯","小唐","小辛","小浮","小良","小莫","小鴻"],
      "end": ["嘛","啦","呢","吧","喔","哦","呀","啊","唷","耶","囉","欸","哪"],
      "punct": ["～","~","……","...","！","？","「","“"]}
KR = {"저/제(謙)": r"(?:^|[\s\"'…])(?:저는|저도|저를|저의|제가|제게|저에게|저한테)", "나/내": r"(?:^|[\s\"'…])(?:나는|나도|나를|나의|내가|내게|나에게|나한테)",
      "이 몸": r"이 몸", "당신": r"당신", "너/네": r"(?:^|[\s\"'…])(?:너는|너도|너를|너의|네가|너에게|너한테)", "그대": r"그대", "자네": r"자네",
      "-님": r"님(?:[은이을께의,.!?…\s]|$)", "-씨": r"(?:[가-힣]) ?씨(?:[는가를의,.!?…\s]|$)", "관리자 양반": r"관리자 양반", "나리": r"나리"}
def ending(s):
    s = re.sub(r"[\s\.\!\?…~\"'”’\)\]」』—\-]+$", "", s.strip())
    if not s: return None
    if re.search(r"(니다|니까|십시오|시죠|세요)$", s): return "합쇼/세요"
    if s.endswith("요"): return "해요"
    if re.search(r"(오|소|구려|시오|소서|이오|하오)$", s): return "하오"
    return "반말"
out = {}
for (c, kind), us in sorted(lines.items()):
    zh = collections.Counter(); kr = collections.Counter(); en = collections.Counter()
    for u in us:
        t = u["para"]
        for cat, toks in ZH.items():
            for tk in toks:
                if tk in t: zh[(cat, tk)] += 1
        for k, rx in KR.items():
            if re.search(rx, u["kr"]): kr[k] += 1
        for sent in re.split(r"(?<=[\.\!\?…])\s+|\n", u["kr"]):
            e = ending(sent)
            if e: en[e] += 1
    out[f"{c}|{kind}"] = (len(us), zh, kr, en)
    n = len(us); te = sum(en.values()) or 1
    def top(cat): return ", ".join(f"{tk} {v*100/n:.0f}%" for (ct, tk), v in zh.most_common() if ct == cat and v*100/n >= 1)
    print(f"\n## {c} [{kind}] lines={n}")
    print("  KR endings:", ", ".join(f"{k} {v*100/te:.0f}%" for k, v in en.most_common()))
    print("  KR person/address:", ", ".join(f"{k} {v*100/n:.1f}%" for k, v in kr.most_common() if v))
    for cat in ("self", "you", "addr", "nick", "end", "punct"): print(f"  zh {cat}:", top(cat))
