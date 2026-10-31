import json, sys, re, collections

END = [("합쇼", r"(니다|니까|십시오|십시다)$"), ("해요", r"요$"), ("하오", r"(오|소|구려|시오|소서|게나|구먼|네만|겠네|하게)$"),
       ("반말", r"(다|어|아|야|지|네|군|냐|니|라|자|게|걸|데|까|래|대|든|고|해|봐|줘|와|워)$")]
def klass(t):
    c = collections.Counter()
    for s in re.split(r"[\.\!\?…\n]+", t):
        s = re.sub(r"[\s\"'”’\)\]」』~—\-,]+$", "", s.strip())
        if not s or not re.search(r"[가-힣]$", s): continue
        for n, rx in END:
            if re.search(rx, s): c[n] += 1; break
    if not c: return None
    if c["합쇼"] or c["해요"]: return "존댓말"
    if c["하오"]: return "하오"
    return "반말"
def main():
    L = json.load(open(sys.argv[1], encoding='utf-8'))
    tab = collections.defaultdict(lambda: collections.Counter()); ex = collections.defaultdict(list)
    for key, us in L.items():
        c = key.split("|")[0]
        for u in us:
            z = u["para"]; nin = "您" in z; ni = bool(re.search(r"你|妳", z))
            if nin == ni: continue  # need exactly one kind
            k = klass(u["kr"])
            if not k: continue
            zz = "您" if nin else "你"
            tab[c][(k, zz)] += 1; tab["ALL"][(k, zz)] += 1
            if c in ("ishmael", "faust", "honglu") and k == "존댓말" and zz == "你" and len(ex[c]) < 3: ex[c].append((u["file"] + "#" + str(u["eid"]), u["kr"][:70], z[:60]))
    print("KR politeness of line  x  zh 2nd person (lines with exactly one of 您 / 你·妳)")
    for c in ["ALL","yisang","faust","donquixote","ryoshu","meursault","honglu","heathcliff","ishmael","rodion","sinclair","outis","gregor","dante","vergilius","charon"]:
        t = tab[c]
        def r(k): a, b = t[(k, "您")], t[(k, "你")]; return f"{k}: 您 {a} / 你 {b}" + (f" ({a*100/(a+b):.0f}% 您)" if a + b else "")
        print(f"{c:11s} " + " | ".join(r(k) for k in ["존댓말", "하오", "반말"]))
    for c, v in ex.items():
        print(f"\n{c}: 존댓말 but 你"); [print("  ", *e, sep="\n    ") for e in v]
if __name__ == '__main__':
    main()
