---
name: weekly-update
description: 在本機演練每週更新：取得最新原文、寫入快照、規劃、翻譯、建置語言包；結果只留在本機 `out/`，不 push、不 publish
disable-model-invocation: true
allowed-tools:
  - Bash(bin/lct fetch *)
  - Bash(bin/lct snapshot *)
  - Bash(bin/lct plan *)
  - Bash(bin/lct batch *)
  - Bash(bin/lct pending *)
  - Bash(bin/lct build *)
  - Bash(go build -o bin/lct ./cmd/lct)
  - Read(./**)
  - Edit(./work/**)
---

# 在本機演練每週更新

依序做 CI 每週做的事：fetch 最新原文 → snapshot → plan → 翻譯 → build。結果只在本機的 `out/`，用來進遊戲測試。指令都在 repo 根目錄執行。

## 限制

- **一切只留在本機**：不執行 `git push`，`raw`、`tw`、`main` 都不 push；不執行 `lct publish`，不建立 Release。這個 skill 的工具也做不到這些。`raw`、`tw` 分支與 Release 只由 CI 寫入
- snapshot 與 plan 會在本機的 `raw`、`tw` 分支與 `.work/` 的 worktree 留下狀態，它們和 CI 的不是同一份，只供這次演練。演練完要回到 CI 的狀態，請人自己清掉本機的 `raw`、`tw` 分支與 `.work/` 後重新 fetch，不要合併
- 翻譯那一步的限制與 `translate-pending` 相同：只寫 `work/` 底下的檔案，不改 `guides/`、`glossary/`、`rules/`；狀態只經由 `bin/lct` 改變
- 執行 `bin/lct` 時不加 `--model`：模型由環境變數 `LCT_MODEL` 提供，沒有設定時 `batch submit` 會失敗，就停止並回報
- 任何一步失敗（exit code 不是 0）就停止並回報錯誤訊息，不跳過那一步

## 流程

0. 執行 `go build -o bin/lct ./cmd/lct`。每次開始都執行；失敗就停止並回報
1. 執行 `bin/lct fetch --out .work/fetch --json`，記下結果的 `token` 與 `created_at`。錯誤訊息說沒有狀態 API 時，請人設定環境變數 `LIMBUS_STATUS_URL`，或提供 `--token` 與 `--created-at` 後重跑，不自己猜
2. 執行 `bin/lct snapshot --from .work/fetch --token <token> --created-at "<created_at>" --json`。結果的 `changed` 是 `false` 表示原文與上次相同，仍繼續（`main` 上的新規則和待譯照樣要處理）
3. 執行 `bin/lct plan --json`，向人回報 `raw_tag`、`pending`（可領的待譯單位數）與 `changes`、`requeue_applied`、`rewrite_applied` 的摘要。`pending` 是 0 就跳過第 4 步
4. 翻譯：讀 `.claude/skills/translate-pending/SKILL.md`，從它的第 0 步照做到「沒有批次可領」，包含它的「限制」與「退回」。到那裡之後回到這個 skill 繼續第 5 步，不在那裡結束
   - 待譯很多時，先告訴人數量，再開始
   - 人要求中途停止，就停在已送出的批次之後；沒翻的單位留在待譯清單，build 照樣會打包，報表列出以 EN 或 KR 暫代的單位
5. 執行 `bin/lct build --json`。結果的 `text_zip`、`complete_zip` 是打包好的語言包，`report_path` 是報表，都在本機 `out/`
6. 向人報告：raw tag、翻譯了幾批、還剩多少待譯、`text_zip` 與 `complete_zip` 的路徑、報表的路徑與其中需要人處理的項目。最後再說一次：沒有 push 任何分支、沒有 `lct publish`，要發布由 CI 做
