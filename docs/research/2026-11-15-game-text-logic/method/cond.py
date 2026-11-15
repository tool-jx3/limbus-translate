#!/usr/bin/env python3
"""把 il2csharp 的輸出去掉 IL2CPP 執行期的雜訊（型別初始化檢查、寫入屏障等），每行前面加原始行號。

  python3 -I cond.py <檔案.cs> [起始行 結束行]
"""
import re
import sys

if len(sys.argv) < 2:
    sys.exit(__doc__)
src = open(sys.argv[1]).read().split("\n")
a = int(sys.argv[2]) if len(sys.argv) > 2 else 1
b = int(sys.argv[3]) if len(sys.argv) > 3 else len(src)
lines = src[a - 1:b]
NOISE = re.compile(r"il2cpp_object_new\(|data_18\w+ [!=]= (0|null)|\.initialized == 0|\+ 0xe4\)\)\[0\] == 0"
                   r"|= new [\w\.]+\(\);\s*$|_comparer = |EqualityComparer<object>|>> 12 &")
KEEP = ("new List", "new HashSet", "new Dictionary", "new StringBuilder", "new Regex", "new char")
i = 0
while i < len(lines):
    line = lines[i]
    s = line.strip()
    if not s:
        i += 1
        continue
    if s.startswith("if (") and NOISE.search(s) and i + 1 < len(lines) and lines[i + 1].strip() == "{":
        depth, j = 0, i + 1
        while j < len(lines):
            t = lines[j].strip()
            depth += (t == "{") - (t == "}")
            if t == "}" and depth == 0:
                break
            j += 1
        if all(not x.strip() or NOISE.search(x) for x in lines[i + 2:j]):
            i = j + 1
            continue
    if NOISE.search(s) and s.endswith(";") and not any(k in s for k in KEEP):
        i += 1
        continue
    print("%d:%s%s" % (a + i, " " * ((len(line) - len(line.lstrip())) // 4), s))
    i += 1
