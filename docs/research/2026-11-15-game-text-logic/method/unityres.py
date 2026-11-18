#!/usr/bin/env python3
r"""讀遊戲本體的 resources.assets（Unity 6000.3，沒有型別資訊）裡和文字有關的設定。

  python3 -I unityres.py <LimbusCompany_Data 資料夾> texts <輸出資料夾>   把所有 TextAsset 寫成檔案（啟動用清單、啟動用文字、換行規則…）
  python3 -I unityres.py <LimbusCompany_Data 資料夾> styles               TextMeshPro 樣式表：名稱、開頭、結尾
  python3 -I unityres.py <LimbusCompany_Data 資料夾> config               下載設定（TextAssetPatchConfig）；加密金鑰不印
  python3 -I unityres.py <LimbusCompany_Data 資料夾> labels [檔案…]      靜態標籤（UITextDataLoader）：文字 id、物件路徑、字級、縮字、換行、溢出、寬度（TSV）
  python3 -I unityres.py <LimbusCompany_Data 資料夾> tmp <檔案> [正規表示式]  某個檔裡的文字元件：物件路徑、字級、縮字範圍、換行、溢出、框的寬高（TSV）

檔案是資料夾裡的檔名：resources.assets（預設）、level0～level8（場景）、sharedassets0～8.assets。戰鬥場景是 level3、
劇情場景是 level4；例如戰鬥氣泡是 `tmp level3 'Dialog/\[Text\]Dialog$'`，劇情台詞是 `tmp level4 'rect_texts/\[Text\]Serif$'`。
需要資料夾裡的 globalgamemanagers.assets（元件的類別名稱在裡面）。物件的欄位順序是照這一版的類別寫死的，
遊戲或 Unity 改版後要重新核對（每個指令解析完都會檢查是否剛好讀到物件結尾或數值是否合理）。
"""
import mmap
import os
import struct
import sys


class SerializedFile:
    """Unity SerializedFile（格式 22 以上）的物件表。"""

    def __init__(self, path):
        self.f = open(path, "rb")
        self.m = m = mmap.mmap(self.f.fileno(), 0, access=mmap.ACCESS_READ)
        version = struct.unpack_from(">I", m, 8)[0]
        if version < 22 or m[16]:
            sys.exit(f"{path}: 只支援格式 22 以上、little-endian（這個檔是 {version}）")
        data_offset = struct.unpack_from(">q", m, 32)[0]
        p = m.find(b"\0", 48) + 1 + 4
        has_tree = m[p]
        p += 1
        if has_tree:
            sys.exit(f"{path}: 帶型別樹的檔案這支腳本沒有處理")
        (n,) = struct.unpack_from("<i", m, p)
        p += 4
        self.types = []
        for _ in range(n):
            (class_id,) = struct.unpack_from("<i", m, p)
            p += 4 + 1 + 2 + (16 if class_id == 114 else 0) + 16
            self.types.append(class_id)
        (n,) = struct.unpack_from("<i", m, p)
        p += 4
        self.objects = {}
        for _ in range(n):
            p = (p + 3) & ~3
            path_id, start, size, type_index = struct.unpack_from("<qqIi", m, p)
            p += 24
            self.objects[path_id] = (data_offset + start, size, self.types[type_index])
        (n,) = struct.unpack_from("<i", m, p)
        p += 4
        for _ in range(n):
            p = ((p + 4 + 3) & ~3) + 8
        (n,) = struct.unpack_from("<i", m, p)
        p += 4
        self.externals = []
        for _ in range(n):
            p = m.find(b"\0", p) + 1 + 20
            q = m.find(b"\0", p)
            self.externals.append(m[p:q].decode())
            p = q + 1

    def data(self, path_id):
        offset, size, _ = self.objects[path_id]
        return self.m[offset : offset + size]


class Reader:
    def __init__(self, data, pos=0):
        self.b, self.p = data, pos

    def i32(self):
        (v,) = struct.unpack_from("<i", self.b, self.p)
        self.p += 4
        return v

    def f32(self):
        (v,) = struct.unpack_from("<f", self.b, self.p)
        self.p += 4
        return v

    def string(self):
        n = self.i32()
        s = self.b[self.p : self.p + n]
        self.p = (self.p + n + 3) & ~3
        return s.decode("utf-8", "replace")

    def pptr(self):
        file_id, path_id = struct.unpack_from("<iq", self.b, self.p)
        self.p += 12
        return file_id, path_id

    def skip(self, n):
        self.p += n

    def array(self, item_size):
        self.p += 4 + item_size * struct.unpack_from("<i", self.b, self.p)[0]


class Resources:
    def __init__(self, folder, name="resources.assets"):
        self.name = name
        self.res = SerializedFile(os.path.join(folder, name))
        gm = SerializedFile(os.path.join(folder, "globalgamemanagers.assets"))
        self.scripts = {}
        for path_id, (_, _, class_id) in gm.objects.items():
            if class_id == 115:  # MonoScript：名稱、執行順序、雜湊、類別、命名空間
                r = Reader(gm.data(path_id))
                r.string()
                r.skip(20)
                name = r.string()
                namespace = r.string()
                self.scripts[path_id] = f"{namespace}.{name}" if namespace else name
        self.gm_index = self.res.externals.index("globalgamemanagers.assets") + 1 if "globalgamemanagers.assets" in self.res.externals else -1

    def behaviours(self, *classes):
        """(path id, 類別, 所在的 GameObject, 讀到自訂欄位開頭的 Reader)"""
        for path_id, (offset, size, class_id) in self.res.objects.items():
            if class_id != 114:
                continue
            r = Reader(self.res.m[offset : offset + size])
            game_object = r.pptr()[1]
            r.skip(4)
            file_id, script = r.pptr()
            cls = self.scripts.get(script, "?") if file_id == self.gm_index else "?"
            if cls in classes:
                r.string()
                yield path_id, cls, game_object, r

    def text_assets(self):
        for path_id, (_, _, class_id) in self.res.objects.items():
            if class_id == 49:
                r = Reader(self.res.data(path_id))
                name = r.string()
                n = r.i32()
                yield path_id, name, r.b[r.p : r.p + n]


def cmd_texts(res, out):
    os.makedirs(out, exist_ok=True)
    for path_id, name, body in res.text_assets():
        with open(os.path.join(out, f"{name.replace('/', '_')}.{path_id}.txt"), "wb") as f:
            f.write(body)
        print(f"{name}\t{len(body)}")


def cmd_styles(res):
    for _, _, _, r in res.behaviours("TMPro.TMP_StyleSheet"):
        for _ in range(r.i32()):
            name = r.string()
            r.skip(4)
            opening, closing = r.string(), r.string()
            r.array(4)
            r.array(4)
            print(f"{name}\t{opening}\t{closing}")
        if r.p != len(r.b):
            sys.exit("樣式表沒有剛好讀完：欄位順序可能變了")


def cmd_config(res):
    for _, _, _, r in res.behaviours("TextAssetPatch.TextAssetPatchConfig"):
        print("patchMode", r.i32())
        print("localizeLoadMode", r.i32(), "（0 是 Addressables，其他是檔案）")
        for name in ("staticDataPath", "localizeDataPath", "fmodDataPath", "fmodLocalPath", "remoteUrlLocalize", "remoteUrlFmod"):
            print(name, r.string())
        r.skip(8)
        r.string()  # 加密金鑰與 IV：不印
        r.string()
        for name in ("encryptedStaticDataPath", "fmodPatchInfoFilePath", "localizePatchInfoFilePath"):
            print(name, r.string())


def text_components(res):
    """物件路徑的函式、每個 GameObject 的文字設定與框。"""
    names, transform_of, parent, owner, rects = {}, {}, {}, {}, {}
    for path_id, (offset, size, class_id) in res.res.objects.items():
        if class_id == 1:  # GameObject：元件、layer、名稱
            r = Reader(res.res.m[offset : offset + size])
            r.array(12)
            r.skip(4)
            names[path_id] = r.string()
        elif class_id in (4, 224):  # Transform／RectTransform
            r = Reader(res.res.m[offset : offset + size])
            game_object = r.pptr()[1]
            r.skip(40)
            r.array(12)
            transform_of[game_object], owner[path_id], parent[path_id] = path_id, game_object, r.pptr()[1]
            if class_id == 224:  # anchorMin、anchorMax、anchoredPosition、sizeDelta、pivot
                rects[game_object] = struct.unpack_from("<10f", r.b, r.p)

    def path(game_object):
        parts, t = [], transform_of.get(game_object)
        while t and len(parts) < 60:
            parts.append(names.get(owner.get(t), "?"))
            t = parent.get(t)
        return "/".join(reversed(parts))

    texts, odd = {}, 0
    for _, _, game_object, r in res.behaviours("TMPro.TextMeshProUGUI", "TMPro.CustomRubyTextMeshProUGUI"):
        r.skip(12 + 16 + 4 + 16 + 4)  # Graphic：材質、顏色、raycast、padding、maskable
        if r.i32():  # 有事件接收者的元件，版面不同，略過
            continue
        r.string()  # 編輯時留下的文字
        r.skip(4 + 12 + 12)
        r.array(12)
        r.skip(12)
        r.array(12)
        r.skip(4 + 16 + 4 + 4 + 64 + 12 + 12 + 4 + 12 + 4 + 4 + 4)
        size = r.f32()
        r.skip(8)
        auto, low, high = r.i32(), r.f32(), r.f32()
        r.skip(16 + 28)  # 樣式、對齊；字距、橫向縮放、詞距、行距…
        wrap = r.i32()
        r.skip(4)
        overflow = r.i32()
        # 世界座標的標籤（RPG 地圖上的地名）字級可以小於 1
        if not (0.01 <= size <= 2000 and auto in (0, 1) and 0 <= wrap <= 4 and 0 <= overflow <= 8):
            odd += 1
            continue
        texts[game_object] = (size, auto, low, high, wrap, overflow)
    if odd > max(3, len(texts) // 100):
        sys.exit(f"{res.name}: {odd} 個文字元件的數值不合理：欄位順序可能變了")
    if odd:
        print(f"{res.name}: 略過 {odd} 個數值不合理的文字元件", file=sys.stderr)
    return path, texts, rects


def cmd_tmp(res, pattern):
    import re

    path, texts, rects = text_components(res)
    wanted = re.compile(pattern)
    print("path\tsize\tautosize\tmin\tmax\twrap\toverflow\twidth\theight")
    for game_object, (size, auto, low, high, wrap, overflow) in texts.items():
        p = path(game_object)
        if not wanted.search(p):
            continue
        rect = rects.get(game_object)
        # 錨點左右（上下）相同時寬（高）是固定的；否則跟著上層物件伸縮
        width = f"{rect[6]:g}" if rect and abs(rect[0] - rect[2]) < 1e-6 else "隨上層"
        height = f"{rect[7]:g}" if rect and abs(rect[1] - rect[3]) < 1e-6 else "隨上層"
        print(f"{p}\t{size:g}\t{auto}\t{low:g}\t{high:g}\t{wrap}\t{overflow}\t{width}\t{height}")


def cmd_labels(res, header=True):
    path, texts, rects = text_components(res)
    if header:
        print("file\tkey\tlocal\tpath\tsize\tautosize\tmin\tmax\twrap\toverflow\tfixed_width")
    for _, _, game_object, r in res.behaviours("UITextDataLoader"):
        key = r.string()
        local = r.b[r.p] if r.p < len(r.b) else 0
        size, auto, low, high, wrap, overflow = texts.get(game_object, ("", "", "", "", "", ""))
        rect = rects.get(game_object)
        width = round(rect[6]) if rect and abs(rect[0] - rect[2]) < 1e-6 else ""
        print(f"{res.name}\t{key}\t{local}\t{path(game_object)}\t{size}\t{auto}\t{low}\t{high}\t{wrap}\t{overflow}\t{width}")


def main():
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    folder, command, rest = sys.argv[1], sys.argv[2], sys.argv[3:]
    if command == "texts" and rest:
        cmd_texts(Resources(folder), rest[0])
    elif command == "styles":
        cmd_styles(Resources(folder))
    elif command == "config":
        cmd_config(Resources(folder))
    elif command == "labels":
        for i, name in enumerate(rest or ["resources.assets"]):
            cmd_labels(Resources(folder, name), header=i == 0)
    elif command == "tmp" and rest:
        cmd_tmp(Resources(folder, rest[0]), rest[1] if len(rest) > 1 else "")
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
