"""分析腳本共用的部分：工作目錄、PE 檔讀取、方法位址索引。

工作目錄由環境變數 RE_WORK 指定（預設 ./re-work），內容：

  輸入（自己放）
    GameAssembly.dll            遊戲根目錄的檔案
    global-metadata.dat         LimbusCompany_Data/il2cpp_data/Metadata/ 的檔案（加密的）
    il2csharp/                  git clone https://github.com/HZBHZB1234/il2csharp
  輸出（腳本產生）
    global-metadata.std.dat     md_decrypt.py：還原成標準格式的 metadata
    decls/                      lift.sh decls：全部型別的宣告、script.json、stringliteral.json
    lift/<名稱>/                lift.sh：個別型別的反編譯結果
    calls.pkl                   who.py：call 指令的索引（第一次執行時建立）

需要：pip install iced-x86 numpy
"""
import bisect
import json
import mmap
import os
import pickle
import struct

WORK = os.environ.get("RE_WORK", "re-work")
DLL = os.path.join(WORK, "GameAssembly.dll")
META_RAW = os.path.join(WORK, "global-metadata.dat")
META_STD = os.path.join(WORK, "global-metadata.std.dat")
DECLS = os.path.join(WORK, "decls")
CALLS = os.path.join(WORK, "calls.pkl")


class PE:
    """只做需要的事：區段表、RVA 與檔案位移互轉、.pdata 的函式範圍。"""

    def __init__(self, path=DLL):
        f = open(path, "rb")
        self.d = d = mmap.mmap(f.fileno(), 0, access=mmap.ACCESS_READ)
        pe = struct.unpack_from("<I", d, 0x3C)[0]
        nsec = struct.unpack_from("<H", d, pe + 6)[0]
        opt = struct.unpack_from("<H", d, pe + 20)[0]
        self.base = struct.unpack_from("<Q", d, pe + 24 + 24)[0]
        so = pe + 24 + opt
        self.secs = []  # (名稱, RVA, 虛擬大小, 檔案位移, 檔案大小)
        for i in range(nsec):
            name = d[so + i * 40:so + i * 40 + 8].rstrip(b"\0").decode()
            vs, va, rs, ro = struct.unpack_from("<IIII", d, so + i * 40 + 8)
            self.secs.append((name, va, vs, ro, rs))
        self.pdata_rva, self.pdata_size = struct.unpack_from("<II", d, pe + 24 + 112 + 3 * 8)

    def code_sections(self):
        return [s for s in self.secs if s[0] in (".text", "il2cpp")]

    def rva2off(self, rva):
        for _, va, vs, ro, rs in self.secs:
            if va <= rva < va + max(vs, rs):
                return ro + rva - va if rva - va < rs else None
        return None

    def read(self, rva, n):
        o = self.rva2off(rva)
        return self.d[o:o + n]

    def func_of(self, rva):
        """.pdata 裡包含這個 RVA 的 (起點, 終點, unwind)。大函式會被切成好幾段。"""
        if not hasattr(self, "_f"):
            o = self.rva2off(self.pdata_rva)
            self._f = [struct.unpack_from("<III", self.d, o + i * 12) for i in range(self.pdata_size // 12)]
            self._fs = [x[0] for x in self._f]
        i = bisect.bisect_right(self._fs, rva) - 1
        if i >= 0 and self._f[i][0] <= rva < self._f[i][1]:
            return self._f[i]
        return None


def rel32_targets(pe, sec, positions_of_disp):
    """給一批 disp32 的位置（區段內位移），回傳它們指向的 RVA（以 disp 結尾為基準）。"""
    import numpy as np
    _, va, _, ro, rs = sec
    a = np.frombuffer(pe.d[ro:ro + rs], dtype=np.uint8)
    p = positions_of_disp
    v = a[p].astype(np.int64) | (a[p + 1].astype(np.int64) << 8) | (a[p + 2].astype(np.int64) << 16) | (a[p + 3].astype(np.int64) << 24)
    v = np.where(v >= 2 ** 31, v - 2 ** 32, v)
    return p + va + 4 + v


def load_calls(pe=None):
    """回傳 (names, src, dst)：names 是 {方法起點 RVA: [名稱…]}（多個方法共用同一段機器碼時有多個名稱），
    src／dst 是所有 call／jmp rel32 的來源與目標 RVA，依 dst 排序。"""
    import numpy as np
    if os.path.exists(CALLS):
        return pickle.load(open(CALLS, "rb"))
    pe = pe or PE()
    names = {}
    for m in json.load(open(os.path.join(DECLS, "script.json")))["ScriptMethod"]:
        names.setdefault(m["Address"] - pe.base, []).append(m["Name"])
    srcs, dsts = [], []
    for sec in pe.code_sections():
        _, va, _, ro, rs = sec
        a = np.frombuffer(pe.d[ro:ro + rs], dtype=np.uint8)
        pos = np.nonzero((a[:-5] == 0xE8) | (a[:-5] == 0xE9))[0]
        srcs.append(pos + va)
        dsts.append(rel32_targets(pe, sec, pos + 1))
    src, dst = np.concatenate(srcs), np.concatenate(dsts)
    order = np.argsort(dst, kind="stable")
    out = (names, src[order], dst[order])
    pickle.dump(out, open(CALLS, "wb"))
    return out


class Owners:
    """RVA → 所在方法的起點（用方法起點排序後二分搜尋；沒有名稱的原生函式會被算進前一個方法）。"""

    def __init__(self, names):
        self.names = names
        self.starts = sorted(names)

    def start(self, rva):
        i = bisect.bisect_right(self.starts, rva) - 1
        return self.starts[i] if i >= 0 else None

    def end(self, start):
        i = bisect.bisect_right(self.starts, start)
        return self.starts[i] if i < len(self.starts) else start + 0x1000

    def find(self, query):
        """'0x…' 當 RVA；否則找名稱含這段字的方法。"""
        if query.lower().startswith("0x"):
            return [int(query, 16)]
        return sorted(a for a, ns in self.names.items() if any(query in n for n in ns))
