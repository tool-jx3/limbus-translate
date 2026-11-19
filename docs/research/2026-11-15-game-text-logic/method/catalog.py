#!/usr/bin/env python3
"""解讀 Addressables 的 JSON 目錄（catalog_S1.json）：每個資源的型別，以及它在哪個 bundle。

  python3 -I catalog.py <catalog_S1.json> [正規表示式]

沒有 bundle 的資源（印成「本體」）在遊戲安裝資料夾的 LimbusCompany_Data 裡（Resources 或場景），不在下載的 bundle 裡。
不給正規表示式時只印各型別的數量。

目錄是遊戲下載的資料，不在版控裡：
  %USERPROFILE%\\AppData\\LocalLow\\ProjectMoon\\LimbusCompany\\com.unity.addressables\\catalog_S1.json

注意（2026-11-19）：這支腳本只讀得懂 JSON 格式。遊戲現在用的是同一個資料夾裡二進位格式的 catalog_S1.bin；
catalog_S1.json 是舊版（2026-05）留下來的，內容不是現行版本，只能拿來看目錄的結構。
"""
import base64
import collections
import json
import re
import struct
import sys


def load(path):
    with open(path, encoding="utf-8-sig") as f:
        d = json.load(f)
    ids = d["m_InternalIds"]
    keys = base64.b64decode(d["m_KeyDataString"])
    buckets_raw = base64.b64decode(d["m_BucketDataString"])
    entries_raw = base64.b64decode(d["m_EntryDataString"])

    # bucket：一個 key 對到哪些 entry
    n, off, buckets = struct.unpack_from("<i", buckets_raw, 0)[0], 4, []
    for _ in range(n):
        _key_offset, count = struct.unpack_from("<ii", buckets_raw, off)
        off += 8
        buckets.append(struct.unpack_from(f"<{count}i", buckets_raw, off))
        off += 4 * count
    del keys  # key 的內容用不到：相依關係以 bucket 的編號表示

    # entry：internal id、provider、相依的 bucket、相依雜湊、額外資料、主要 key、型別
    n = struct.unpack_from("<i", entries_raw, 0)[0]
    entries = [struct.unpack_from("<7i", entries_raw, 4 + 28 * i) for i in range(n)]
    types = [t["m_ClassName"].split(".")[-1] for t in d["m_resourceTypes"]]
    is_bundle = ["AssetBundleProvider" in p for p in d["m_ProviderIds"]]

    assets = {}
    for internal, provider, dep, _hash, _data, _key, rtype in entries:
        if is_bundle[provider]:
            continue
        bundles = sorted({ids[entries[e][0]].rsplit("/", 1)[-1] for e in buckets[dep]}) if dep >= 0 else []
        assets.setdefault(ids[internal], (types[rtype], bundles))
    return assets


def main():
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    assets = load(sys.argv[1])
    if len(sys.argv) < 3:
        print(f"{len(assets)} 個資源")
        for t, n in collections.Counter(t for t, _ in assets.values()).most_common():
            print(f"  {n:6d} {t}")
        print(f"  其中 {sum(1 for _, b in assets.values() if not b)} 個在本體")
        return
    pattern = re.compile(sys.argv[2])
    for name, (rtype, bundles) in sorted(assets.items()):
        if pattern.search(name):
            print(f"{name}\t{rtype}\t{bundles[0] if bundles else '本體'}")


if __name__ == "__main__":
    main()
