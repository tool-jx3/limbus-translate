#!/usr/bin/env python3
"""呼叫關係與常值引用。

  RE_WORK=… python3 -I who.py callers <方法名稱的一部分|0xRVA>…      誰直接呼叫它（只有 call／jmp rel32；虛擬呼叫、委派找不到）
  RE_WORK=… python3 -I who.py calls   <方法名稱的一部分|0xRVA> [regex]  它依序呼叫了哪些有名稱的方法
  RE_WORK=… python3 -I who.py literal <字串常值編號>…                 哪些方法用到這個字串常值（編號見 stringliteral.json）
  RE_WORK=… python3 -I who.py slot    <0xRVA>…                         哪些方法引用這個 metadata 欄位（反組譯裡 rip 相對位址的 rva）

共用同一段機器碼的方法（空函式、回傳固定值的 getter、泛型共用實作）只能查到位址，分不出是哪一個。
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式不會自動加入腳本所在目錄

import re
import struct

import numpy as np
from iced_x86 import Decoder

from re_common import PE, Owners, load_calls, rel32_targets

NOISE = re.compile(r"^(System\.(Object|Runtime|Threading)|UnityEngine\.Object\.op_)")


def callers(own, src, dst, query):
    for t in own.find(query):
        lo, hi = np.searchsorted(dst, t, "left"), np.searchsorted(dst, t, "right")
        print("==", hex(t), own.names.get(t, ["?"])[:3], "呼叫處:", hi - lo)
        seen = set()
        for s in src[lo:hi]:
            o = own.start(int(s))
            if o not in seen:
                seen.add(o)
                print("   ", hex(o), own.names[o][:2])


def calls(pe, own, query, flt=None):
    for t in own.find(query):
        end = min(own.end(t), t + 0x8000)
        print("==", hex(t), own.names.get(t, ["?"])[0])
        last = None
        for ins in Decoder(64, bytes(pe.read(t, end - t)), ip=pe.base + t):
            try:
                g = ins.near_branch_target - pe.base
            except Exception:
                continue
            ns = own.names.get(g)
            if not ns or len(ns) > 3 or NOISE.match(ns[0]) or (flt and not flt.search(ns[0])):
                continue
            if ns[0] != last:
                print("   %x %s" % (ins.ip - pe.base, ns[0]))
            last = ns[0]


def slot_refs(pe, own, slots):
    """所有 disp32 逐一當成 rip 相對位址，找指向 slots 的。"""
    for sec in pe.code_sections():
        _, va, _, ro, rs = sec
        tg = rel32_targets(pe, sec, np.arange(rs - 3, dtype=np.int64))
        for t in slots:
            seen = set()
            for h in np.nonzero(tg == t)[0]:
                o = own.start(int(h) + va)
                if o not in seen:
                    seen.add(o)
                    print("   ", hex(t), "<-", hex(int(h) + va), own.names[o][0])


def literal_slots(pe, idx):
    """字串常值的 metadata usage 欄位值是 (5 << 29) | (編號 << 1) | 1，以 8 bytes 存在資料區段。"""
    pat = struct.pack("<Q", (5 << 29) | (idx << 1) | 1)
    out = []
    for _, va, _, ro, rs in pe.secs:
        d = pe.d[ro:ro + rs]
        i = d.find(pat)
        while i >= 0:
            if i % 8 == 0:
                out.append(va + i)
            i = d.find(pat, i + 1)
    return out


if __name__ == "__main__":
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    pe = PE()
    names, src, dst = load_calls(pe)
    own = Owners(names)
    cmd, args = sys.argv[1], sys.argv[2:]
    if cmd == "callers":
        for q in args:
            callers(own, src, dst, q)
    elif cmd == "calls":
        calls(pe, own, args[0], re.compile(args[1]) if len(args) > 1 else None)
    elif cmd == "literal":
        for a in args:
            slots = literal_slots(pe, int(a))
            print("字串常值", a, "的欄位:", [hex(s) for s in slots])
            slot_refs(pe, own, slots)
    elif cmd == "slot":
        slot_refs(pe, own, [int(a, 16) for a in args])
    else:
        sys.exit(__doc__)
