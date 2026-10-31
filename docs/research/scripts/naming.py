import json, os, sys, collections, re
S = sys.argv[1]; base = f"{S}/raw/kr/LocalizeTemp_kr"; para = f"{S}/tw/para"
sd = sorted(f[3:-5] for f in os.listdir(f"{base}/StoryData"))
pats = [("S + 3–4碼 主線", r"^S\d{3,4}[A-Z]?\d*$"), ("ZDYXX 主線地牢", r"^\d+D\d{3}[A-Z]?\d*$"), ("P1YYXX 人格故事", r"^P1\d{4}$"),
        ("E + 3碼 間章/活動", r"^E\d{3}[A-Z]?\d*$"), ("PC + 2碼", r"^PC\d+[A-Z]?\d*$")]
left = list(sd)
for name, rx in pats:
    m = [x for x in left if re.match(rx, x)]; left = [x for x in left if not re.match(rx, x)]
    print(f"{name}: {len(m)} 例 {m[:4]}")
print("其他 StoryData:", len(left), left[:25])
# what PC is: show place/first lines from Paratranz
for f in [x for x in sd if x.startswith("PC")][:2]:
    d = json.load(open(f"{para}/StoryData/{f}.json", encoding="utf-8-sig"))["dataList"]
    print(f, [x.get("place") or x.get("content", "")[:30] for x in d[:3]])
P = collections.Counter(int(x[2:4]) for x in sd if re.match(r"^P1\d{4}$", x)); print("P1YYXX by YY:", dict(sorted(P.items())))
names = sorted(f[3:-5] for f in os.listdir(base) if f.endswith(".json"))
for label, rx in [("鏡牢", r"Mirror|MD\d|DungeonStartBuffs"), ("飾品 EGOgift", r"^EGOgift"), ("異想體", r"^Abnormality|^AbEvents|^AbDlg"),
                  ("人格/E.G.O", r"^(Personalit|Skills_Personality|Skills_Ego|Passive_Ego|Passives_Personality|Egos?\b|Ego)"),
                  ("敵方", r"^(Enemies|Skills_Enemy|Passives_Enemy|Skills_Abnormality|Passives_Abnormality)"), ("但丁筆記", r"DanteNote"),
                  ("瓦夜 walpu", r"walpu|Walpu"), ("折射軌道", r"Refraction|Railway"), ("RPG", r"^rpg"), ("戰鬥泡泡", r"BattleSpeechBubble")]:
    m = [x for x in names if re.search(rx, x)]
    print(f"{label}: {len(m)} 檔，例 {m[:6]}")
d = json.load(open(f"{base}/KR_BattleSpeechBubbleDlg-a1c10p3.json", encoding="utf-8-sig"))["dataList"]
print("BattleSpeechBubble ids:", [x.get("id") for x in d[:6]], "files:", [x for x in names if "BattleSpeechBubble" in x])
print("BgmLyrics files:", sorted(os.listdir(f"{base}/BgmLyrics"))[:8])
