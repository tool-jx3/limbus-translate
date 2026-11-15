#!/usr/bin/env python3
"""把遊戲的 global-metadata.dat 還原成標準 IL2CPP 格式（v39），讓 il2csharp 讀得懂。

  RE_WORK=… python3 -I md_decrypt.py            解密並寫出 $RE_WORK/global-metadata.std.dat
  RE_WORK=… python3 -I md_decrypt.py probe      只印出載入函式的常數與解密後的檔頭，不寫檔

遊戲做了什麼（2026-11-12 版）：

  檔頭    不是標準的 0xFAB11BAF＋版本＋31 組 (位移, 大小, 筆數)，而是 0x324 bytes、67 組 (大小, 筆數, 位移)，
          其中 31 組是真的表，其餘是假的；順序打亂；真正的位移 ＝ 檔頭裡的位移 ＋ 一個寫死在程式裡的常數。
  加密    檔頭與 7 張表各有一個 64 位元種子。金鑰流：
              x ^= x << 13;  x ^= x >> 7;  x ^= x << 17          （xorshift64）
              byte = SBOX[(x ^ x>>8 ^ x>>16 ^ x>>24) & 0xFF]      （SBOX 是 .rdata 裡 256 bytes 的替換表）
          資料與金鑰流 XOR。
  載入    GameAssembly.dll 裡「開啟 Metadata/global-metadata.dat」之後的那個函式：把檔頭複製出來解密，
          再逐張表配置記憶體、複製、解密。種子、替換表位址、表在檔頭裡的欄位位置與位移常數都在這個函式裡。

這個腳本怎麼做：

  1. 找載入函式：檔名是在堆疊上用立即值組出來的，搜尋 `mov dword [rsp+20h], 'glob'` 的機器碼。
  2. 反組譯它，讀出：檔頭大小、檔頭種子、替換表位址、每張加密表的 (大小欄位的位置, 位移常數, 種子)。
  3. 解密檔頭。各表在檔案裡仍照標準順序緊接排列（4 bytes 對齊），所以從第一張表開始依大小往後推，
     遇到加密表就用它確定的位移校正。沒加密的表位移常數散在各個讀取函式裡，不必去找。
  4. 用內容檢查位置對不對（名稱索引是否都落在字串開頭）。
  5. 依標準順序重寫檔頭與各表。

遊戲改版後可能要改的地方（都在下面的常數）：

  TABLES   標準表名 → 檔頭欄位位置。對不上時先跑 probe：看每組的「大小÷筆數」（stride）與位移順序重新對應。
           已知的 stride：stringLiteral 4、events 24、properties 20、methods 32、parameters／fields／兩種
           DefaultValues／fieldMarshaledSizes 12、genericParameters 14、genericContainers 16、
           interfaceOffsets／fieldRefs／attributeDataRange／unresolved…Ranges 8、typeDefinitions 82、images 36、
           assemblies 68、其餘索引表 4；三張位元組資料表的「筆數」沒有意義或等於大小。
  GAPS     表與表之間多出來的 bytes。推算的位移與加密表的位移對不上時，代表前面某張表前有空隙，
           用步驟 4 的檢查找出是哪一張。
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式不會自動加入腳本所在目錄

import struct

import numpy as np
from iced_x86 import Code, Decoder, Mnemonic, OpKind, Register

from re_common import META_RAW, META_STD, PE

M64 = (1 << 64) - 1

# 標準 v39 的表順序，以及（這一版）各表在自訂檔頭裡的欄位位置：該位置是大小，+4 筆數，+8 位移
TABLES = [
    ("stringLiteral", 0x288), ("stringLiteralData", 0x2E8), ("string", 0x0CC), ("events", 0x24C),
    ("properties", 0x1EC), ("methods", 0x0E4), ("parameterDefaultValues", 0x258), ("fieldDefaultValues", 0x1D4),
    ("fieldAndParameterDefaultValueData", 0x1E0), ("fieldMarshaledSizes", 0x084), ("parameters", 0x2A0),
    ("fields", 0x00C), ("genericParameters", 0x168), ("genericParameterConstraints", 0x09C),
    ("genericContainers", 0x120), ("nestedTypes", 0x0C0), ("interfaces", 0x240), ("vtableMethods", 0x1B0),
    ("interfaceOffsets", 0x060), ("typeDefinitions", 0x2D0), ("images", 0x078), ("assemblies", 0x174),
    ("fieldRefs", 0x228), ("referencedAssemblies", 0x1A4), ("attributeData", 0x21C), ("attributeDataRange", 0x090),
    ("unresolvedVirtualCallParameterTypes", 0x018), ("unresolvedVirtualCallParameterRanges", 0x210),
    ("windowsRuntimeTypeNames", 0x2C4), ("windowsRuntimeStrings", 0x03C), ("exportedTypeDefinitions", 0x114),
]
GAPS = {"fieldMarshaledSizes": 4}
# 第一欄是名稱索引（u32）的表：(表名, stride)，用來檢查位置
NAME_INDEXED = [("events", 24), ("properties", 20), ("methods", 32), ("parameters", 12), ("fields", 12)]


def keystream(seed, n, sbox):
    out = bytearray(n)
    x = seed
    for i in range(n):
        x ^= (x << 13) & M64
        x ^= x >> 7
        x ^= (x << 17) & M64
        out[i] = sbox[(x ^ (x >> 8) ^ (x >> 16) ^ (x >> 24)) & 0xFF]
    return bytes(out)


def xor(a, b):
    return (np.frombuffer(a, dtype=np.uint8) ^ np.frombuffer(b, dtype=np.uint8)).tobytes()


def find_loader(pe):
    """回傳 (檔頭大小, 檔頭種子, 替換表 RVA, [(大小欄位位置, 位移常數, 種子)…])。"""
    pat = bytes.fromhex("c744242067 6c6f62".replace(" ", ""))  # mov dword ptr [rsp+20h], 'glob'
    hits = []
    for _, va, _, ro, rs in pe.code_sections():
        i = pe.d.find(pat, ro, ro + rs)
        while i >= 0:
            hits.append(va + i - ro)
            i = pe.d.find(pat, i + 1, ro + rs)
    if len(hits) != 1:
        sys.exit("找不到唯一的 metadata 載入函式（找到 %d 個）；載入方式可能改了" % len(hits))
    start = hits[0]
    header_size = sbox = None
    seeds, size_field, delta, sections = [], None, 0, []
    for ins in Decoder(64, bytes(pe.read(start, 0x2000)), ip=pe.base + start):
        m = ins.mnemonic
        if m == Mnemonic.RET and seeds:
            break
        if m == Mnemonic.MOV and ins.op0_kind == OpKind.REGISTER and ins.op1_kind == OpKind.IMMEDIATE32 \
                and ins.op0_register == Register.ECX and header_size is None and not seeds:
            header_size = ins.immediate32  # 配置檔頭副本的大小
        elif m == Mnemonic.LEA and ins.is_ip_rel_memory_operand and sbox is None and header_size is not None:
            sbox = ins.ip_rel_memory_address - pe.base
        elif ins.code == Code.MOV_R64_IMM64 and ins.op0_register == Register.RDX:
            if seeds:
                sections.append((size_field, delta, ins.immediate64))
            seeds.append(ins.immediate64)
            size_field, delta = None, 0
        elif m == Mnemonic.MOVSXD and ins.op0_register == Register.RCX and ins.op1_kind == OpKind.MEMORY and seeds:
            if size_field is None:
                size_field = ins.memory_displacement  # movsxd rcx,[檔頭+X]：這張表的大小
        elif m in (Mnemonic.ADD, Mnemonic.SUB) and ins.op0_kind == OpKind.REGISTER and ins.op0_register == Register.ECX \
                and ins.op1_kind in (OpKind.IMMEDIATE32, OpKind.IMMEDIATE8TO32) and seeds:
            delta = ins.immediate32 if m == Mnemonic.ADD else -ins.immediate32
            if delta >= 1 << 31:
                delta -= 1 << 32
    table = bytes(pe.read(sbox, 256)) if sbox else b""
    if header_size is None or len(set(table)) != 256 or not sections:
        sys.exit("載入函式的形狀與預期不同（檔頭大小 %r、替換表 %r、加密表 %d 張）" % (header_size, sbox, len(sections)))
    return header_size, seeds[0], sbox, sections


def main(probe=False):
    pe = PE()
    md = open(META_RAW, "rb").read()
    header_size, header_seed, sbox_rva, sections = find_loader(pe)
    sbox = bytes(pe.read(sbox_rva, 256))
    hdr = xor(md[:header_size], keystream(header_seed, header_size, sbox))
    enc = {field: (delta, seed) for field, delta, seed in sections}

    def h(o):
        return struct.unpack_from("<I", hdr, o)[0]

    print("檔頭 0x%x bytes，種子 %016x，替換表 rva 0x%x，加密表 %d 張" % (header_size, header_seed, sbox_rva, len(sections)))
    for field, (delta, seed) in sorted(enc.items()):
        print("  欄位 0x%03x  位移常數 %+#x  種子 %016x" % (field, delta, seed))
    if probe:
        known = {f: n for n, f in TABLES}
        print("解密後的檔頭（依位移排序）：")
        for o in sorted(range(0, header_size - 8, 12), key=lambda o: h(o + 8)):
            size, count, off = h(o), h(o + 4), h(o + 8)
            print("  0x%03x  位移 %8x  大小 %8x  筆數 %7x  stride %-8s %s %s" % (
                o, off, size, count, "%.3f" % (size / count) if count else "-",
                "加密" if o in enc else "    ", known.get(o, "")))
        return
    missing = set(enc) - {f for _, f in TABLES}
    if missing:
        sys.exit("加密表的檔頭欄位 %s 不在 TABLES 裡：檔頭配置改了，先跑 probe" % [hex(x) for x in sorted(missing)])

    # 依標準順序推算每張表的實際位移
    first = TABLES[0][1]
    if first not in enc:
        sys.exit("第一張表沒有加密，無法定位起點")
    pos, layout = h(first + 8) + enc[first][0], []
    for name, field in TABLES:
        size, count, off = h(field), h(field + 4), h(field + 8)
        pos = ((pos + 3) & ~3) + GAPS.get(name, 0)
        if field in enc and off + enc[field][0] != pos:
            sys.exit("%s：推算位移 0x%x，載入函式說是 0x%x。前面有表的大小或空隙不對（見 GAPS）" % (name, pos, off + enc[field][0]))
        layout.append((name, field, pos, size, count))
        pos += size
    print("各表結束於 0x%x，檔案大小 0x%x（後面是填充資料）" % (pos, len(md)))
    if pos > len(md):
        sys.exit("推算超出檔案大小：TABLES 對應錯了")

    # 解密、重寫
    out, body, base, entries, plain = bytearray(struct.pack("<Ii", 0xFAB11BAF, 39)), bytearray(), 8 + 12 * len(TABLES), [], {}
    for name, field, pos, size, count in layout:
        while (base + len(body)) % 4:
            body.append(0)
        data = md[pos:pos + size]
        if field in enc:
            data = xor(data, keystream(enc[field][1], size, sbox))
        plain[name] = data
        entries.append((base + len(body), size, count))
        body += data
    for e in entries:
        out += struct.pack("<iii", *e)
    out += body

    # 檢查：名稱索引都要落在字串表裡某個字串的開頭
    strings = np.frombuffer(plain["string"], dtype=np.uint8)
    starts = np.zeros(len(strings) + 1, dtype=bool)
    starts[0] = True
    starts[1:][strings == 0] = True
    counts = {n: c for n, _, _, _, c in layout}
    for name, stride in NAME_INDEXED:
        n = counts[name]
        idx = np.frombuffer(plain[name][:n * stride], dtype=np.uint8).reshape(n, stride)[:, :4].copy().view("<u4")[:, 0]
        ok = idx < len(strings)
        ratio = float((starts[np.where(ok, idx, 0)] & ok).mean()) if n else 1.0
        print("  %-12s %7d 筆，名稱索引正確 %.4f" % (name, n, ratio))
        if ratio < 0.999:
            sys.exit("%s 的位置或 stride 不對" % name)
    open(META_STD, "wb").write(out)
    print("寫出", META_STD, len(out), "bytes")


if __name__ == "__main__":
    main(probe=len(sys.argv) > 1 and sys.argv[1] == "probe")
