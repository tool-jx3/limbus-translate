import json, sys, collections, re
S = sys.argv[1]
L = json.load(open(f"{S}/charlines.json", encoding="utf-8"))
def lines(c, kinds=("story", "voice")): return [u for k in kinds for u in L.get(f"{c}|{k}", [])]
END = [("합쇼", r"(니다|니까|십시오|십시다)$"), ("해요", r"요$"), ("하오", r"(오|소|구려|시오|소서|게나|구먼)$"),
       ("반말", r"(다|어|아|야|지|네|군|냐|니|라|자|게|걸|데|까|래|대|든|고|해|봐|줘|와|워)$")]
def endings(t):
    c = collections.Counter()
    for s in re.split(r"[\.\!\?…\n]+", t):
        s = re.sub(r"[\s\"'”’\)\]」』~—\-,]+$", "", s.strip())
        if not s or not re.search(r"[가-힣]$", s): continue
        for name, rx in END:
            if re.search(rx, s): c[name] += 1; break
    return c
NICK = {"格雷格": r"格雷格", "以實(暱稱)": r"以實(?!瑪)", "希斯(暱稱)": r"希斯(?!克)", "小唐": r"小唐", "小辛": r"小辛", "小浮": r"小浮", "小良": r"小良", "小莫": r"小莫", "小鴻": r"小鴻", "小李": r"小李", "小奧": r"小奧"}
print("== KR sentence endings (verb-final sentences only) & nicknames")
for c in ["yisang","faust","donquixote","ryoshu","meursault","honglu","heathcliff","ishmael","rodion","sinclair","outis","gregor","dante","vergilius","charon"]:
    us = lines(c); e = collections.Counter()
    for u in us: e += endings(u["kr"])
    t = sum(e.values()) or 1
    nk = {k: sum(1 for u in us if re.search(rx, u["para"])) for k, rx in NICK.items()}
    print(f"{c:11s} n={len(us):5d}  " + ", ".join(f"{k} {v*100/t:.0f}%" for k, v in e.most_common()) + "  | nick: " + ", ".join(f"{k}={v}" for k, v in nk.items() if v))
print("\n== How each character addresses Dante (lines whose KR names 관리자/단테): zh 您 vs 你/妳 and address term")
for c in ["yisang","faust","donquixote","ryoshu","meursault","honglu","heathcliff","ishmael","rodion","sinclair","outis","gregor","vergilius","charon"]:
    us = [u for u in lines(c) if re.search(r"관리자|단테", u["kr"])]
    n = len(us) or 1
    nin = sum("您" in u["para"] for u in us); ni = sum(bool(re.search("你|妳", u["para"])) for u in us)
    kr = collections.Counter(m for u in us for m in re.findall(r"관리자 ?양반|관리자님|관리자 ?나리|단테 ?나리|단테 ?님|단테 ?씨|단테 ?경|관리자", u["kr"]))
    zh = collections.Counter(m for u in us for m in re.findall(r"但丁老爺|經理老爺|經理兄|老兄|但丁經理|經理大人|但丁先生|但丁大人|執行經理|經理|但丁", u["para"]))
    print(f"{c:11s} n={len(us):4d} 您={nin} 你/妳={ni} | KR {dict(kr.most_common(4))} | zh {dict(zh.most_common(5))}")
print("\n== Faust self-reference: KR 파우스트+particle vs 저/제; zh 浮士德 vs 我(not 我們)")
us = lines("faust"); kname = sum(bool(re.search(r"파우스트(?:는|가|도|를|의|에게|한테|만|로서)", u["kr"])) for u in us)
kjeo = sum(bool(re.search(r"(?:^|[\s\"'…])(?:저는|저도|저를|저의|제가|제게)", u["kr"])) for u in us)
zn = sum("浮士德" in u["para"] for u in us); zw = sum(bool(re.search(r"我(?!們)", u["para"])) for u in us)
both = sum(bool(re.search(r"파우스트(?:는|가|도|를|의)", u["kr"])) and "浮士德" in u["para"] for u in us)
print(f"lines={len(us)} KR name-self={kname} KR 저/제={kjeo} | zh 浮士德={zn} zh 我={zw} | KR name-self & zh 浮士德 = {both}")
for c, pats in {"donquixote": ["浮士德女士","浮士德小姐","希斯克利夫先生","辛克萊","但丁老爺","經理老爺","維吉里烏斯老爺","老爺"],
                "vergilius": ["浮士德女士","但丁經理","女士"], "charon": ["維吉"], "gregor": ["經理兄","老兄"], "yisang": ["女士","小姐"],
                "heathcliff": ["先生","小姐","女士"], "faust": ["先生","小姐","女士"], "honglu": ["先生","小姐","女士","閣下"]}.items():
    us = lines(c); print(f"{c}: " + ", ".join(f"{p}={sum(p in u['para'] for u in us)}" for p in pats))
