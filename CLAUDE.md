# limbus-translate

為《Limbus Company》建立臺灣繁體中文（zh-TW）語言包。以 KR 韓文為主，JP 日文與 EN 英文作為輔助。

## 指令

- `go build -o bin/lct ./cmd/lct`：產生 CLI `bin/lct`（`bin/` 不進版控）
- `go vet ./... && go test ./...`：提交前必跑
- 所有 `lct` 指令支援 `--json`；exit code：0 成功、1 有發現或使用者錯誤、2 系統錯誤

## 規則與狀態

- 翻譯規則在 `guides/`（指引，先讀 `guides/README.md`）、`glossary/`（詞彙表，硬性規定）、`rules/`（欄位與調整規則）。改規則就改這些檔案，不要把規則寫進程式或提示詞
- 狀態（待譯清單、翻譯紀錄、批次租約）只能經由 `lct` 修改，不要手改檔案
- `raw`、`tw` 分支與 Release 只由 CI 寫入；人工介入一律經由 `main`
- skill 與 `prompts/` 內不寫模型名稱；模型由 `--model` 或環境變數 `LCT_MODEL` 給

## 命名與授權

- 專案名稱、Release 名稱不得使用「Claude Code」「Anthropic」或其標誌，不得把本專案稱為 Claude Code；說明文件可以寫「使用 Claude Code 翻譯」
- 程式碼 GPL-3.0（`LICENSE`）；`guides/`、`glossary/`、`rules/` 與 `tw` 分支譯文 CC BY-NC-SA 4.0（`LICENSE-CONTENT`）

## 文件

- 設計：`docs/superpowers/specs/2026-10-31-limbus-zh-tw-translation-design.md`
- 實作計畫：`docs/superpowers/plans/`
