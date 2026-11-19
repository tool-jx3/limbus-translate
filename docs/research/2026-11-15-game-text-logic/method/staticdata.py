#!/usr/bin/env python3
r"""讀遊戲的靜態資料包（Addressables 的 static_s1 bundle），並從裡面產生劇情檔的順序表。

  python3 -I staticdata.py unpack <__data> <輸出資料夾>
      把資料包裡的 JSON（TextAsset）寫成檔案，一個資源一個檔。
  python3 -I staticdata.py storyorder <JSON 資料夾> <語言包的 StoryData 資料夾> [前綴]
      印出 rules/story-order.yaml。語言包資料夾是 raw 分支的 kr/StoryData（前綴 KR_，預設）。

資料包在遊戲的快取裡，不在版控裡：
  %USERPROFILE%\AppData\LocalLow\Unity\ProjectMoon_LimbusCompany\<bundle 的內部名稱>\<hash>\__data
內部名稱與 hash 寫在現行的 Addressables 目錄（com.unity.addressables\catalog_S1.bin）裡 static_s1_0_assets_all_<hash>.bundle
那一筆的後面；2026-11-12 版是 64bd0105e9544bef32ddae650b8bcb26\9fd67c23bc1902c686b4231cc46f6d3a。
資料包沒有加密（UnityFS，LZ4），內容是 JSON。

順序的依據（storyorder）：
  1. 劇情劇場清單 storytheater-main-*、storytheater-other-*、storytheater-personality-*：每一章、每個活動、每個角色
     底下逐筆列出劇情代號（檔名）。主線照 subChapterId，活動照 sortingOrder，人格故事照角色與人格編號。
  2. 載入設定 story-loading-cg*.json 的 storyIds：同樣逐章列出代號，順序和 1 一致，另外多列了戰鬥中播放的檔
     （字尾 I）與第 10 章（RPG 式流程，劇場清單沒有逐筆列）。1 沒有的代號照它在 2 裡的位置插入。
  3. 兩者都沒列的檔照檔名插入同一章、同一類（S、E、ES、<n>D）裡：編號小的在前，同一編號 B → I → A → C。
     這些在輸出裡標「推定」。PC（巡禮）整組推定，放在最接近的活動（subChapterId 較小者）之後；
     E000X～E003X 沒有位置資料，照編號放在活動的最前面。
  其餘沒有任何依據的檔不列，網頁會把它們排在同類別已列的檔之後。
"""
import collections
import glob
import json
import lzma
import os
import re
import struct
import sys


# ---- 資料包 ----

def lz4(src, size):
    out = bytearray()
    i, n = 0, len(src)
    while i < n:
        token = src[i]
        i += 1
        length = token >> 4
        if length == 15:
            while True:
                x = src[i]
                i += 1
                length += x
                if x != 255:
                    break
        out += src[i : i + length]
        i += length
        if i >= n:
            break
        offset = src[i] | src[i + 1] << 8
        i += 2
        length = token & 15
        if length == 15:
            while True:
                x = src[i]
                i += 1
                length += x
                if x != 255:
                    break
        length += 4
        start = len(out) - offset
        if offset >= length:
            out += out[start : start + length]
        else:
            for k in range(length):
                out.append(out[start + k])
    if len(out) != size:
        sys.exit(f"LZ4：解出 {len(out)} bytes，應為 {size}")
    return bytes(out)


def decompress(kind, data, size):
    if kind == 0:
        return data
    if kind in (2, 3):
        return lz4(data, size)
    if kind == 1:
        prop = data[0]
        filters = [{"id": lzma.FILTER_LZMA1, "dict_size": struct.unpack_from("<I", data, 1)[0],
                    "lc": prop % 9, "lp": prop // 9 % 5, "pb": prop // 45}]
        return lzma.LZMADecompressor(lzma.FORMAT_RAW, filters=filters).decompress(data[5:], size)
    sys.exit(f"沒有處理的壓縮方式 {kind}")


def cstring(data, pos):
    end = data.index(b"\0", pos)
    return data[pos:end].decode(), end + 1


def bundle_files(path):
    """UnityFS 資料包裡的檔案：{名稱: 內容}。"""
    data = open(path, "rb").read()
    signature, p = cstring(data, 0)
    if signature != "UnityFS":
        sys.exit(f"{path}: 不是 UnityFS 資料包")
    version = struct.unpack_from(">I", data, p)[0]
    p += 4
    _, p = cstring(data, p)
    _, p = cstring(data, p)
    _, csize, usize, flags = struct.unpack_from(">qIII", data, p)
    p += 20
    if version >= 7:
        p = (p + 15) & ~15
    if flags & 0x80:
        info = data[len(data) - csize :]
    else:
        info = data[p : p + csize]
        p += csize
    info = decompress(flags & 0x3F, info, usize)
    if flags & 0x200:
        p = (p + 15) & ~15
    q = 16
    count = struct.unpack_from(">I", info, q)[0]
    q += 4
    blocks = []
    for _ in range(count):
        blocks.append(struct.unpack_from(">IIH", info, q))
        q += 10
    count = struct.unpack_from(">I", info, q)[0]
    q += 4
    nodes = []
    for _ in range(count):
        offset, size, _ = struct.unpack_from(">qqI", info, q)
        name, q = cstring(info, q + 20)
        nodes.append((name, offset, size))
    body = bytearray()
    for usize, csize, flag in blocks:
        body += decompress(flag & 0x3F, data[p : p + csize], usize)
        p += csize
    return {name: bytes(body[offset : offset + size]) for name, offset, size in nodes}


def text_assets(data):
    """SerializedFile（格式 22 以上）裡的 TextAsset：(名稱, 內容)。"""
    version = struct.unpack_from(">I", data, 8)[0]
    if version < 22:
        sys.exit(f"只支援格式 22 以上（這個檔是 {version}）")
    e = ">" if data[16] else "<"
    data_offset = struct.unpack_from(">q", data, 32)[0]
    p = data.index(b"\0", 48) + 1 + 4
    has_tree = data[p]
    p += 1
    count = struct.unpack_from(e + "i", data, p)[0]
    p += 4
    classes = []
    for _ in range(count):
        class_id = struct.unpack_from(e + "i", data, p)[0]
        p += 4 + 1 + 2
        if class_id == 114:
            p += 16
        p += 16
        if has_tree:
            nodes, strings = struct.unpack_from(e + "ii", data, p)
            p += 8 + nodes * 32 + strings
            deps = struct.unpack_from(e + "i", data, p)[0]
            p += 4 + 4 * deps
        classes.append(class_id)
    count = struct.unpack_from(e + "i", data, p)[0]
    p += 4
    for _ in range(count):
        p = (p + 3) & ~3
        _, start, size, type_index = struct.unpack_from(e + "qqIi", data, p)
        p += 24
        if classes[type_index] != 49:
            continue
        o = data_offset + start
        n = struct.unpack_from(e + "i", data, o)[0]
        name = data[o + 4 : o + 4 + n].decode()
        o = (o + 4 + n + 3) & ~3
        n = struct.unpack_from(e + "i", data, o)[0]
        yield name, data[o + 4 : o + 4 + n]


def cmd_unpack(bundle, out):
    os.makedirs(out, exist_ok=True)
    count = 0
    for content in bundle_files(bundle).values():
        for name, body in text_assets(content):
            if not re.fullmatch(r"[\w.\-]+", name):
                sys.exit(f"資源名稱不能當檔名：{name!r}")
            with open(os.path.join(out, name + ".json"), "wb") as f:
                f.write(body)
            count += 1
    print(f"{count} 個 JSON → {out}")


# ---- 劇情檔的順序 ----

NAME = re.compile(r"(\d*)([A-Z]+?)(\d+)([A-Z])(\d*)")
SUFFIX = {"B": 0, "X": 0, "I": 1, "A": 2, "C": 3}


def load(path):
    with open(path, encoding="utf-8-sig") as f:
        return json.load(f)


def natural(s):
    return [(0, int(t), "") if t.isdigit() else (1, 0, t) for t in re.findall(r"\d+|\D+", s)]


def name_key(story):
    """(類別, 章, 排序鍵)；不合命名規則的回 None。"""
    m = NAME.fullmatch(story)
    if not m or m[4] not in SUFFIX:
        return None
    lead, letters, number, suffix, tail = m.groups()
    chapter = None if lead else int(number) // 100
    return (lead, letters), chapter, (int(number), SUFFIX[suffix], int(tail or 0))


class Order:
    """有序的分組；每個代號只在一組裡。"""

    def __init__(self):
        self.groups = collections.OrderedDict()  # id → [代號]
        self.where = {}  # 代號 → 組 id
        self.inferred = set()

    def add(self, group, story, index=None, inferred=False):
        if story in self.where:
            sys.exit(f"{story} 同時在 {self.where[story]} 與 {group}")
        files = self.groups.setdefault(group, [])
        files.insert(len(files) if index is None else index, story)
        self.where[story] = group
        if inferred:
            self.inferred.add(story)

    def after(self, anchor, story, inferred=False):
        group = self.where[anchor]
        self.add(group, story, self.groups[group].index(anchor) + 1, inferred)

    def before(self, anchor, story, inferred=False):
        group = self.where[anchor]
        self.add(group, story, self.groups[group].index(anchor), inferred)


def story_order(static, pack_names):
    order = Order()

    # 1. 劇場清單。
    main, other = [], []
    for kind, rows in (("main", main), ("other", other)):
        for path in sorted(glob.glob(os.path.join(static, f"storytheater-{kind}-*.json")), key=natural):
            for entry in load(path)["list"]:
                rows.append(entry)
    main.sort(key=lambda e: e["subChapterId"])  # 穩定：同一章的幾個檔維持檔名順序
    other.sort(key=lambda e: e["sortingOrder"])
    for kind, rows in (("main", main), ("other", other)):
        if kind == "other":
            order.groups["other-0"] = []  # 沒有位置資料的 E0xx，見步驟 3
        for entry in rows:
            group = f"{kind}-{entry['subChapterId']}"
            order.groups.setdefault(group, [])
            for story in entry["storyList"]:
                if story.get("storyId"):
                    order.add(group, story["storyId"])
    characters = []
    for path in glob.glob(os.path.join(static, "storytheater-personality-*.json")):
        characters += load(path)["list"]
    for character in sorted(characters, key=lambda c: c["characterId"]):
        for personality in character["storyList"]:
            for story in personality.get("personalityStoryList", []):
                order.add(f"personality-{character['characterId']:02d}", story["storyId"])

    # 2. 載入設定：劇場清單沒有的代號照它在這裡的位置插入。
    lists = []

    def walk(node):
        if isinstance(node, dict):
            if node.get("storyIds"):
                lists.append((node.get("id"), node["storyIds"]))
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    for path in sorted(glob.glob(os.path.join(static, "story-loading-cg*.json")), key=natural):
        walk(load(path))
    for list_id, stories in lists:
        known = [s for s in stories if s in order.where]
        if not known:
            sys.exit(f"載入設定 {list_id} 的代號都不在劇場清單裡，不知道屬於哪一組")
        position = [(list(order.groups).index(order.where[s]), order.groups[order.where[s]].index(s)) for s in known]
        if position != sorted(position):
            sys.exit(f"載入設定 {list_id} 的順序和劇場清單不一致")
        previous = None
        pending = []
        for story in stories:
            if story in order.where:
                for early in pending:  # 排在第一個已知代號之前的
                    order.before(story, early)
                pending = []
            elif previous is None:
                pending.append(story)
                continue
            else:
                order.after(previous, story)
            previous = story

    # 3. 其餘的照檔名插入。
    rest = sorted((n for n in pack_names if n not in order.where), key=lambda n: (name_key(n) is None, name_key(n) or (), n))
    unlisted = []
    for story in rest:
        key = name_key(story)
        if key is None:
            unlisted.append(story)
            continue
        kind, chapter, sort_key = key
        same = []
        for placed in order.where:
            k = name_key(placed)
            if k and k[0] == kind and k[1] == chapter:
                same.append((k[2], placed))
        if kind == ("", "E") and sort_key[0] < 10:
            order.add("other-0", story, inferred=True)
        elif same:
            lower = [x for x in same if x[0] <= sort_key]
            if lower:
                order.after(max(lower)[1], story, inferred=True)
            else:
                order.before(min(same)[1], story, inferred=True)
        elif kind == ("", "PC"):
            order.add("pilgrimage", story, inferred=True)
        else:
            unlisted.append(story)

    # 巡禮放在同一系列（91xx）裡 subChapterId 比它小的最後一個活動之後。
    if "pilgrimage" in order.groups:
        number = min(e["id"] for e in load(os.path.join(static, "storybattle-pilgrimage-normal.json"))["list"]) // 100
        ids = list(order.groups)
        ids.remove("pilgrimage")
        at = max((i for i, g in enumerate(ids) if g.startswith("other-") and number // 100 * 100 <= int(g[6:]) < number), default=len(ids) - 1)
        ids.insert(at + 1, "pilgrimage")
        order.groups = collections.OrderedDict((g, order.groups[g]) for g in ids)

    missing = sorted(s for s in order.where if s not in pack_names)
    return order, unlisted, missing


def cmd_storyorder(static, pack, prefix):
    names = set()
    for path in glob.glob(os.path.join(pack, "*.json")):
        name = os.path.basename(path)[: -len(".json")]
        if not name.startswith(prefix):
            sys.exit(f"{path}: 檔名不是 {prefix} 開頭")
        names.add(name[len(prefix) :])
    order, unlisted, missing = story_order(static, names)
    print("# 劇情檔在遊戲裡的順序。網頁同一個類別裡的檔照這個順序排；這裡沒列的檔排在已列的之後，照檔名。")
    print("# 一組是一章、一個活動或一個角色的人格故事；組的先後就是順序（類別仍由 guides/routing.yaml 決定）。")
    print("#")
    print("# 這個檔是產生的，不要手改：遊戲更新後照")
    print("# docs/research/2026-11-15-game-text-logic/method/README.md 的「劇情檔的順序」重新產生。")
    print("# 依據是遊戲靜態資料的劇情劇場清單與載入設定（研究報告 11-story-order.md）；")
    print("# 標「推定」的檔兩者都沒列，是照檔名插進同一章的（同一編號 B → I → A）。")
    print("groups:")
    for group, stories in order.groups.items():
        stories = [s for s in stories if s in names]
        if not stories:
            continue
        print(f"  - id: {group}")
        print("    files:")
        for story in stories:
            print(f"      - StoryData/{story}.json" + ("  # 推定" if story in order.inferred else ""))
    listed = sum(1 for s in order.where if s in names)
    print(f"{listed} 個檔列入，其中推定 {len(order.inferred)} 個", file=sys.stderr)
    print(f"沒有依據、不列的檔（{len(unlisted)}）：{' '.join(unlisted)}", file=sys.stderr)
    print(f"遊戲資料有、語言包沒有的代號（{len(missing)}）：{' '.join(missing)}", file=sys.stderr)


def main():
    args = sys.argv[1:]
    if len(args) == 3 and args[0] == "unpack":
        cmd_unpack(args[1], args[2])
    elif len(args) in (3, 4) and args[0] == "storyorder":
        cmd_storyorder(args[1], args[2], args[3] if len(args) == 4 else "KR_")
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
