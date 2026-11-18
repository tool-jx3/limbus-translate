#!/usr/bin/env python3
"""官方檔案清單（RemoteLocalizeFileList.json）與語言包的檔案互相比對。

  python3 -I filelist.py <RemoteLocalizeFileList.json> <語言包資料夾> [前綴]

語言包資料夾是 raw 分支的 kr/、en/、jp/（前綴 KR_、EN_、JP_，預設 KR_），或譯文的 Lang/zh-TW（前綴給空字串）。
印出：清單的大小、重複列出的檔名、清單有而資料夾沒有的檔、資料夾有而清單沒有的檔（遊戲不會載入）、
同一張表裡不同檔案的重複 id（語言包以清單裡排在後面的檔為準）。

清單是遊戲下載的資料，不在版控裡：它在遊戲下載的在地化資料夾裡，和 kr、en、jp、etc 同一層。
"""
import collections
import json
import os
import sys

# 這四張表的檔案在子資料夾裡；劇情與 RPG 由各自的載入器讀，不經過這份清單。
SUBDIR = {"personalityVoice": "PersonalityVoiceDlg/", "announcerVoice": "BattleAnnouncerDlg/", "egoVoice": "EGOVoiceDig/", "bgmLyrics": "BgmLyrics/"}
OTHER_LOADERS = ("StoryData/", "RPGSystem/")
# 啟動時就要用的表：檔案清單是遊戲本體 Resources 裡的 Localize/LocalizeFileList，不在這份清單裡。
STARTUP = {"BattleHint", "FileDownloadDesc", "LoginUIText", "UserAgreements", "ErrorCodeMsgLocal"}
# 整個檔取代、不以 id 合併的表
WHOLE_FILE = set(SUBDIR) | {"abnormalityCharDlgs"}


def pack_files(root, prefix):
    out = set()
    for d, _, names in os.walk(root):
        for n in names:
            if n.endswith(".json") and n.startswith(prefix):
                out.add(os.path.relpath(os.path.join(d, n[len(prefix):-5]), root).replace(os.sep, "/"))
    return out


def entries(root, prefix, rel):
    d, b = os.path.split(rel)
    try:
        with open(os.path.join(root, d, prefix + b + ".json"), encoding="utf-8-sig") as f:
            doc = json.load(f)
    except FileNotFoundError:
        return None
    return doc.get("dataList") or [] if isinstance(doc, dict) else []


def main():
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    with open(sys.argv[1], encoding="utf-8-sig") as f:
        lists = json.load(f)
    root, prefix = sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else "KR_"
    have = pack_files(root, prefix)

    listed = collections.defaultdict(list)
    for name, files in lists.items():
        for f in files:
            listed[SUBDIR.get(name, "") + f].append(name)
    total = sum(len(v) for v in lists.values())
    print(f"清單：{len(lists)} 張表、{total} 個檔名（不重複 {len(listed)} 個）；資料夾：{len(have)} 個檔")

    print("\n## 列出不只一次的檔名")
    for f, names in sorted(listed.items()):
        if len(names) > 1:
            print(f"  {f}: {', '.join(names)}")

    missing = sorted(f for f in listed if f not in have)
    print(f"\n## 清單有、資料夾沒有（{len(missing)}）")
    for f in missing:
        print(f"  {f}（{', '.join(listed[f])}）")

    rest = sorted(f for f in have if f not in listed)
    other = collections.Counter(f.split("/")[0] + "/" for f in rest if f.startswith(OTHER_LOADERS))
    startup = [f for f in rest if f in STARTUP]
    unloaded = [f for f in rest if not f.startswith(OTHER_LOADERS) and f not in STARTUP]
    print(f"\n## 資料夾有、清單沒有（{len(rest)}）")
    for d, n in sorted(other.items()):
        print(f"  {d} {n} 個檔：由另外的載入器讀")
    print(f"  啟動用的表（另一份清單）：{', '.join(startup) or '無'}")
    print(f"  其餘 {len(unloaded)} 個不會被載入：")
    for f in unloaded:
        items = entries(root, prefix, f) or []
        print(f"    {f}（{len(items)} 筆）")

    print("\n## 同一張表裡不同檔案的重複 id")
    found = False
    for name, files in lists.items():
        if name in WHOLE_FILE:
            continue
        seen, pairs = {}, collections.defaultdict(lambda: [0, 0])
        for f in files:
            for it in entries(root, prefix, SUBDIR.get(name, "") + f) or []:
                if not isinstance(it, dict) or "id" not in it:
                    continue
                key = str(it["id"])
                if key in seen and seen[key][0] != f:
                    p = pairs[(seen[key][0], f)]
                    p[0] += 1
                    p[1] += json.dumps(seen[key][1], sort_keys=True) != json.dumps(it, sort_keys=True)
                seen[key] = (f, it)
        for (a, b), (n, differ) in pairs.items():
            found = True
            print(f"  {name}: {a} → {b}，{n} 個 id，其中 {differ} 個內容不同（語言包以 {b} 為準）")
    if not found:
        print("  無")


if __name__ == "__main__":
    main()
