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

Release（`zh-TW-YYYYMMDD.N`）也只由 CI 建立，而且只在語言包的內容和上一個 Release 不同時才建立（見「GitHub Actions 設定」）。人要改什麼，都經由 `main`（見「人工介入」）。CI 對 `main` 不直接 push，只開規則 PR 讓人審核（見「收到規則 PR 怎麼處理」）。

## GitHub Actions 設定

fork 或搬到別的 repo 時，要設定下面的 secrets 與 variables，並確認 workflow 有寫入權限。每次建置，`tw` 都照常 commit 與推送，但語言包沒變就不發 Release（見下面「細節」）。

### Secrets 與 variables

設定位置：repo 的 Settings → Secrets and variables → Actions。Secrets 在「Secrets」分頁，variables 在「Variables」分頁。

| 名稱 | 種類 | 用途 | 必填 | 預設值 | 怎麼取得 |
|---|---|---|---|---|---|
| `ANTHROPIC_API_KEY` | Secret | Claude 的 API key；只有翻譯那一步（`pipeline.yml`）和 `rules-proposal.yml` 的 action 那一步讀得到 | 要翻譯、要自動開規則 PR 時必填。沒有時仍會 plan、build 與發布，只是不翻譯；`rules-proposal.yml` 有新建議時會失敗 | 無 | 在 Claude Console 建立 API key；費用記在 Console 的帳戶，不是 GitHub |
| `LIMBUS_STATUS_URL` | Secret | 狀態 API 的網址，用來查最新的遊戲更新（token）。網址是第三方服務的，可能更換，所以放 secret；不會出現在記錄裡 | `update.yml` 必填；`rebuild.yml` 用不到 | 無 | 狀態 API 的完整網址（回傳最新的 `latest_token`），向維護者取得 |
| `LCT_MODEL` | Variable | 翻譯用的模型，傳給 `lct translate`；`rules-proposal.yml` 也用它作 action 的 `--model` | 要翻譯、要自動開規則 PR 時必填 | 無 | 填 Claude 的模型 ID，可用的 ID 見 Claude Console 或官方文件 |
| `LCT_MAX_USD` | Variable | 每次建置翻譯花費的上限（美元），到了就停止翻譯 | 否 | 30 | 自行決定；手動執行時可用 `max_usd` 輸入蓋過 |
| `LIMBUS_CDN_TEMPLATE` | Variable | 下載官方語言包的 CDN 網址範本，含 `{token}` 與 `{lang}` | 否 | 同 `lct.yaml` 的 `cdn_template`：`https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip` | CDN 換了位置才需要設定 |

### Workflow permissions

Settings → Actions → General → Workflow permissions 選「Read and write permissions」。各 workflow 也自己宣告了所需的權限：`contents: write`（推 `raw`、`tw`、tag 與建立 Release）與 `issues: write`（開審核 Issue）；`rules-proposal.yml` 另需 `pull-requests: write`（開規則 PR）與 `id-token: write`（action 向 Claude GitHub App 換 token）。

### 各 workflow 何時執行

| workflow | 何時執行 | 做什麼 |
|---|---|---|
| `update.yml` | 每週四 12:00（UTC+8）；收到 `repository_dispatch`（類型 `limbus_update`）；手動 | 向狀態 API 查最新的 token。沒有新 token、沒有待譯、沒有要套用的重翻或改寫檔，`tw` 上次建置用的就是目前的 `raw`，也沒有還欠著的 Release、沒有只保存了進度卻還沒建置的 commit，就結束。否則：下載並 snapshot 到 `raw` → plan → 翻譯 → 把翻譯進度推到 `tw` → build → 推 `tw` → 語言包有變才建立 Release → 有需要人處理的項目時開審核 Issue |
| `rebuild.yml` | push 到 `main` 且動到 `rules/**`、`glossary/**`、`guides/**` 或 `lct.yaml`；手動 | 不查狀態 API，用目前的 `raw` 重做：plan → 翻譯（規則改動後新增的待譯）→ 把翻譯進度推到 `tw` → build → 推 `tw` → 語言包有變才建立 Release → 審核 Issue |
| `rules-proposal.yml` | `update.yml` 或 `rebuild.yml` 成功結束後；手動（沒有輸入） | 選規則 PR 的分支（有開著的規則 PR 就用它的，沒有就用 `rules-proposal/<日期>`）→ `bin/lct suggest` 數出還沒在任何規則 PR、也沒被不採用的建議，沒有就結束 → 有的話以 `anthropics/claude-code-action@v1` 執行 `/rules-review` skill，修改 `guides/`、`glossary/`、`rules/`，開規則 PR 或更新開著的那個 → 確認 PR 存在，把網址寫在 summary |
| `ci.yml` | 每次 push 與 PR | `go vet`、`go test`、`lct rules lint`。PR 改到 `guides/`、`glossary/` 或 `rules/` 時另外執行 `bin/lct impact`，把影響評估寫在該次執行的 summary（見「收到規則 PR 怎麼處理」） |

`update.yml` 與 `rebuild.yml` 都呼叫 `pipeline.yml`（共同的流程，不單獨執行）。`rules-proposal.yml` 在它們成功結束後才執行，失敗或被取消的不執行。細節：

- 翻譯最多進行 5 小時。每一輪執行最多 100 個回合，用完回合是正常的結束，接著開下一輪；單輪另有 90 分鐘的上限，只用來擋住卡死的執行（被它終止算錯誤，整個翻譯迴圈就此結束）。到了花費上限或時間就停止，沒翻完的留在待譯清單，下次繼續，打包時暫以 EN 或 KR 顯示。翻譯失敗只發警告，已翻的照常 build 與發布。
- 翻譯一結束（包括失敗、被取消或逾時），已翻的內容馬上 commit 並推到 `tw`（commit 訊息 `translate: keep the progress of <raw tag>`，沒有變動就不 commit）。之後 build、發布或推送失敗，已付費翻好的內容也不會丟掉；build 與發布的 commit 接在它後面。
- 「沒事可做」還要看三件事，任何一件不成立就不算沒事可做：
  - `tw` 上次建置的 raw tag（`tw` 的 `.meta/build.json` 的 `raw_tag`）是不是目前 `raw` 的 tag；`tw` 落後 `raw`（例如上次翻完卻沒建置成功），或讀不到那個檔，這次會補建置與發布。
  - 有沒有還欠著的 Release：`bin/lct publish --check` 只讀 `tw` 的 git 狀態（HEAD、tag、檔案樹），`tw` 的語言包和最新的 Release 不同、HEAD 又沒有 Release 的 tag（上次 `tw` 推上去了，Release 卻沒建成），這次會補建立。
  - `tw` 最後一個 commit 的標題是不是以 `translate: keep the progress of` 開頭：保存進度之後 build 失敗，語言包仍和最新的 Release 相同，上一項看不出來，靠這一項補建置。
- Claude Code CLI 在單獨的一個步驟安裝，那一步沒有 `ANTHROPIC_API_KEY`；key 只在執行 `lct translate` 的那一步。執行時設定 `DISABLE_AUTOUPDATER=1`，CLI 不會在執行中更新自己。
- 語言包沒變就不發 Release：只有語言包的內容和最新一個 `zh-TW-*` Release 不同時才建立。比較這次的 `tw` commit 與該 Release 的 `LimbusCompany_Data/` 檔案樹（git blob），不含 `Info/version.json`（它記錄規則 commit、建置時間、模型與花費，每次建置都不同）。相同就不建立，`tw` 照常 commit 與推送，該次執行的 summary 會寫「語言包與 `<tag>` 相同，這次不建立 Release」。
- 審核 Issue 的標題是「審核：YYYY-MM-DD 更新」（日期取 raw tag 的日期）；沒有需要處理的項目時不開。同一個 raw tag 重跑時，內容會留言在還開著的同名 Issue，不另開新的。完整報表在該次執行的 summary。本機的 `bin/lct review`（或 `review` skill）從 `.work/tw` 印出同樣的項目。
- 所有會寫入 `raw`、`tw` 的執行共用 concurrency 群組 `lct-state`，一次只跑一個。GitHub 的群組最多一個執行中、一個等待中，新的等待會取代舊的等待；如果某次被取代，到 Actions 頁面手動執行一次即可。
- GitHub 只從預設分支執行排程，公開 repo 連續 60 天沒有活動會停用排程（到 Actions 頁面重新啟用）。排程在整點附近可能延遲。

### 規則 PR 要先設定的事（`rules-proposal.yml`）

1. **安裝 Claude GitHub App**：到 https://github.com/apps/claude 安裝，授權給這個 repo。action 以這個 App 的身分開 PR（`id-token: write` 讓它換到 App 的 token）。以 App 開的 PR 才會觸發 `ci.yml`，影響評估才會出現在 PR 上；用 workflow 自己的 `GITHUB_TOKEN` 開的 PR 不會觸發其他 workflow。
2. 設定 `ANTHROPIC_API_KEY` 與 `LCT_MODEL`（見上表）。
3. `rules-proposal` label 不必先建立：`lct proposal publish` 開第一個規則 PR 時會建立。
4. action 裡的 skill 以 `lct proposal publish` commit、推分支並開 PR，它需要 GitHub 的 token（環境變數 `GH_TOKEN` 或 `GITHUB_TOKEN`）。如果它回報拿不到 token（`gh needs a token`），修法在實作計畫 `docs/superpowers/plans/2026-11-02-rules-improvement.md` 的 Task 16 第 4 步：由 workflow 把 action 輸出的 `github_token` 傳給那一步，並記在這裡。若改用 workflow 自己的 `GITHUB_TOKEN`，另要在 Settings → Actions → General 勾選「Allow GitHub Actions to create and approve pull requests」。
5. 擋住 token 外洩：action 讀得到遊戲原文與譯者備註，那些文字誰都能寫進去，所以 checkout 設了 `persist-credentials: false`（token 不留在 `.git/config`），`claude_args` 也用 `--disallowedTools "Read(./.git/**)"` 不讓它讀 `.git/`；`lct proposal publish` 另外拒絕含有 `ghp_`、`gho_`、`ghs_`、`ghu_`、`github_pat_`、`x-access-token`、`AUTHORIZATION: basic`、`sk-ant-` 的說明與規則檔修改。因此 action 之前的步驟以匿名方式 fetch（公開 repo 沒有問題）。如果 action 沒有這個設定就推不了分支，由 Task 16 把 `persist-credentials: false` 改回來，`--disallowedTools` 要留著。
6. 這次有沒有真的提出建議，由最後一步檢查：skill 把結果寫在 `work/proposal/outcome.txt`（`published`、`nothing: <理由>` 或 `failed: <原因>`）。`published` 時，規則 PR 的說明要有一個開始前不存在的建議，否則算錯誤（`lct proposal publish` 被拒絕時 action 不會失敗，不檢查的話已經開著的 PR 會讓這次顯示成功）；`nothing: …` 只在記錄與 summary 留下理由，維持成功；沒有這個檔或其他內容都是錯誤。

### 手動執行的輸入

到 Actions 頁面選 workflow，按「Run workflow」。

| 輸入 | 用在 | 說明 |
|---|---|---|
| `force` | `update.yml` | 即使官方原文沒有更新（token 與 `raw` 的快照相同），也重新下載，且不因「沒事可做」而結束，照樣 plan、build、推 `tw`；語言包有變才建立 Release |
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
| 改進翻譯規則 | `guides/`、`glossary/`、`rules/fields*.yaml` | 改檔案；`glossary/` 是硬性規定。規則改了不會自動重翻舊譯文，要重翻必須另外要求。CI 也會提出建議，見「收到規則 PR 怎麼處理」 |

合併到 `main` 後，`rebuild.yml` 會自動執行。重翻要求與改寫檔每個只套用一次（套用過的記在 `tw` 的 `.meta/applied.json`）。

## 收到規則 PR 怎麼處理

每次 `update.yml` 或 `rebuild.yml` 成功之後，`rules-proposal.yml` 會看建置與 plan 發現了什麼規則還沒回答的事：同一句 KR 的不同譯法、新說話者、新欄位、新檔案、譯者的備註、詞彙候選，以及 key 已移動或過期的人工調整。有新建議就由 `rules-review` skill 改 `guides/`、`glossary/`、`rules/`，開一個規則 PR：

- 分支 `rules-proposal/<日期>`（同一天用過就接 `-2`、`-3`），label `rules-proposal`，標題「規則建議：YYYY-MM-DD」，目標分支 `main`。
- 說明逐項列出建議的內容、理由、證據（key、KR、現行譯文、出現次數），最後是影響評估。要重翻的範圍附在 `rules/requeue/*.yaml`。
- 同一時間最多一個開著的規則 PR。已有的話，新建議加進同一個 PR（分支多一個 commit，說明多幾個項目），不另開。每次加之前，`lct proposal start` 會把 `main` 合併進這個分支（程式、skill 與規則就是 `main` 目前的樣子）；合併有衝突就停下並列出衝突的檔案，到分支上把 `main` 合併進去、解掉衝突再重跑。
- 每個項目的第一行是 `<!-- suggestion:<id> -->`，`lct` 靠它知道 PR 有哪些建議。編輯說明時不要刪這一行，除非整個項目都不要。
- 影響評估：PR 改到 `guides/`、`glossary/` 或 `rules/` 時，`ci.yml` 執行 `bin/lct impact`，算出新規則會讓多少既有譯文出現新的硬性問題（舊規則是 `main` 的）。結果在該次 `ci` 執行的 summary（PR 的 Checks 頁籤）。找到有問題的譯文不算失敗，是讓你決定要不要重翻。

怎麼處理：

1. **直接改 PR**。可以改 PR 分支上的檔案，也可以改說明。CI 之後再往同一個 PR 加建議時，會從分支與說明目前的樣子接著做，你的修改會保留。
2. **不要的項目**：把它的 id（說明裡 `<!-- suggestion:` 後面的字串）記進 `rules/declined.yaml`（`id`、`reason`、`date`，`reason` 必填），並還原那一項對規則檔的修改，再合併。之後 `lct suggest` 不會再提這個 id。`bin/lct rules lint` 檢查格式。
3. **合併或關閉**：
   - 合併：`rebuild.yml` 依新規則重建。附有的 `rules/requeue/*.yaml`，合併就是同意重翻，只套用一次。規則改了不會自動重翻舊譯文，沒有附重翻要求檔的項目，舊譯文維持原樣。
   - 關閉而不合併：PR 裡所有項目都視為不採用，不再提出。想留的項目先移到別處。

本機也可以用 Claude Code 執行 `rules-review` skill，流程相同，只是最後不開 PR：修改留在工作樹，由你檢查後 commit 到 `main`。

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
- 用 Claude Code 開啟本 repo，可以使用 `.claude/skills/` 的 skill：`translate-pending`（翻譯待譯清單）、`adjust`（人工調整）、`weekly-update`（在本機演練每週更新，結果只留在 `out/`）、`rules-review`（把 `lct suggest` 的建議整理成規則修改）、`review`（依 `lct review` 帶你逐項處理審核 Issue 的項目）、`term-decision`（逐個決定詞彙表的候選條目）。模型由環境變數 `LCT_MODEL` 提供。
- 本機的 `.work/` 與 CI 的不是同一份。`raw`、`tw` 分支與 Release 只由 CI 寫入，不要 push；改動一律走 `main`。

設計與實作計畫見 `docs/superpowers/specs/` 與 `docs/superpowers/plans/`。
