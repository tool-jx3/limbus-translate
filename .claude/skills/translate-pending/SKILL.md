---
name: translate-pending
description: 依規則翻譯待譯清單；由 `lct batch next／submit` 取批次與送出
disable-model-invocation: true
allowed-tools:
  - Bash(bin/lct batch *)
  - Bash(bin/lct pending *)
  - Bash(go build -o bin/lct ./cmd/lct)
  - Read(./**)
  - Edit(./work/**)
---

# 翻譯待譯清單

一次領一個批次，翻完、送出，再領下一批，直到沒有批次可領。指令都在 repo 根目錄執行。

## 限制

- 只寫 `work/` 底下的檔案，也就是 `work/batches/<id>.out.json`。不改 `.meta/`、指引、詞彙表、規則（`guides/`、`glossary/`、`rules/`）或其他任何檔案；狀態只經由 `bin/lct` 改變
- 規則有問題、不清楚或互相矛盾時，照最合理的方式翻，並寫在該單位的 `note`
- 執行 `bin/lct` 時不加 `--model`：模型由環境變數 `LCT_MODEL` 提供；沒有設定時 `batch submit` 會失敗，就停止並回報

## 流程

0. 執行 `go build -o bin/lct ./cmd/lct`。每次開始都執行（增量建置，很快）；失敗就停止並回報
1. 執行 `bin/lct batch next --json`
   - exit code 不是 0：停止，回報錯誤訊息
   - 結果為 `{"batch": null}`：沒有批次可領（有 `stopped` 時是這次執行的回合已用完，一樣照下面結束，下次執行再領）。執行 `bin/lct pending --count --json`，回報它的結果、這次送出的批次數，以及 `released` 不是空的批次與其中的 key（見「退回」），然後結束
   - 否則記下 `batch`（批次 id，以下寫作 `<id>`）、`path`（批次檔）與 `units`（單位數）
2. 讀批次檔，格式見 `schemas/batch.schema.json`。檔案長時分段讀，確定讀到全部 `units`。劇情以外的批次可能含同族群的多個檔案（`files`），每個單位的 `file` 是它所在的檔案，依檔名判斷的事（語氣）以單位的 `file` 為準；`glossary` 條目哪些單位要用，看條目的 `units`。讀 `prompts/translate.md`，再依序讀批次 `guides` 列出的檔案。讀過的檔案只有在內容仍完整留在上下文中時才可以不重讀；不確定就重讀
3. 依 `prompts/translate.md` 翻譯全部單位，寫成 `work/batches/<id>.out.json`，格式見 `schemas/batch-out.schema.json`；物件數要等於 `units`。詞彙表沒有、但應統一譯法的名稱，除了 `note`，也在該單位的 `terms` 登記 `{kr, zh}`（`kr` 用 KR 原文中的完整詞），規則見 `prompts/translate.md`
4. 執行 `bin/lct batch submit <id> --json`：
   - exit 0：全部收下
   - exit 1 且有 JSON 結果：有單位被退回，依下方「退回」處理
   - exit 1 但沒有 JSON 結果：錯誤訊息若說 out 檔不存在或無效，修正後重送，最多重試一次；仍失敗，或是其他錯誤，就停止並回報
   - exit 2：停止，回報錯誤訊息
   - 每次送出後都檢查 `released`（見「退回」）
5. 回到第 1 步

## 退回

`batch submit` 的結果有 `accepted`、`rejected`（`[{key, errors: [{code, severity, class, message}]}]`）、`warnings`、`needs_review`、`released`，另有 `terms`（新記下的暫定詞數）與 `terms_skipped`（略過的登記與理由，不影響單位，不必處理）。

`terms_differ` 是與既有譯名或舊譯不同的暫定詞（`existing` 是名稱表的既有譯名，`old_units` 是 KR 含這個詞的舊譯單位數，`old_with` 是其中含暫定譯法的，`examples` 是舊譯例句）：已記下但沒有生效，之後的批次不會帶它。已收下的單位不必重送；同一批次還沒送出的單位、之後的批次，依例句與批次的 `names` 用舊譯的寫法。

- **要修正的單位**：在 `rejected` 中、不在 `needs_review` 中，而且 `errors` 裡沒有任何一個 `code` 是協定錯誤碼：`lease-lost`、`not-in-batch`、`duplicate-key`、`kr-missing`、`kr-changed`、`needs-review`
- 其他被退回的單位不再處理：`needs_review` 中的已用完嘗試次數，`lct` 已寫入替代文字；含協定錯誤碼的不能重送
- **修正方式**：依 `errors` 的 `message` 修正（`severity` 為 `hard` 的一定要改）。保留整個 `work/batches/<id>.out.json`，只就地修改要修正的單位的條目，其他條目與 `note` 都不動，再執行 `bin/lct batch submit <id> --json` 送出整個檔案。用同一個 `<id>`：這些單位的租約仍屬於這個批次。已收下的單位重送不會有影響；不處理的單位會再被退回，不必理會。重複到沒有要修正的單位為止
- 批次有 `glossary_check: "warn"` 時，`term-missing` 只是警告，不會退回；詞彙表譯名是預設，語境不合可以不用，不用時請在 `note` 寫出 KR 原詞與理由（會列入詞彙統計，不寫也不會退回）。`forbidden-term` 照樣退回
- `term-missing` 預設只是警告，`check: "off"` 的條目不會有；條目有 `check: "hard"` 時，缺少詞彙表指定的譯法才會退回（`glossary_check: "warn"` 的批次除外，見上一點），暫定詞不會：先改用指定的譯法。真有例外（例如 KR 的詞在這裡是別的意思）就不改譯文，在該單位的 `note` 寫出 KR 原詞與理由（如「시즌：劇中季節，非遊戲賽季」）再送出；`note` 沒有寫出原詞，一樣會被退回
- 嘗試次數由 `lct` 計算，到上限的單位會出現在 `needs_review`，不必自己數
- `warnings` 的單位已經收下，不必處理。沒有 `key` 的一筆 `code: rules-dirty` 是整批的警告：規則有沒 commit 的修改，這批紀錄記下的規則 commit 不含這些修改；照樣繼續，最後的回報中提一句
- `released` 不是空的，表示 out 檔漏了這些單位。這是錯誤，不可忽略：在最後的回報中列出批次 id 與這些 key（它們已放回待譯清單，之後會再發放）
