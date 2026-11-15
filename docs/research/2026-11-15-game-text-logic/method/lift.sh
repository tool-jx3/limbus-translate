#!/bin/bash
# 用 il2csharp 反編譯。先跑 md_decrypt.py 產生 global-metadata.std.dat。
#
#   RE_WORK=… ./lift.sh decls                         全部型別的宣告（不含方法內容），約 1.5 分鐘 → $RE_WORK/decls
#   RE_WORK=… ./lift.sh <型別全名的一部分> <輸出名稱> [組件]   含方法內容，約 1 分鐘、1.5 GB 記憶體 → $RE_WORK/lift/<輸出名稱>
#
# 型別名稱是不分大小寫的子字串（'StorySystem.DialogController'；'TextLibrary.' 是整個命名空間）。
# 組件預設 Assembly-CSharp.dll；TextMeshPro 本身在 Unity.TextMeshPro.dll。
# 一次只跑一個；整個 Assembly-CSharp 一起反編譯會用光 16 GB 記憶體。
set -e
W=$(realpath "${RE_WORK:-re-work}")
cd "$W/il2csharp"
common=(--metadata "$W/global-metadata.std.dat" --binary "$W/GameAssembly.dll" -q --json)
if [ "$1" = decls ]; then
  python3 il2csharp.py "${common[@]}" -o "$W/decls" --only Assembly-CSharp --decls-only
else
  out="$W/lift/$2"
  [ -d "$out" ] || python3 il2csharp.py "${common[@]}" -o "$out" --only "${3:-Assembly-CSharp.dll}" --types "$1"
  find "$out" -name '*.cs' | grep -v __SharedBodyStubs
fi
