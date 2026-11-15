#!/usr/bin/env python3
"""反組譯一段機器碼，標出 call／jmp 的目標方法名稱與 rip 相對位址指到的字串。

  RE_WORK=… python3 -I disasm.py <起點RVA|方法名稱的一部分> [終點RVA] [最多幾行]

給方法名稱時，範圍是到下一個有名稱的方法為止。.pdata 會把大函式切成幾段，所以用 RVA 時建議自己給終點。
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式不會自動加入腳本所在目錄

from iced_x86 import Decoder, Formatter, FormatterSyntax

from re_common import PE, Owners, load_calls


def cstr(pe, rva):
    o = pe.rva2off(rva)
    if o is None:
        return None
    s = pe.d[o:o + 80]
    e = s.find(b"\0")
    s = s[:e] if e >= 0 else s
    return s.decode() if len(s) >= 3 and all(32 <= c < 127 for c in s) else None


def disasm(pe, names, start, end, limit=100000):
    fmt = Formatter(FormatterSyntax.INTEL)
    for n, ins in enumerate(Decoder(64, bytes(pe.read(start, end - start)), ip=pe.base + start)):
        if n >= limit:
            break
        text, extra = fmt.format(ins), ""
        if ins.is_ip_rel_memory_operand:
            t = ins.ip_rel_memory_address - pe.base
            s = cstr(pe, t)
            extra = "  ; rva %x" % t + (' "%s"' % s if s else "")
        if text.startswith(("call", "jmp")):
            try:
                t = ins.near_branch_target - pe.base
                if t in names:
                    more = len(names[t]) - 1
                    extra += "  ; -> " + names[t][0] + (" (+%d 個方法共用)" % more if more else "")
            except Exception:
                pass
        yield "%08x  %s%s" % (ins.ip - pe.base, text, extra)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    pe = PE()
    try:
        names = load_calls(pe)[0]
    except FileNotFoundError:
        names = {}
    own = Owners(names)
    for start in own.find(sys.argv[1]):
        end = int(sys.argv[2], 16) if len(sys.argv) > 2 else (own.end(start) if names else start + 0x400)
        limit = int(sys.argv[3]) if len(sys.argv) > 3 else 100000
        if start in names:
            more = len(names[start]) - 1
            print("==", hex(start), names[start][0] + (" (+%d 個方法共用這段機器碼)" % more if more else ""))
        for line in disasm(pe, names, start, end, limit):
            print(line)
