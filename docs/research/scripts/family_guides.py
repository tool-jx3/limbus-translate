"""guides/families/*.md、guides/shared/*.md 與 base.md 第 7 條（括號）統計數字的重算腳本。

用法：python3 -I family_guides.py DIR
DIR 是 `lct site --out DIR` 寫出的網站資料（含 files/<路徑>，每檔有 family 與 units）。

「既有 a／b」：該族群 KR 符合條件、來源是 paratranz／claude／human（不含 copy）的 b 個單位中，
a 個的譯文符合寫法。只有 a/b >= 80% 且 b >= 30 的才寫進指引（印出時標 OK）。
共用指引（SHARED）的 a／b 是引入它的族群合計，另逐族群列出；每個族群有單位時也都要 >= 80%。
"""

import json
import os
import re
import sys
from collections import Counter

RECORDED = {"paratranz", "claude", "human"}
STATUS = r"(?P<x>Combustion|Laceration|Vibration|Burst|Sinking)"
ALLY = r"(?<![가-힣])아군(?!\s?전체)"
BRACKET_KR = r"\[[가-힣][가-힣 ]*\]"
BRACKET_ZH = r"\[[^\]]*[一-鿿][^\]]*\]"

# (族群, 說明, 欄位（空字串＝全部）, KR 樣式, 譯文樣式（{名稱} 代入 KR 的具名群組）, 檔名樣式)
SPECS = [
    ("ally", "[X] N 부여", "desc", r"\[" + STATUS + r"\] (?P<n>\d+) 부여", r"使目標增加{n}級 \[{x}\] 強度", ""),
    ("ally", "피해량 +N%", "", r"(?<!받는 )피해량 \+(?P<n>\d+)%", r"造成的傷害\+{n}%", ""),
    ("enemy", "[X] N 부여", "desc", r"\[" + STATUS + r"\] (?P<n>\d+) 부여", r"使目標增加{n}級 \[{x}\] 強度", ""),
    ("enemy", "피해량 +N%", "", r"(?<!받는 )피해량 \+(?P<n>\d+)%", r"造成的傷害\+{n}%", ""),
    ("keywords", "- 패닉", "", r"^- 패닉\n", r"^- 陷入恐慌\n", ""),
    ("keywords", "- 최댓값: N", "", r"(^|\n)- 최댓값 ?: ?(?P<n>\d+)", r"(^|\n)- 最大值：{n}", ""),
    ("keywords", "수치 N당", "", r"수치 (?P<n>\d+)당", r"每(帶)?有{n}層", ""),
    ("keywords", "공격 레벨", "", r"공격 레벨 (?P<s>[+-])(?P<n>\d+)", r"攻擊等級{s}{n}", ""),
    ("keywords", "적중 시", "", r"적중 시", r"命中時", ""),
    ("mirror", "선택 시,", "messageDesc", r"^선택 시, ", r"^選擇後，", ""),
    ("mirror", "E.G.O GIFT 획득", "messageDesc", r"(?<![0-9] )E\.G\.O GIFT 획득", r"獲得1件[^，。；\n]*E\.G\.O飾品", ""),
    ("mirror", "별도 효과 없음", "messageDesc", r"^별도 효과 없음$", r"^沒有效果$", ""),
    ("mirror", "판정 성공 시", "", r"판정 성공 시", r"判定 成功後", ""),
    ("mirror", "피해량 +N%", "", r"(?<!받는 )피해량 \+(?P<n>\d+)%", r"造成的傷害\+{n}%", ""),
    ("railway", "제 N 역", "", r"제 ?(?P<n>\d+) ?역", r"第{n}站", ""),
    ("railway", "제 N 구간", "", r"제 ?(?P<n>\d+) ?구간", r"第{n}區段", ""),
    ("railway", "N호선", "", r"(?P<n>\d+)호선", r"{n}號線", ""),
    ("railway", "굴절률", "", r"굴절률 ?: ?(?P<n>\d+)%", r"折射率：{n}%", ""),
    ("railway", "클리어", "", r"클리어", r"通關", ""),
    ("stage", "클리어", "", r"클리어", r"通關", ""),
    ("ui", " x N", "", r" x (?P<n>\d+)", r" × {n}", ""),
    ("abnormality", "[{n}] 수감자", "", r"\[\{(?P<n>\d)\}\] 수감자", r"罪人\[\{{n}\}\]", ""),
    ("abnormality", "어떤／어느 수감자", "behaveDesc", r"^(어떤|어느) 수감자", r"哪位罪人", ""),
    ("identity", "흐트러질 시 대사", "", r"^흐트러질 시 대사$", r"^自身混亂$", r"^PersonalityVoiceDlg/"),
    ("identity", "아군 사망", "", r"^아군 사망$", r"^友方陣亡$", r"^PersonalityVoiceDlg/"),
    ("ego-voice", "E.G.O 침식 발동", "desc", r"^E\.G\.O 침식 발동$", r"^E\.G\.O侵蝕$", ""),
    # 第二人稱：譯文有「你／妳」而沒有「您」的，占有第二人稱的單位多少
    ("ego-voice", "第二人稱用你", "dlg", r".", r"(?s)^(?=.*[你妳])(?!.*您)", ""),
]

# 共用指引：(檔, 引入的族群（同 routing.yaml 的 include）, 說明, 欄位, KR 樣式, 譯文樣式, 檔名樣式)
# a／b 是這些族群合計；逐族群的數字一併印出。
BATTLE = ["ally", "enemy", "keywords", "mirror"]
KOREAN_BRACKETS = ["ui", "events", "stage", "rpg", "ally", "enemy", "keywords", "mirror"]
SHARED = [
    ("battle-text", BATTLE, "(턴당 N회)", "", r"\(턴당 (?P<n>\d+)회\)", r"\(每回合最多{n}次\)", ""),
    ("battle-text", BATTLE, "최종 위력", "", r"최종 위력 (?P<s>[+-])(?P<n>\d+)", r"最終威力{s}{n}", ""),
    ("battle-text", BATTLE, "정신력 N 회복", "", r"정신력 (?P<n>\d+) 회복", r"恢復{n}點理智值", ""),
    ("battle-text", BATTLE, "체력 N 회복", "", r"체력 (?P<n>\d+) 회복", r"恢復{n}點體力", ""),
    ("battle-text", BATTLE, "아군", "", ALLY, r"友方單位", ""),
    ("battle-text", BATTLE, "처치 시", "", r"처치 시", r"擊殺.*(時|後)", ""),
    ("korean-brackets", KOREAN_BRACKETS, "[韓文]", "", BRACKET_KR, BRACKET_ZH, ""),
]

# 括號（base.md 第 7 條、shared/battle-text.md）：譯文去掉標籤後有括號的單位，分只用半形、只用全形、兩者都有。
# 戰鬥說明類照半形；KR 在韓文音譯後的括號附原文（라 노제(la nausée)）時寫「中文意思（原文）」是例外。
# 「半形括號」的 a：去掉這種「中文（原文）」後不剩全形括號的單位。
TAG = re.compile(r"<[^<>]*>")
LATIN = r"[A-Za-zÀ-ÿ'’][A-Za-zÀ-ÿ'’ ,.!?-]*"
ORIGINAL_KR = re.compile(r"[가-힣] ?\(" + LATIN + r"\)")
ORIGINAL_ZH = re.compile(r"[一-鿿] ?（" + LATIN + r"）")
HALF = re.compile(r"[()]")
FULL = re.compile(r"[（）]")

# 「- 」開頭的條數與 KR 相同
LIST_FAMILIES = ["dante-note", "keywords", "abnormality"]
LIST_MARK = re.compile(r"(^|\n)- ")

# 搬出 base.md 的規則：KR 含這個樣式的單位分在哪些族群
LOCATIONS = [
    ("語言樣式標籤", r'<style="dkr'),
    ("恐慌標題行", r"^- 패닉"),
    ("%…/% 樣板", r"%[^/]*/%"),
]


def load(root):
    files = []
    for base, _, names in os.walk(os.path.join(root, "files")):
        for name in names:
            with open(os.path.join(base, name), encoding="utf-8") as f:
                files.append(json.load(f))
    return files


def recorded(files, family, field="", path=""):
    path_re = re.compile(path) if path else None
    for d in files:
        if d["family"] != family or (path_re and not path_re.search(d["path"])):
            continue
        for u in d["units"]:
            if u.get("o") in RECORDED and (not field or u["f"] == field):
                yield d["path"] + "|" + u["p"], u


def shortest(examples):
    return [key for _, key in sorted(examples)[:2]]


def count(files, family, label, field, kr, zh, path):
    kr_re = re.compile(kr)
    total = hits = 0
    examples = []
    for key, u in recorded(files, family, field, path):
        m = kr_re.search(u["kr"])
        if not m:
            continue
        pattern = zh
        for name, value in m.groupdict().items():
            pattern = pattern.replace("{" + name + "}", re.escape(value or ""))
        if label == "第二人稱用你" and not re.search("[你妳您]", u.get("zh") or ""):
            continue
        total += 1
        if re.search(pattern, u.get("zh") or ""):
            hits += 1
            examples.append((len(u["kr"]), key))
    return hits, total, examples


def main():
    files = load(sys.argv[1])
    for family, label, field, kr, zh, path in SPECS:
        hits, total, examples = count(files, family, label, field, kr, zh, path)
        show(family, label, hits, total, shortest(examples))
    for name, families, label, field, kr, zh, path in SHARED:
        parts = {f: count(files, f, label, field, kr, zh, path) for f in families}
        hits = sum(p[0] for p in parts.values())
        total = sum(p[1] for p in parts.values())
        examples = [e for p in parts.values() for e in p[2]]
        show("shared/" + name, label, hits, total, shortest(examples))
        print("   " + "  ".join(f"{f} {h}/{t}" + ("" if not t or h / t >= 0.8 else " <80%") for f, (h, t, _) in parts.items()))
    brackets(files)
    for family in LIST_FAMILIES:
        total = hits = 0
        examples = []
        for key, u in recorded(files, family):
            n = len(LIST_MARK.findall(u["kr"]))
            if not n:
                continue
            total += 1
            if n == len(LIST_MARK.findall(u.get("zh") or "")):
                hits += 1
                examples.append((len(key), key))
        show(family, "「- 」條數同 KR", hits, total, shortest(examples))
    for label, kr in LOCATIONS:
        kr_re = re.compile(kr)
        where = Counter(d["family"] for d in files for u in d["units"] if kr_re.search(u["kr"]))
        print(f"位置 {label}: {dict(where)}")


def brackets(files):
    by_family = {}
    half_rule = [0, 0, []]
    for d in files:
        family = d["family"]
        c = by_family.setdefault(family, Counter())
        for u in d["units"]:
            if u.get("o") not in RECORDED:
                continue
            zh = TAG.sub("", u.get("zh") or "")
            half, full = bool(HALF.search(zh)), bool(FULL.search(zh))
            if not (half or full):
                continue
            c["both" if half and full else "half" if half else "full"] += 1
            if family in BATTLE:
                half_rule[1] += 1
                if not FULL.search(ORIGINAL_ZH.sub("", zh)):
                    half_rule[0] += 1
                    half_rule[2].append((len(u["kr"]), d["path"] + "|" + u["p"]))
    for family, c in sorted(by_family.items(), key=lambda x: -sum(x[1].values())):
        print(f"括號 {family:16} 只用半形 {c['half']:5}  只用全形 {c['full']:5}  兩者都有 {c['both']:4}")
    for label, families in (("戰鬥說明類", BATTLE), ("其他族群", [f for f in by_family if f not in BATTLE])):
        c = sum((by_family[f] for f in families), Counter())
        print(f"括號 {label}：只用半形 {c['half']}  只用全形 {c['full']}  兩者都有 {c['both']}  共 {sum(c.values())}")
    show("shared/battle-text", "半形括號", half_rule[0], half_rule[1], shortest(half_rule[2]))
    hits = total = 0
    examples = []
    for family in BATTLE:
        for key, u in recorded(files, family):
            if not ORIGINAL_KR.search(u["kr"]):
                continue
            total += 1
            if ORIGINAL_ZH.search(TAG.sub("", u.get("zh") or "")):
                hits += 1
                examples.append((len(u["kr"]), key))
    show("shared/battle-text", "中文意思（原文）", hits, total, shortest(examples))


def show(family, label, hits, total, examples):
    share = hits / total if total else 0
    flag = "OK" if total >= 30 and share >= 0.8 else "--"
    print(f"{flag} {family:18} {label}: {hits}/{total} ({share:.1%}) {examples}")


if __name__ == "__main__":
    main()
