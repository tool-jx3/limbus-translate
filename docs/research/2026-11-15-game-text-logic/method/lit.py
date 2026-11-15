#!/usr/bin/env python3
"""在全部字串常值裡用正規表示式搜尋。

  RE_WORK=… python3 -I lit.py '<regex>' [i]      第二個參數給 i 表示不分大小寫
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式不會自動加入腳本所在目錄

import json
import re

from re_common import DECLS

if len(sys.argv) < 2:
    sys.exit(__doc__)
pat = re.compile(sys.argv[1], re.I if len(sys.argv) > 2 else 0)
for x in json.load(open(os.path.join(DECLS, "stringliteral.json"))):
    if pat.search(x["value"]) and len(x["value"]) < 400:
        print(x["index"], repr(x["value"]))
