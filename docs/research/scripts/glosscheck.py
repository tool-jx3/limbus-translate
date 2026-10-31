import json, sys, collections, re
S = sys.argv[1]
T = [("도약 모듈", "跳躍成長模組", ["跳躍成長模塊"]), ("특정 추출", "特定提取", ["特定抽取"]), ("패키지", "組合包", ["禮包"]),
     ("인격 학습 티켓", "人格訓練券", ["人格訓練卷", "人格經驗券"]), ("3성 인격 출현 추출 티켓", "3星人格必得十連提取券", ["3★人格必得十連提取券"]),
     ("인격 특수 강화 티켓", "人格等級直升券", []), ("인격 성장 도약 티켓", "人格跳躍成長券", []), ("시즌", "賽季", [r"第\d+季"]),
     ("주요 이야기", "主線劇情", ["主線故事"]), ("자아 파편", "自我碎片", ["人格碎片"]), ("배틀 패스", "戰鬥通行證", []), ("림버스 패스", "巴士通行證", ["邊獄通行證"])]
kr_hit = collections.Counter(); ok = collections.Counter(); bad = collections.Counter(); other = collections.defaultdict(collections.Counter)
corpus_ok = collections.Counter(); corpus_bad = collections.Counter()
for line in open(f"{S}/units.jsonl", encoding="utf-8"):
    u = json.loads(line); t = u["para"] or ""
    if not t or t == u["en"]: continue
    for kr, good, bads in T:
        if good in t: corpus_ok[kr] += 1
        for b in bads:
            if re.search(b, t): corpus_bad[(kr, b)] += 1
        if kr in u["kr"]:
            kr_hit[kr] += 1
            if good in t: ok[kr] += 1
            elif any(re.search(b, t) for b in bads): bad[kr] += 1
            else:
                m = re.findall(r"[一-鿿★\d]{0,4}(?:模組|模塊|提取|抽取|包|券|卷|季|劇情|故事|碎片|通行證)", t)
                other[kr][" / ".join(sorted(set(m))[:3]) or t[:24]] += 1
for kr, good, bads in T:
    print(f"{kr} -> {good}: KR lines {kr_hit[kr]}, approved {ok[kr]}, forbidden {bad[kr]}, corpus approved {corpus_ok[kr]}, corpus forbidden {[(b, corpus_bad[(kr,b)]) for b in bads]}")
    if other[kr]: print("     other renderings:", other[kr].most_common(4))
