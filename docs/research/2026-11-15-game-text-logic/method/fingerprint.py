#!/usr/bin/env python3
"""把研究報告依賴的「程式事實」抽成一份指紋，遊戲更新後比對哪些變了。

  RE_WORK=… python3 -I fingerprint.py watch                    從報告裡引用的位址整理出要盯的方法 → watch.json
  RE_WORK=… python3 -I fingerprint.py make <輸出.json>          用 RE_WORK 裡這一版的遊戲程式產生指紋
            python3 -I fingerprint.py diff <舊.json> <新.json>  列出差異與要重看的章節（Markdown）

指紋的內容：
  classes   文字相關類別的欄位（名稱、型別、位移）：TextDataSet 的每張表、檔案清單的每個欄位、每個 TextData_* 類別
  enums     幾個 enum 的成員名稱（依宣告順序；反編譯出來的數值不可靠，不收）
  methods   watch.json 列的每個方法：依序呼叫了哪些有名稱的方法、用到哪些字串常值、
            讀了哪些浮點常數、cmp 指令比較了哪些立即值，以及去掉位址之後的機器碼雜湊

watch 與 make 需要遊戲檔案（見 re_common.py 的說明，先跑過 md_decrypt.py 與 lift.sh decls）；diff 只需要兩份指紋。
watch.json 以方法名稱記錄，所以新版只要重跑 make，不必重跑 watch；報告新增了引用再重跑 watch。

限制：
  - 方法改名或被內嵌就會列在「找不到」，要人工看。
  - 機器碼雜湊會因為別的類別多了欄位（位移改變）而不同；呼叫、字串、常數都沒變時，多半不影響結論。
  - 虛擬呼叫、委派不在「呼叫」裡。
"""
import glob
import hashlib
import json
import os
import re
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式不會自動加入腳本所在目錄

HERE = os.path.dirname(os.path.abspath(__file__))
WATCH = os.path.join(HERE, "watch.json")

# 收進指紋的類別：decls/Assembly-CSharp 底下的相對路徑（正規表示式，比對整個路徑）
TYPE_FILES = [
    r"TextData[^/]*\.cs",
    r"LocalizeTextData[^/]*\.cs",
    r"[^/]*JsonDataList[^/]*\.cs",
    r"AbEvent[^/]*Dlg[^/]*\.cs",
    r"LocalGameOptionData\.cs",
    r"StorySystem/Dialog\.cs",
    r"StorySystem/StoryPlayData\.cs",
    r"RPGSystem/RpgLoc(Text[^/]*|FileList|DomainSet)\.cs",
]
# 收進指紋的 enum（型別名稱）
ENUMS = [
    "BUFF_UNIQUE_KEYWORD", "LOCALIZE_LANGUAGE", "AB_DLG_EVENT_TYPE", "SCENE_STATE", "GROWTH_REWARD_CATEGORY",
    "UNIT_KEYWORD", "STAGE_BATTLE_TYPE", "ATK_BEHAVIOUR", "ATTRIBUTE_TYPE", "SkillAndCoinUIType",
]

TYPE_RE = re.compile(r"^(\s*)(?:public|private|protected|internal)?\s*(?:static |sealed |abstract |partial )*(class|struct|enum|interface) ([A-Za-z_]\w*)(?:<[^>]*>)?(?:\s*:\s*([^{]+?))?\s*$")
FIELD_RE = re.compile(r"^\s*(?:public|private|protected|internal)\s+((?:static |readonly |const )*)(.+?) ([A-Za-z_<][\w<>]*); // (0x[0-9A-Fa-f]+)\s*$")
ENUM_MEMBER_RE = re.compile(r"^\s*([A-Za-z_]\w*)\s*(?:=\s*[^,]+)?,?\s*$")
RVA_RE = re.compile(r"0x([0-9A-Fa-f]{5,8})\b")
# 報告裡寫在反引號中的「類別.方法」：至少兩段，最後一段是方法名（或 .ctor）
NAME_RE = re.compile(r"`((?:[A-Za-z_][\w<>]*\.)+(?:\.ctor|[A-Za-z_<][\w<>]*))(?:\([^`]*\))?`")
# 報告的結論依賴、但內文沒有寫出名稱或位址的方法：「類別.方法」→ 章節
EXTRA = {
    "TextDataSet.LoadRemote": "01", "TextDataSet.LoadLocal": "01", "CustomLocalizeManager.GetConfigFile": "01", "CustomLocalizeManager.GetCandidates": "01", "CustomLocalizeManager.GetLangDataPath": "01",
    "StringUtility.ParseBuffKeywords": "03", "StringUtility.BuildBuffTrie": "03",
    "ChoiceEventLocalizeCheckStaticData.ReplaceText": "04", "TextData_SkillPerLevel.GetDesc": "04",
    "Util.CountMatchingWords_Classification": "04", "TextData_AbnormalityEvents.CheckData": "04",
    "VoiceGenerator.PrintSpeechBubble": "05", "VoiceGenerator.TryPrintCenterSubtitle": "05",
    "VoiceGenerator.AddMultiVoice": "05", "VoiceGenerator.SetMainVoice": "05",
    "BattleDialogUI.ShowDialog": "05", "BattleCenterSubtitleUI.SetText": "05", "BattleCenterSubtitleUI.Show": "05",
    "OutterGradiantEffectController.SetDialog": "05", "ProduceBase_ShowVoice.SetDialog": "05",
    "ProduceBase_ShowVoiceMulti.SetDialog": "05", "UnitScript_1522.UseCenterSubtitle": "05",
    "SkillAndCoinUI.SetCoinText": "07", "TextData_Skill_Coins.GetSummary": "07", "TextData_Skill_Coins.GetDetail": "07",
    "BuffInfo.UpdateString": "07", "TextData_Buf.GetSummary": "07", "TextData_Passive.GetSummary": "07",
    "UnitSkillInfoPassiveSlot.SetPassive": "07", "UnitSkillInfoBufSlot.SetBuf": "07", "UnitSkillInfoEgoGiftSlot.SetEgoGift": "07",
    "BufKeywordUI.SetData": "07", "BuffKeywordToolTipUI.SetData": "07", "IncontrollableUnitStaticData.GetDesc": "07",
    "PersonalityStoryPersonalityUI.SetVoiceListByPath": "07", "PersonalityStoryVoiceSlot.SetData": "07",
    "AbEventCharDlgRoot.GetDlg": "07", "ActionChoiceButtonManager.SetButtonDatas": "07",
    "AbnormalityChoiceDialogController.SetDialogAfterJudgementData": "07", "StageChapterAreaSlot.SetRegionData": "07",
    "DungeonStageInfoUIPanel.GetUIData": "07", "StoryDataParser.GetScenario": "07", "Dialog..ctor": "07",
    "BuffModel.GetDesc": "08", "CountableBuff.GetDesc": "08", "VariableSinBuff.GetDesc": "08",
    "MirrorDungeonStartBuffEffectStaticData.GetDescription": "08", "EachMentalConditionInfo.SetDescription": "08",
    "ChoiceEventResultSet.SetFormatString": "08", "ChoiceEventResultSet.CheckValidParameter": "08",
    "GreetingSentenceStaticDataList.GetSentence": "08", "TMProStringMatcher.GetTagMatchedAbChoiceLogString": "08",
    "MissionStaticData.GetMissionTitle": "08", "CalculationDiff.ExtractString": "08", "CalculationDiff.ExtractValue": "08",
    "PersonalityChoiceResultData..ctor": "08", "ChoiceEventModel.GetBehaviourLog": "08",
    "PersonalityStaticData.GetName": "09", "TextData_Personality.GetTitle_ExceptionNewLine": "09",
    "PersonalityTextDataManager.GetTitleWithName": "09", "BuffModel.GetNameWithStack": "09",
    "BattleUnitView.ViewCancelTextTypo_Lack": "09", "DateUtil.get_TimeZoneOffset": "09",
    "DateUtil.GetRemainTimeStringByKey": "09", "DateUtil.ToString_PeriodBetweenKST": "09",
    "StoryTheaterPersonalityInfo.GetStoryTitle": "09", "UITextMaker.GetResonanceText": "09", "UITextMaker.GetRequirementText": "09",
    "UnitInformationSynergyBox.BuildName": "09", "RateInfoItem.BuildName": "09", "UnitLevelTooltipData..ctor": "09",
    "BattleChoiceSelectionOmenUI.SetText": "09", "StoryTheaterDanteNoteCategoryButton.SetFont": "09",
    "GacksungLevelUpCompletionPopup.GetTextForBasicPersonalities": "09", "DantesNoteDetailInfoData.GetTitleText": "09",
}
# enum 變了要重看的章節
ENUM_CHAPTERS = {
    "BUFF_UNIQUE_KEYWORD": ["03"], "LOCALIZE_LANGUAGE": ["04", "09"], "AB_DLG_EVENT_TYPE": ["07"], "SCENE_STATE": ["05"],
    "GROWTH_REWARD_CATEGORY": ["08"], "UNIT_KEYWORD": ["08"], "STAGE_BATTLE_TYPE": ["07"], "ATK_BEHAVIOUR": ["08"],
    "ATTRIBUTE_TYPE": ["08"], "SkillAndCoinUIType": ["07"],
}
# 同名的多載超過這個數目就不收（多半是到處都有的 SetData、Init 之類）
MAX_OVERLOADS = 6


def parse_types(path):
    """一個 .cs 檔 → {型別的完整名稱: {"kind", "base", "fields" 或 "members"}}。巢狀型別用「外層.內層」。"""
    out, stack = {}, []  # stack: (縮排寬度, 名稱)
    for line in open(path, encoding="utf-8", errors="replace"):
        m = TYPE_RE.match(line.rstrip("\n"))
        if m:
            indent = len(m.group(1))
            while stack and stack[-1][0] >= indent:
                stack.pop()
            name = ".".join([s[1] for s in stack] + [m.group(3)])
            stack.append((indent, name.split(".")[-1]))
            entry = {"kind": m.group(2)}
            if m.group(4):
                entry["base"] = m.group(4).split(",")[0].strip()
            entry["members" if m.group(2) == "enum" else "fields"] = []
            out[name] = entry
            continue
        if not stack:
            continue
        indent = len(line) - len(line.lstrip(" "))
        while stack and stack[-1][0] >= indent and line.strip() not in ("{", "}", ""):
            stack.pop()
        if not stack:
            continue
        cur = out[".".join(s[1] for s in stack)]
        if cur["kind"] == "enum":
            m = ENUM_MEMBER_RE.match(line.rstrip("\n"))
            if m and line.strip() not in ("{", "}"):
                cur["members"].append(m.group(1))
        else:
            m = FIELD_RE.match(line.rstrip("\n"))
            if m and "static" not in m.group(1) and "const" not in m.group(1):
                cur["fields"].append([m.group(3), m.group(2), m.group(4).lower()])
    return out


def collect_types(decls):
    root = os.path.join(decls, "Assembly-CSharp")
    pats = [re.compile(p + r"\Z") for p in TYPE_FILES]
    classes, enums = {}, {}
    for path in sorted(glob.glob(os.path.join(root, "**", "*.cs"), recursive=True)):
        rel = os.path.relpath(path, root).replace(os.sep, "/")
        base = os.path.basename(rel)[:-3]
        want_enum = base in ENUMS
        if not want_enum and not any(p.match(rel) for p in pats):
            continue
        for name, t in parse_types(path).items():
            if t["kind"] == "enum":
                if name.split(".")[-1] in ENUMS:
                    enums[name] = t["members"]
            elif not want_enum and (t["fields"] or t.get("base")):
                classes[name] = {k: v for k, v in t.items() if k != "kind"}
    return classes, enums


class Binary:
    def __init__(self):
        from re_common import PE, Owners, load_calls, DECLS
        self.pe = PE()
        self.names = load_calls(self.pe)[0]
        self.own = Owners(self.names)
        self.by_name = {}
        for rva, ns in self.names.items():
            for n in ns:
                self.by_name.setdefault(n, rva)
        self.literals = {x["index"]: x["value"] for x in json.load(open(os.path.join(DECLS, "stringliteral.json"), encoding="utf-8"))}
        self.code = [(va, va + max(vs, rs)) for name, va, vs, ro, rs in self.pe.secs if name in (".text", "il2cpp")]

    def in_code(self, rva):
        return any(lo <= rva < hi for lo, hi in self.code)

    def read(self, rva, n):
        """檔案裡沒有內容的位址（執行時才配置的資料）回傳 None。"""
        return self.pe.read(rva, n) if self.pe.rva2off(rva) is not None else None

    def literal_at(self, rva):
        """rip 相對位址指到的 8 bytes 若是字串常值的 metadata usage（(5 << 29) | (編號 << 1) | 1），回傳那個字串。"""
        raw = self.read(rva, 8)
        if raw is None or len(raw) < 8:
            return None
        v = struct.unpack("<Q", raw)[0]
        if v >> 32 == 0 and (v >> 29) & 7 == 5 and v & 1:
            return self.literals.get((v >> 1) & 0x0FFFFFFF)
        return None

    def digest(self, start):
        from iced_x86 import Decoder, Mnemonic, OpKind, Register
        end = min(self.own.end(start), start + 0x20000)
        float_ops = {getattr(Mnemonic, n) for n in dir(Mnemonic) if n.endswith(("SS", "SD")) and not n.startswith("_") and n not in ("MOVS", "CMPS")}
        imm_kinds = {OpKind.IMMEDIATE8, OpKind.IMMEDIATE16, OpKind.IMMEDIATE32, OpKind.IMMEDIATE64, OpKind.IMMEDIATE8TO16,
                     OpKind.IMMEDIATE8TO32, OpKind.IMMEDIATE8TO64, OpKind.IMMEDIATE32TO64}
        calls, strings, floats, ints = [], [], [], []
        h = hashlib.sha1()
        for ins in Decoder(64, bytes(self.pe.read(start, end - start)), ip=self.pe.base + start):
            if ins.mnemonic == Mnemonic.INVALID:
                break
            part = [ins.mnemonic, ins.op_count]
            for i in range(ins.op_count):
                k = ins.op_kind(i)
                part.append(k)
                if k in imm_kinds:
                    v = ins.immediate(i)
                    part.append(v)
                    if ins.mnemonic == Mnemonic.CMP and v not in ints:
                        ints.append(v)
            if ins.is_ip_rel_memory_operand:
                t = ins.ip_rel_memory_address - self.pe.base
                if ins.mnemonic in float_ops:
                    name = Mnemonic.__dict__ and [n for n in dir(Mnemonic) if getattr(Mnemonic, n) == ins.mnemonic][0]
                    raw = self.read(t, 8 if name.endswith("SD") else 4)
                    if raw is not None and len(raw) in (4, 8):
                        v = round(struct.unpack("<d" if len(raw) == 8 else "<f", raw)[0], 6)
                        if v not in floats:
                            floats.append(v)
                        part.append(v)
                else:
                    s = self.literal_at(t)
                    if s is not None:
                        if s not in strings:
                            strings.append(s)
                        part.append(s)
            elif ins.op_count and ins.op_kind(0) != OpKind.NEAR_BRANCH64 and ins.memory_base not in (Register.NONE, Register.RIP):
                part.append(ins.memory_displacement)
            try:
                g = ins.near_branch_target - self.pe.base
            except Exception:
                g = None
            if g is not None and ins.mnemonic in (Mnemonic.CALL, Mnemonic.JMP):
                ns = self.names.get(g)
                if ns and g != start:
                    if not calls or calls[-1] != ns[0]:
                        calls.append(ns[0])
                    part.append(ns[0])
            h.update(repr(part).encode())
        return {"calls": calls, "strings": strings, "floats": floats, "ints": ints, "code": h.hexdigest()[:12], "size": end - start}


def cmd_watch():
    b = Binary()
    # 「類別.方法」（不含參數）→ 方法名稱；同名的多載都收
    by_tail = {}
    for name in b.by_name:
        bare = name.split("(")[0]
        parts = bare.split(".")
        if bare.endswith("..ctor"):
            parts = bare[:-6].split(".") + [".ctor"]
        for n in range(2, len(parts) + 1):
            key = ".".join(parts[-n:]).replace("..ctor", ".ctor")
            by_tail.setdefault(key, []).append(name)
    found = {}
    for md in sorted(glob.glob(os.path.join(HERE, "..", "*.md"))):
        chapter = os.path.basename(md)[:2]
        if not chapter.isdigit():
            continue
        text = open(md, encoding="utf-8").read()
        # 報告裡引用的位址
        for m in RVA_RE.finditer(text):
            rva = int(m.group(1), 16)
            if not b.in_code(rva):
                continue
            start = b.own.start(rva)
            if start is None or rva >= b.own.end(start) or start not in b.names:
                continue
            found.setdefault(b.names[start][0], set()).add(chapter)
        # 報告裡以「類別.方法」寫出的名稱
        for m in NAME_RE.finditer(text):
            key = m.group(1).replace("..ctor", ".ctor")
            hits = by_tail.get(key, [])
            if 0 < len(hits) <= MAX_OVERLOADS:
                for name in hits:
                    found.setdefault(name, set()).add(chapter)
    unresolved = []
    for key, chapter in EXTRA.items():
        hits = by_tail.get(key.replace("..ctor", ".ctor"), [])
        if not hits:
            unresolved.append(key)
        for name in hits[:MAX_OVERLOADS]:
            found.setdefault(name, set()).add(chapter)
    if unresolved:
        print("EXTRA 裡找不到的名稱（改名或刪除了？）：%s" % "、".join(unresolved), file=sys.stderr)
    with open(WATCH, "w", encoding="utf-8") as f:
        f.write('{"methods": {\n')
        f.write(",\n".join("%s: %s" % (json.dumps(n, ensure_ascii=False), json.dumps(sorted(c))) for n, c in sorted(found.items())))
        f.write("\n}}\n")
    print("%d 個方法 → %s" % (len(found), WATCH), file=sys.stderr)


def cmd_make(out):
    from re_common import DECLS, DLL
    watch = json.load(open(WATCH, encoding="utf-8"))["methods"]
    b = Binary()
    classes, enums = collect_types(DECLS)
    methods, missing = {}, []
    for name, chapters in sorted(watch.items()):
        rva = b.by_name.get(name)
        if rva is None:
            missing.append(name)
            continue
        d = b.digest(rva)
        d["chapters"] = chapters
        d["shared"] = len(b.names[rva])
        methods[name] = d
    pe_off = struct.unpack_from("<I", b.pe.d, 0x3C)[0]
    fp = {
        "build": {"dll_size": os.path.getsize(DLL), "pe_timestamp": struct.unpack_from("<I", b.pe.d, pe_off + 8)[0], "methods_named": len(b.by_name)},
        "classes": classes, "enums": enums, "methods": methods, "missing": missing,
    }
    with open(out, "w", encoding="utf-8") as f:
        f.write("{\n")
        f.write('"build": %s,\n' % json.dumps(fp["build"], sort_keys=True))
        for key in ("classes", "enums", "methods"):
            f.write('"%s": {\n' % key)
            f.write(",\n".join("%s: %s" % (json.dumps(k, ensure_ascii=False), json.dumps(v, ensure_ascii=False, sort_keys=True)) for k, v in sorted(fp[key].items())))
            f.write("\n},\n")
        f.write('"missing": %s\n}\n' % json.dumps(missing, ensure_ascii=False))
    print("%d 個類別、%d 個 enum、%d 個方法（%d 個找不到）→ %s" % (len(classes), len(enums), len(methods), len(missing), out), file=sys.stderr)


def show_int(v):
    if 0x20 < v < 0x110000 and not 0xD800 <= v < 0xE000 and (v > 0x7E or chr(v).isalnum() is False) and v > 0xFF:
        return "0x%X「%s」" % (v, chr(v))
    return str(v) if v < 256 else "0x%X" % v


def list_diff(a, b):
    """回傳 (b 多出來的, b 少掉的)，保留順序。"""
    return [x for x in b if x not in a], [x for x in a if x not in b]


def cmd_diff(old_path, new_path):
    old, new = (json.load(open(p, encoding="utf-8")) for p in (old_path, new_path))
    out, chapters = [], set()

    def section(title, lines):
        if lines:
            out.append("## " + title + "\n")
            out.extend(lines)
            out.append("")

    lines = []
    for name in sorted(set(old["classes"]) | set(new["classes"])):
        a, b = old["classes"].get(name), new["classes"].get(name)
        if a is None:
            lines.append("- 新類別 `%s`：%s" % (name, "、".join("`%s`" % f[0] for f in b["fields"]) or "（沒有欄位）"))
        elif b is None:
            lines.append("- 類別不見了：`%s`" % name)
        else:
            fa, fb = {f[0]: f for f in a["fields"]}, {f[0]: f for f in b["fields"]}
            added, removed = [n for n in fb if n not in fa], [n for n in fa if n not in fb]
            changed = [n for n in fb if n in fa and fb[n][1] != fa[n][1]]
            moved = [n for n in fb if n in fa and fb[n][1] == fa[n][1] and fb[n][2] != fa[n][2]]
            bits = []
            if added:
                bits.append("多了 " + "、".join("`%s`（%s）" % (n, fb[n][1]) for n in added))
            if removed:
                bits.append("少了 " + "、".join("`%s`" % n for n in removed))
            if changed:
                bits.append("型別變了 " + "、".join("`%s`（%s → %s）" % (n, fa[n][1], fb[n][1]) for n in changed))
            if a.get("base") != b.get("base"):
                bits.append("基底類別 %s → %s" % (a.get("base"), b.get("base")))
            if bits:
                lines.append("- `%s`：%s" % (name, "；".join(bits)))
            elif moved:
                lines.append("- `%s`：欄位相同，位移變了（%d 個）" % (name, len(moved)))
    if any("位移變了" not in l for l in lines):
        # 表與檔案清單 → 載入流程與族群；條目的欄位 → 各欄位的顯示
        chapters.update(["01", "06"] if any("`TextDataSet" in l for l in lines) else [])
        chapters.add("07")
    section("類別的欄位（表、檔案清單、各種文字條目）", lines)

    lines = []
    for name in sorted(set(old["enums"]) | set(new["enums"])):
        a, b = old["enums"].get(name, []), new["enums"].get(name, [])
        added, removed = list_diff(a, b)
        if added or removed:
            cap = lambda xs: "、".join("`%s`" % x for x in xs[:40]) + ("……（共 %d 個）" % len(xs) if len(xs) > 40 else "")
            lines.append("- `%s`（%d → %d 個）%s%s" % (name, len(a), len(b), "：多了 " + cap(added) if added else "", "；少了 " + cap(removed) if removed else ""))
        elif a != b:
            lines.append("- `%s`：成員相同，順序變了" % name)
        if added or removed:
            chapters.update(ENUM_CHAPTERS.get(name.split(".")[-1], []))
            if name.split(".")[-1] == "BUFF_UNIQUE_KEYWORD":
                lines.append("  - 這是會自動連結的狀態 id 名單：`kwsim.py` 的 `NOT_IN_ENUM` 要對照新的 `Bufs*` 重算")
    section("enum 的成員", lines)

    behaviour, code_only, gone, fresh = [], [], [], []
    for name in sorted(set(old["methods"]) | set(new["methods"])):
        a, b = old["methods"].get(name), new["methods"].get(name)
        if b is None:
            gone.append("- `%s`（第 %s 章）" % (name, "、".join(a["chapters"])))
            chapters.update(a["chapters"])
            continue
        if a is None:
            fresh.append("- `%s`（第 %s 章）" % (name, "、".join(b["chapters"])))
            continue
        bits = []
        for key, label, fmt in (("calls", "呼叫", lambda x: "`%s`" % x), ("strings", "字串", lambda x: "`%s`" % x.replace("\n", "\\n")),
                                ("floats", "浮點常數", str), ("ints", "比較的數值", show_int)):
            added, removed = list_diff(a[key], b[key])
            if added:
                bits.append("%s多了 %s" % (label, "、".join(fmt(x) for x in added[:12]) + ("……" if len(added) > 12 else "")))
            if removed:
                bits.append("%s少了 %s" % (label, "、".join(fmt(x) for x in removed[:12]) + ("……" if len(removed) > 12 else "")))
            if not added and not removed and a[key] != b[key]:
                bits.append("%s的順序變了" % label)
        if bits:
            behaviour.append("- `%s`（第 %s 章）：%s" % (name, "、".join(b["chapters"]), "；".join(bits)))
            chapters.update(b["chapters"])
        elif a["code"] != b["code"]:
            code_only.append("- `%s`（第 %s 章，%d → %d bytes）" % (name, "、".join(b["chapters"]), a["size"], b["size"]))
    section("方法：行為有變（呼叫、字串或常數不同）", behaviour)
    section("方法：找不到了（改名、刪除或被內嵌）", gone)
    section("方法：只有機器碼不同（呼叫、字串、常數都相同）", code_only)
    section("方法：新加入盯的", fresh)

    print("# 遊戲程式的差異\n")
    print("舊：%s（%s）\n新：%s（%s）\n" % (old_path, json.dumps(old["build"], sort_keys=True), new_path, json.dumps(new["build"], sort_keys=True)))
    if not out:
        print("沒有差異。")
        return 0
    if chapters:
        print("要重看的章節：%s\n" % "、".join(sorted(chapters)))
    print("\n".join(out))
    return 1


if __name__ == "__main__":
    a = sys.argv[1:]
    if a == ["watch"]:
        cmd_watch()
    elif len(a) == 2 and a[0] == "make":
        cmd_make(a[1])
    elif len(a) == 3 and a[0] == "diff":
        sys.exit(cmd_diff(a[1], a[2]))
    else:
        sys.exit(__doc__)
