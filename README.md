# limbus-translate

為《Limbus Company》建立臺灣繁體中文（zh-TW）語言包：每次遊戲更新後，取得官方原文（KR 為主，JP、EN 輔助），使用 Claude Code 翻譯，產出可安裝的語言包並發布成 GitHub Release。

授權：程式碼為 GPL-3.0（見 `LICENSE`）；`guides/`、`glossary/`、`rules/` 與 `tw` 分支的譯文為 CC BY-NC-SA 4.0（見 `LICENSE-CONTENT`）。遊戲原文的著作權屬 Project Moon；語言包內的字型 Sarasa Gothic 為 SIL OFL。

## 安裝語言包

1. 到本 repo 的 Releases，下載最新的 zip。兩種擇一：`zh-TW_text_*.zip` 只有文字；`zh-TW_complete_*.zip` 另含字型。
2. 把 zip 內的 `LimbusCompany_Data/` 解壓到遊戲資料夾。
3. 在遊戲設定選擇 zh-TW。

## 運作方式

| 分支 | 內容 | 誰寫入 |
|---|---|---|
| `main` | 程式、翻譯規則（`guides/`、`glossary/`、`rules/`）、workflow | 人 |
| `raw` | 每次遊戲更新的官方原文快照，一次更新一個 commit 與 tag（`raw-YYYYMMDD`） | 只有 CI |
| `tw` | 每個單位的譯文紀錄、待譯清單、報表與建置出來的語言包 | 只有 CI |

Release（`zh-TW-YYYYMMDD.N`）也只由 CI 建立。人要改什麼，都經由 `main`（見「人工介入」）。

## GitHub Actions 設定

fork 或搬到別的 repo 時，要設定下面的 secrets 與 variables，並確認 workflow 有寫入權限。

### Secrets 與 variables

設定位置：repo 的 Settings → Secrets and variables → Actions。Secrets 在「Secrets」分頁，variables 在「Variables」分頁。

| 名稱 | 種類 | 用途 | 必填 | 預設值 | 怎麼取得 |
|---|---|---|---|---|---|
| `ANTHROPIC_API_KEY` | Secret | 翻譯時 Claude 的 API key；只有翻譯那一步讀得到 | 要翻譯時必填。沒有時仍會 plan、build 與發布，只是不翻譯 | 無 | 在 Claude Console 建立 API key；費用記在 Console 的帳戶，不是 GitHub |
| `LIMBUS_STATUS_URL` | Secret | 狀態 API 的網址，用來查最新的遊戲更新（token）。網址是第三方服務的，可能更換，所以放 secret；不會出現在記錄裡 | `update.yml` 必填；`rebuild.yml` 用不到 | 無 | 狀態 API 的完整網址（回傳最新的 `latest_token`），向維護者取得 |
| `LCT_MODEL` | Variable | 翻譯用的模型，傳給 `lct translate` | 要翻譯時必填 | 無 | 填 Claude 的模型 ID，可用的 ID 見 Claude Console 或官方文件 |
| `LCT_MAX_USD` | Variable | 每次建置翻譯花費的上限（美元），到了就停止翻譯 | 否 | 30 | 自行決定；手動執行時可用 `max_usd` 輸入蓋過 |
| `LIMBUS_CDN_TEMPLATE` | Variable | 下載官方語言包的 CDN 網址範本，含 `{token}` 與 `{lang}` | 否 | 同 `lct.yaml` 的 `cdn_template`：`https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip` | CDN 換了位置才需要設定 |

### Workflow permissions

Settings → Actions → General → Workflow permissions 選「Read and write permissions」。各 workflow 也自己宣告了所需的權限：`contents: write`（推 `raw`、`tw`、tag 與建立 Release）與 `issues: write`（開審核 Issue）。

### 三個 workflow 何時執行

| workflow | 何時執行 | 做什麼 |
|---|---|---|
| `update.yml` | 每週四 12:00（UTC+8）；收到 `repository_dispatch`（類型 `limbus_update`）；手動 | 向狀態 API 查最新的 token。沒有新 token、沒有待譯、沒有要套用的重翻或改寫檔，而且 `tw` 上次建置用的就是目前的 `raw` 就結束。否則：下載並 snapshot 到 `raw` → plan → 翻譯 → 把翻譯進度推到 `tw` → build → 推 `tw` → 建立 Release → 有需要人處理的項目時開審核 Issue |
| `rebuild.yml` | push 到 `main` 且動到 `rules/**`、`glossary/**`、`guides/**` 或 `lct.yaml`；手動 | 不查狀態 API，用目前的 `raw` 重做：plan → 翻譯（規則改動後新增的待譯）→ 把翻譯進度推到 `tw` → build → 推 `tw` → 建立 Release → 審核 Issue |
| `ci.yml` | 每次 push 與 PR | `go vet`、`go test`、`lct rules lint` |

`update.yml` 與 `rebuild.yml` 都呼叫 `pipeline.yml`（共同的流程，不單獨執行）。細節：

- 翻譯最多進行 5 小時。每一輪執行最多 100 個回合，用完回合是正常的結束，接著開下一輪；單輪另有 90 分鐘的上限，只用來擋住卡死的執行（被它終止算錯誤，整個翻譯迴圈就此結束）。到了花費上限或時間就停止，沒翻完的留在待譯清單，下次繼續，打包時暫以 EN 或 KR 顯示。翻譯失敗只發警告，已翻的照常 build 與發布。
- 翻譯一結束（包括失敗、被取消或逾時），已翻的內容馬上 commit 並推到 `tw`（commit 訊息 `translate: keep the progress of <raw tag>`，沒有變動就不 commit）。之後 build、發布或推送失敗，已付費翻好的內容也不會丟掉；build 與發布的 commit 接在它後面。
- 「沒事可做」還要看 `tw` 上次建置的 raw tag（`tw` 的 `.meta/build.json` 的 `raw_tag`）是不是目前 `raw` 的 tag；`tw` 落後 `raw`（例如上次翻完卻沒建置成功），或讀不到那個檔，就不算沒事可做，這次會補建置與發布。
- Claude Code CLI 在單獨的一個步驟安裝，那一步沒有 `ANTHROPIC_API_KEY`；key 只在執行 `lct translate` 的那一步。執行時設定 `DISABLE_AUTOUPDATER=1`，CLI 不會在執行中更新自己。
- 只有 `tw` 有新 commit、或該 commit 還沒有 Release 時才建立 Release。
- 審核 Issue 的標題是「審核：YYYY-MM-DD 更新」（日期取 raw tag 的日期）；沒有需要處理的項目時不開。同一個 raw tag 重跑時，內容會留言在還開著的同名 Issue，不另開新的。完整報表在該次執行的 summary。
- 所有會寫入 `raw`、`tw` 的執行共用 concurrency 群組 `lct-state`，一次只跑一個。GitHub 的群組最多一個執行中、一個等待中，新的等待會取代舊的等待；如果某次被取代，到 Actions 頁面手動執行一次即可。
- GitHub 只從預設分支執行排程，公開 repo 連續 60 天沒有活動會停用排程（到 Actions 頁面重新啟用）。排程在整點附近可能延遲。

### 手動執行的輸入

到 Actions 頁面選 workflow，按「Run workflow」。

| 輸入 | 用在 | 說明 |
|---|---|---|
| `force` | `update.yml` | 即使官方原文沒有更新（token 與 `raw` 的快照相同），也重新下載，且不因「沒事可做」而結束，照樣 plan、build、發布 |
| `requeue` | 兩者 | 一個檔名或 glob（例如 `StoryData/S1*`）：plan 之後把這個範圍的譯文送回待譯清單重翻，提示為「手動執行 workflow 要求重翻」。只能給一個範圍（相當於 `lct requeue --file`） |
| `max_usd` | 兩者 | 這次翻譯花費的上限（美元）；空白時用 `LCT_MAX_USD`，再沒有就是 30 |
| `dry_run` | 兩者 | 勾選時不翻譯（不花 API 費用）；plan、build 與發布照常，待譯的部分以 EN 或 KR 暫代 |

## 人工介入

人不直接改 `raw`、`tw` 或 Release。想改譯文或規則，都寫成檔案、經由 `main`（commit 與 push，或開 PR 合併）：

| 想做的事 | 放在 | 怎麼做 |
|---|---|---|
| 修正某個單位的譯文 | `rules/adjustments/` | 用 `adjust` skill，或 `bin/lct adjust add`；只在該單位的 KR 沒變時有效，KR 改了就會重翻，舊調整當參考 |
| 要求某個範圍重翻 | `rules/requeue/` | `bin/lct requeue --file GLOB --reason R --to-file TOPIC`；CI 套用一次，之後不再套用 |
| 對既有譯文做機械式改寫（例如統一寫法） | `rules/rewrite/` | 寫改寫檔（格式見設計文件 §11.1）；CI 套用一次，不送回翻譯 |
| 改進翻譯規則 | `guides/`、`glossary/`、`rules/fields*.yaml` | 改檔案；`glossary/` 是硬性規定。規則改了不會自動重翻舊譯文，要重翻必須另外要求 |

合併到 `main` 後，`rebuild.yml` 會自動執行。重翻要求與改寫檔每個只套用一次（套用過的記在 `tw` 的 `.meta/applied.json`）。

## 在本機使用

需要 Go（版本見 `go.mod`）與 git。

```sh
git clone https://github.com/tool-jx3/limbus-translate.git
cd limbus-translate
go build -o bin/lct ./cmd/lct
```

`lct` 讀兩個 worktree：`.work/raw`（原文快照）與 `.work/tw`（現行譯文）。新 clone 要先從 `raw`、`tw` 分支建立：

```sh
git fetch --tags origin raw tw
git worktree add .work/raw raw
git worktree add .work/tw tw
```

之後 `bin/lct search`、`bin/lct pending --count`、`bin/lct build` 等指令（`bin/lct help` 列出全部，都支援 `--json`）就能在本機執行。`.work/`、`bin/`、`out/` 不進版控。

- 提交前執行 `go vet ./... && go test ./...`。
- 用 Claude Code 開啟本 repo，可以使用 `.claude/skills/` 的 skill：`translate-pending`（翻譯待譯清單）、`adjust`（人工調整）、`weekly-update`（在本機演練每週更新，結果只留在 `out/`）。模型由環境變數 `LCT_MODEL` 提供。
- 本機的 `.work/` 與 CI 的不是同一份。`raw`、`tw` 分支與 Release 只由 CI 寫入，不要 push；改動一律走 `main`。

設計與實作計畫見 `docs/superpowers/specs/` 與 `docs/superpowers/plans/`。
