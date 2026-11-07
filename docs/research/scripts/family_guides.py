"""guides/families/*.md 統計數字的重算腳本。

用法：python3 -I family_guides.py DIR
DIR 是 `lct site --out DIR` 寫出的網站資料（含 files/<路徑>，每檔有 family 與 units）。

「既有 a／b」：該族群 KR 符合條件、來源是 paratranz／claude／human（不含 copy）的 b 個單位中，
a 個的譯文符合寫法。只有 a/b >= 80% 且 b >= 30 的才寫進指引（印出時標 OK）。
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
    ("ally", "(턴당 N회)", "", r"\(턴당 (?P<n>\d+)회\)", r"\(每回合最多{n}次\)", ""),
    ("ally", "피해량 +N%", "", r"(?<!받는 )피해량 \+(?P<n>\d+)%", r"造成的傷害\+{n}%", ""),
    ("ally", "최종 위력", "", r"최종 위력 (?P<s>[+-])(?P<n>\d+)", r"最終威力{s}{n}", ""),
    ("ally", "정신력 N 회복", "", r"정신력 (?P<n>\d+) 회복", r"恢復{n}點理智值", ""),
    ("ally", "체력 N 회복", "", r"체력 (?P<n>\d+) 회복", r"恢復{n}點體力", ""),
    ("ally", "아군", "", ALLY, r"友方單位", ""),
    ("ally", "처치 시", "", r"처치 시", r"擊殺.*(時|後)", ""),
    ("enemy", "[X] N 부여", "desc", r"\[" + STATUS + r"\] (?P<n>\d+) 부여", r"使目標增加{n}級 \[{x}\] 強度", ""),
    ("enemy", "(턴당 N회)", "", r"\(턴당 (?P<n>\d+)회\)", r"\(每回合最多{n}次\)", ""),
    ("enemy", "피해량 +N%", "", r"(?<!받는 )피해량 \+(?P<n>\d+)%", r"造成的傷害\+{n}%", ""),
    ("enemy", "최종 위력", "", r"최종 위력 (?P<s>[+-])(?P<n>\d+)", r"最終威力{s}{n}", ""),
    ("enemy", "정신력 N 회복", "", r"정신력 (?P<n>\d+) 회복", r"恢復{n}點理智值", ""),
    ("enemy", "체력 N 회복", "", r"체력 (?P<n>\d+) 회복", r"恢復{n}點體力", ""),
    ("enemy", "아군", "", ALLY, r"友方單位", ""),
    ("keywords", "- 패닉", "", r"^- 패닉\n", r"^- 陷入恐慌\n", ""),
    ("keywords", "- 최댓값: N", "", r"(^|\n)- 최댓값 ?: ?(?P<n>\d+)", r"(^|\n)- 最大值：{n}", ""),
    ("keywords", "수치 N당", "", r"수치 (?P<n>\d+)당", r"每(帶)?有{n}層", ""),
    ("keywords", "(턴당 N회)", "", r"\(턴당 (?P<n>\d+)회\)", r"\(每回合最多{n}次\)", ""),
    ("keywords", "공격 레벨", "", r"공격 레벨 (?P<s>[+-])(?P<n>\d+)", r"攻擊等級{s}{n}", ""),
    ("keywords", "최종 위력", "", r"최종 위력 (?P<s>[+-])(?P<n>\d+)", r"最終威力{s}{n}", ""),
    ("keywords", "정신력 N 회복", "", r"정신력 (?P<n>\d+) 회복", r"恢復{n}點理智值", ""),
    ("keywords", "적중 시", "", r"적중 시", r"命中時", ""),
    ("keywords", "처치 시", "", r"처치 시", r"擊殺.*時", ""),
    ("keywords", "아군", "", ALLY, r"友方單位", ""),
    ("keywords", "[韓文]", "", BRACKET_KR, BRACKET_ZH, ""),
    ("mirror", "선택 시,", "messageDesc", r"^선택 시, ", r"^選擇後，", ""),
    ("mirror", "E.G.O GIFT 획득", "messageDesc", r"(?<![0-9] )E\.G\.O GIFT 획득", r"獲得1件", ""),
    ("mirror", "별도 효과 없음", "messageDesc", r"^별도 효과 없음$", r"^沒有效果$", ""),
    ("mirror", "판정 성공 시", "", r"판정 성공 시", r"判定 成功後", ""),
    ("mirror", "피해량 +N%", "", r"(?<!받는 )피해량 \+(?P<n>\d+)%", r"造成的傷害\+{n}%", ""),
    ("mirror", "최종 위력", "", r"최종 위력 (?P<s>[+-])(?P<n>\d+)", r"最終威力{s}{n}", ""),
    ("mirror", "아군", "", ALLY, r"友方單位", ""),
    ("mirror", "[韓文]", "", BRACKET_KR, BRACKET_ZH, ""),
    ("railway", "제 N 역", "", r"제 ?(?P<n>\d+) ?역", r"第{n}站", ""),
    ("railway", "제 N 구간", "", r"제 ?(?P<n>\d+) ?구간", r"第{n}區段", ""),
    ("railway", "N호선", "", r"(?P<n>\d+)호선", r"{n}號線", ""),
    ("railway", "굴절률", "", r"굴절률 ?: ?(?P<n>\d+)%", r"折射率：{n}%", ""),
    ("railway", "클리어", "", r"클리어", r"通關", ""),
    ("stage", "클리어", "", r"클리어", r"通關", ""),
    ("stage", "[韓文]", "", BRACKET_KR, BRACKET_ZH, ""),
    ("events", "[韓文]", "", BRACKET_KR, BRACKET_ZH, ""),
    ("ui", " x N", "", r" x (?P<n>\d+)", r" × {n}", ""),
    ("ui", "[韓文]", "", BRACKET_KR, BRACKET_ZH, ""),
    ("abnormality", "[{n}] 수감자", "", r"\[\{(?P<n>\d)\}\] 수감자", r"罪人\[\{{n}\}\]", ""),
    ("abnormality", "어떤／어느 수감자", "behaveDesc", r"^(어떤|어느) 수감자", r"哪位罪人", ""),
    ("identity", "흐트러질 시 대사", "", r"^흐트러질 시 대사$", r"^自身混亂$", r"^PersonalityVoiceDlg/"),
    ("identity", "아군 사망", "", r"^아군 사망$", r"^友方陣亡$", r"^PersonalityVoiceDlg/"),
    ("ego-voice", "E.G.O 침식 발동", "desc", r"^E\.G\.O 침식 발동$", r"^E\.G\.O侵蝕$", ""),
    # 第二人稱：譯文有「你／妳」而沒有「您」的，占有第二人稱的單位多少
    ("ego-voice", "第二人稱用你", "dlg", r".", r"(?s)^(?=.*[你妳])(?!.*您)", ""),
]

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


def main():
    files = load(sys.argv[1])
    for family, label, field, kr, zh, path in SPECS:
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
        show(family, label, hits, total, shortest(examples))
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


def show(family, label, hits, total, examples):
    share = hits / total if total else 0
    flag = "OK" if total >= 30 and share >= 0.8 else "--"
    print(f"{flag} {family:12} {label}: {hits}/{total} ({share:.1%}) {examples}")


if __name__ == "__main__":
    main()
