---
name: translate-pending
description: 依規則翻譯待譯清單；由 `lct batch next／submit` 取批次與送出
disable-model-invocation: true
allowed-tools:
  - Bash(bin/lct *)
  - Bash(go build -o bin/lct ./cmd/lct)
  - Read
  - Write
---

# 翻譯待譯清單

一次領一個批次，翻完、送出，再領下一批，直到沒有批次可領。指令都在 repo 根目錄執行。

## 限制

- 只寫 `work/` 底下的檔案，也就是 `work/batches/<id>.out.json`。不改 `.meta/`、指引、詞彙表、規則（`guides/`、`glossary/`、`rules/`）或其他任何檔案；狀態只經由 `bin/lct` 改變
- 規則有問題、不清楚或互相矛盾時，照最合理的方式翻，並寫在該單位的 `note`
- 執行 `bin/lct` 時不加 `--model`：模型由環境提供

## 流程

0. `bin/lct` 不存在時，先執行 `go build -o bin/lct ./cmd/lct`
1. 執行 `bin/lct batch next --json`
   - exit code 不是 0：停止，回報錯誤訊息
   - 結果為 `{"batch": null}`：沒有批次可領。執行 `bin/lct pending --count --json`，回報它的結果與這次送出的批次數，然後結束
   - 否則記下 `batch`（批次 id，以下寫作 `<id>`）、`path`（批次檔）與 `units`（單位數）
2. 讀批次檔，格式見 `schemas/batch.schema.json`。檔案長時分段讀，確定讀到全部 `units`。讀 `prompts/translate.md`，再依序讀批次 `guides` 列出的檔案；這個工作階段已經讀過的不必重讀
3. 依 `prompts/translate.md` 翻譯全部單位，寫成 `work/batches/<id>.out.json`，格式見 `schemas/batch-out.schema.json`；物件數要等於 `units`
4. 執行 `bin/lct batch submit <id> --json`：
   - exit 0：全部收下
   - exit 1 且有 JSON 結果：有單位被退回，依下方「退回」處理
   - exit 1 但沒有 JSON 結果：錯誤訊息若說 out 檔不存在或無效，修正後再送一次（不算嘗試次數）；其他錯誤就停止並回報
   - exit 2：停止，回報錯誤訊息
5. 回到第 1 步

## 退回

`batch submit` 的結果有 `accepted`、`rejected`（`[{key, errors: [{code, severity, class, message}]}]`）、`warnings`、`needs_review`、`released`。

- `needs_review` 中的單位已用完嘗試次數，`lct` 已寫入替代文字：不再處理
- `rejected` 中 `errors` 含下列任一 `code` 的單位不重送：`lease-lost`、`not-in-batch`、`duplicate-key`、`kr-missing`、`kr-changed`、`needs-review`
- 其餘 `rejected` 的單位：依 `errors` 的 `message` 修正譯文（`severity` 為 `hard` 的一定要改），把 `work/batches/<id>.out.json` 改寫成只含這些單位，再執行一次 `bin/lct batch submit <id> --json`，重複到沒有要重送的單位為止。這些單位的租約仍屬於這個批次，所以用同一個 `<id>` 重送
- 嘗試次數由 `lct` 計算，到上限的單位會出現在 `needs_review`，不必自己數
- `warnings` 的單位已經收下，不必處理；`released` 是沒送出的單位，已放回待譯清單
