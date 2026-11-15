#!/usr/bin/env python3
"""重現遊戲的「關鍵字自動連結」判斷（依 2026-11-12 版 GameAssembly.dll 反編譯結果）。

對應的遊戲程式：
  UI.Utility.TMProStringMatcher.SetSkillKeywords                      關鍵字名稱表
  StringUtility.BuildBuffTrie / ParseBuffKeywords                      名稱比對（Aho-Corasick）
  UI.Utility.TMProStringMatcher.GetApplySpriteAndLinkAndColorString    後接字元規則、共鳴例外
  UI.Utility.TMProStringMatcher.GetSkillTagMatchedString               [標籤] 的處理（只重現對比對有影響的部分）

用法：
  python3 -I kwsim.py link  <語言資料夾> <檔名前綴> '文字'        列出這段文字裡會被連結的名稱
  python3 -I kwsim.py diff  <KR資料夾> <譯文資料夾>               逐欄位比較 KR 與譯文產生的連結

  例：python3 -I kwsim.py link .work/tw/LimbusCompany_Data/Lang/zh-TW '' '施加1～2層易損 '
      python3 -I kwsim.py diff .work/raw/kr .work/tw/LimbusCompany_Data/Lang/zh-TW

限制：
  - 名稱表以「Bufs*.json 裡有 name 的 id」近似。遊戲實際只收 enum BUFF_UNIQUE_KEYWORD 的成員；
    2026-11-12 版有 54 個 Bufs id 不是成員（見 NOT_IN_ENUM），遊戲更新後這份名單需要重查。
  - 同長度、互相重疊的名稱，遊戲用不穩定排序決定先後，這裡以出現位置先後決定。
  - 只算「純文字名稱」的自動連結與 [ID]；不模擬 TextMeshPro 的顯示。
"""
import glob
import json
import os
import re
import sys
import unicodedata

# 名稱後面緊接這些字元（或空白、數字、文字結尾）才會連結
ALLOWED_AFTER = "・「」『』.,/&;"
ZWSP = "​"

# 2026-11-12 版：Bufs*.json 有、但不是 BUFF_UNIQUE_KEYWORD 成員的 id（不進名稱表）
NOT_IN_ENUM = set("""
ATL_EndureCombustion BlackWingHoleFaust Blame_LowMorale Blame_Panic BulletFreischutz DamageOverTime
DuelDeclaration DuelDeclarationMeursault Emergency_Operation FamineBloodPriestServant_LowMorale
FamineBloodPriestServant_Panic GreedyReady Hardening Lensman_LowMorale Liu_Meursault_Guard_Buff MoonLight
Perfected PseudoLastBoss_KingCrap RemoveBreakSection SelfHarm_LowMorale SelfHarm_Panic SinkingBlack
StickCoinHeadAndTailToMaxPer Target Taunt TimesCoinValueDown TimesCoinValueUp UnfinishedDreamSanchoMirror
""".split()) | {"RailBuffForLine2_%s_ALLY" % c for c in "ABCDEFGHIJKLMNOPQRST"} \
             | {"RailBuffForLine2_%s_ENEMY" % c for c in "ABCDEF"}


def load(path):
    with open(path, encoding="utf-8-sig") as f:
        return json.load(f)


def is_white_space(ch):
    """System.Char.IsWhiteSpace"""
    return ch in " \t\n\v\f\r\x85\xa0     　" or " " <= ch <= " "


def is_digit(ch):
    """System.Char.IsDigit：Unicode 類別 Nd，全形數字也算"""
    return unicodedata.category(ch) == "Nd"


class Matcher:
    def __init__(self, lang_dir, prefix=""):
        self.buf = {}  # id -> 條目（先出現的優先）
        for f in sorted(glob.glob(os.path.join(lang_dir, prefix + "Bufs*.json"))):
            for d in load(f).get("dataList", []):
                if "id" in d:
                    self.buf.setdefault(str(d["id"]), d)
        self.names = {}    # 名稱 -> id（_skillKeywords；同名先到先得）
        self.skip = set()  # excludeAutoLink 的名稱：不進 Trie
        for i, d in self.buf.items():
            n = d.get("name")
            if i in NOT_IN_ENUM or not n or n.upper() == "UNKNOWN":
                continue
            if d.get("excludeAutoLink"):
                self.skip.add(n)
            self.names.setdefault(n, i)
        self.index = {}
        for n in self.names:
            if n not in self.skip:
                self.index.setdefault(n[0], []).append(n)
        self.resonance = self.perfect = None
        ui = os.path.join(lang_dir, prefix + "BattleUIText.json")
        if os.path.exists(ui):
            for d in load(ui).get("dataList", []):
                if d.get("id") == "resonance":
                    self.resonance = d.get("content")
                elif d.get("id") == "perfect_resonance":
                    self.perfect = d.get("content")

    def candidates(self, text):
        """StringUtility.ParseBuffKeywords：全部出現位置 → 長的優先 → 不重疊 → 緊鄰 U+200B 的跳過"""
        occ = [(s, w) for s, ch in enumerate(text) for w in self.index.get(ch, ()) if text.startswith(w, s)]
        occ.sort(key=lambda t: -len(t[1]))
        used, out = set(), []
        for s, w in occ:
            e = s + len(w)
            if any(i in used for i in range(s, e)):
                continue
            if (s > 0 and text[s - 1] == ZWSP) or (e < len(text) and text[e] == ZWSP):
                continue
            out.append((s, w))
            used.update(range(s, e))
        return sorted(out)

    def linked(self, text):
        """回傳 [(起點, 名稱, id)]：GetApplySpriteAndLinkAndColorString 會加上圖示與連結的純文字名稱"""
        res = []
        for s, w in self.candidates(text):
            e = s + len(w)
            rest = text[e:]

            def longer(n):
                return len(text) > e + n

            if self.resonance and longer(len(self.resonance) + 1) and self.resonance in rest[:len(self.resonance) + 1]:
                continue
            if self.perfect and longer(len(self.perfect) + 2) and self.perfect in rest[:len(self.perfect) + 2].replace(" ", ""):
                continue
            if longer(4) and "속성" in rest[:4]:
                continue
            if longer(6) and "Abs." in rest[:6]:
                continue
            if e >= len(text) or is_white_space(text[e]) or is_digit(text[e]) or text[e] in ALLOWED_AFTER:
                res.append((s, w, self.names[w]))
        return res

    def skill_tag_pass(self, text):
        """GetSkillTagMatchedString 對比對的影響：不是狀態 id 的 [xxx] 會被換成
        <color=…>標籤文字</color>，所以它前面的名稱後接字元變成「<」。這裡只把它換成「<>」。"""
        return re.sub(r"\[(.*?)\]", lambda m: m.group(0) if m.group(1) in self.buf else "<>", text)

    def link_ids(self, text, skill_tag=True):
        """這段文字最後會出現的連結 id：純文字名稱 + [ID]"""
        t = self.skill_tag_pass(text) if skill_tag else text
        ids = [i for _, _, i in self.linked(t)]
        ids += [m for m in re.findall(r"\[(.*?)\]", t) if m in self.buf]
        return ids


def units(path):
    out = {}

    def walk(o, p):
        if isinstance(o, dict):
            for k, v in o.items():
                walk(v, p + "/" + k)
        elif isinstance(o, list):
            for i, v in enumerate(o):
                walk(v, p + "/%d" % i)
        elif isinstance(o, str):
            out[p] = o

    walk(load(path), "")
    return out


# 會經過關鍵字處理的檔案與欄位（見報告 03-keywords.md 的表）
FILES = re.compile(r"^(Skills|Passive|EGOgift)")
FIELDS = re.compile(r"/(desc|summary|simpleDesc)$")


def diff(kr_dir, tw_dir):
    kr, tw = Matcher(kr_dir, "KR_"), Matcher(tw_dir, "")
    n = extra = missing = 0
    for f in sorted(glob.glob(os.path.join(tw_dir, "*.json"))):
        b = os.path.basename(f)
        kf = os.path.join(kr_dir, "KR_" + b)
        if not FILES.match(b) or not os.path.exists(kf):
            continue
        ku = units(kf)
        for p, t in units(f).items():
            k = ku.get(p)
            if k is None or not FIELDS.search(p):
                continue
            n += 1
            ki, ti = set(kr.link_ids(k)), set(tw.link_ids(t))
            for i in sorted(ti - ki):
                extra += 1
                print("多出\t%s|%s\t%s\t%s" % (b, p, i, tw.buf[i].get("name")))
            for i in sorted(ki - ti):
                missing += 1
                print("缺少\t%s|%s\t%s\t%s" % (b, p, i, kr.buf[i].get("name")))
    print("# %d 個欄位；譯文多出 %d 個連結、缺少 %d 個連結" % (n, extra, missing), file=sys.stderr)


if __name__ == "__main__":
    if len(sys.argv) == 5 and sys.argv[1] == "link":
        m = Matcher(sys.argv[2], sys.argv[3])
        for s, w, i in m.linked(m.skill_tag_pass(sys.argv[4])):
            print(s, w, i)
    elif len(sys.argv) == 4 and sys.argv[1] == "diff":
        diff(sys.argv[2], sys.argv[3])
    else:
        sys.exit(__doc__)
