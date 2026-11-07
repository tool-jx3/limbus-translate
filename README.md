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
| `tw` | 每個單位的譯文紀錄、待譯清單、報表、每個 Release 的比對紀錄（`.meta/history/<tag>.json`）與建置出來的語言包 | 只有 CI |

Release（`zh-TW-YYYYMMDD.N`）也只由 CI 建立，而且只在語言包的內容和上一個 Release 不同時才建立（見「GitHub Actions 設定」）。人要改什麼，都經由 `main`（見「人工介入」）。CI 對 `main` 不直接 push，只開規則 PR 讓人審核（見「收到規則 PR 怎麼處理」）。

## 網站

譯文對照網站：https://tool-jx3.github.io/limbus-translate/

網站把每個檔案的 KR、JP、EN 原文與 TW 譯文並排顯示（表格與劇本兩種模式），標出每句譯文的來源並可依來源篩選，也能全文搜尋（子字串或正規表示式）。資料只放最新一次建置，由 CI 產生並部署，不 commit 到任何分支。網站用每頁的 `<meta name="robots" content="noindex, nofollow">` 要求搜尋引擎不收錄。`site/public/robots.txt`（`Disallow: /`）也會一起部署，但網站是專案網站（網址帶 `/limbus-translate/`），爬蟲只讀主機根目錄的 `robots.txt`，所以這個檔案在這個網址上不起作用，有效的是 `<meta>`；只有改放在自訂網域的根目錄時它才會被讀到。網站不收集使用者資料，也不使用外部 CDN。

網站另有「審核」頁（`#/r/<suggestions|warnings|notes|todo|scope>`），把人需要看的項目集中成五個分頁（四個加上「詞彙範圍」），網頁唯讀：

- 五個分頁：規則建議、檢查警告、譯者註記、待辦、詞彙範圍（說明見下一項）。待辦與審核 Issue 的項目相同，只放規則建議分頁沒有的種類。
- 檢查警告分頁：同一個詞彙表詞條漏譯的 `term-missing` 警告合成一張「詞彙群組」卡，不再一句一張。標題為「詞彙表：<KR> → <譯名>（<未確認> 處）」；有單位已確認時才加「，<M> 處已確認」，卡片列出該詞條在每個單位的原文與譯文。
- 詞彙群組的 ID：複製群組卡得到 `termgroup/<KR 雜湊>/<譯名雜湊>@<N>「<KR> → <譯名>」→ `，`@N` 是頁面當時還沒確認的單位數。每個單位列各自複製 `warning/term-missing/…`，只確認那一個單位。
- `bin/lct accept --reason '<理由>' 'termgroup/…@N'` 會展開成群組內還沒確認的每個單位的 ID，把整組警告一次記為確認。N 必須等於目前還沒確認的單位數，不符就拒絕（exit 1，什麼都不寫），代表頁面看到的已經不是現況，請重新整理頁面再貼，或改貼逐單位的 ID；ID 的群組已不在本機審核資料時也拒絕（本機審核資料來自 `.work/raw`、`.work/tw` 與 `glossary/`，輸出的 `raw_tag`、`tw_commit` 可與頁面的比對）：把 `.work/raw`、`.work/tw` 更新到頁面的 raw tag 與 tw commit，或重新整理頁面再貼。群組內的單位都確認過了，`@0` 不寫任何東西，歸在輸出的 `empty`。只傳某個單位的 ID 就只確認那個單位，群組的其他單位照舊列出。
- 規則建議分頁多一種「暫定詞」（見「詞彙檢查與暫定詞」），標題為「暫定詞：<KR> → <譯名>」。不採用的暫定詞和其他已否決的建議一樣預設隱藏，顯示時標為已否決。
- 規則建議分頁的不一致譯法可切換為「連同 JP/EN」：KR 常比 JP、EN 籠統，譯法不同可能是對的，所以每種譯法旁列出它的 JP 與 EN，卡片標出「JP/EN 可解釋」「部分可解釋」「JP/EN 相同」，並可依此篩選；`/review-page` 與 `/rules-review` 處理不一致的譯法時也會列出這些，但不因此替你決定。
- 詞彙範圍分頁（`#/r/scope`）：每個詞彙表條目在各檔案族群與頂層分群（檔案樹的頂層節點，如 `StoryData/`、`Skills`）裡，有幾個單位的 KR 命中、其中幾個真的用了條目的譯名。分頁列出「全部條目」的群組用字表，並依用字的分布提出建議：排除檔案（`not_files` 加上 `family:` 或 `group:` 樣式）、限定檔案（`files`）、詞中詞改 `substring` 或 `word`、需要人看（沒有乾淨的切法）、常被說明偏離（譯者常在註記說明不用此譯名，也許該加 `also`）。每張建議卡列出採用後少掉與多出的警告、少檢查的單位與例子，失去的檢查很多的標為預設不採用。資料由 `lct site` 與 `lct glossary scope --json` 以同一份計算（`review/scope.json`），門檻也在其中。
- 詞彙範圍的 ID 與決定：建議的 ID 是 `scope/<KR 雜湊>/<譯名雜湊>/<種類>`（種類 `not-files`、`files`、`substring`、`word`、`mixed`、`explained`），複製後貼給 `/review-page`。採用：skill 編輯 `glossary/terms.yaml` 的那個條目、在 `notes` 記日期與數字、跑 `bin/lct rules lint`；不採用或維持現狀：`bin/lct accept` 記進 `rules/declined.yaml`；`mixed` 與 `explained` 沒有具體修改，人可以不採用，或附上自己的指示（例如加 `also`、排除某族群）；另有「改為退回」（條目加 `check: hard`，`notes` 記日期、應用率與出現數）與「改回只警告」（刪掉 `check`）兩種決定。條目的 ID `glossary/<KR 雜湊>/<譯名雜湊>` 只用來帶指示，`bin/lct accept` 拒絕它。
- 詞條的官方證據：`bin/lct glossary evidence [--kr K]... [--candidates FILE.yaml] [--json]` 只讀 raw 與規則，列出每個詞條（預設全部 approved，`--kr` 可指定）在官方語言包的名稱（KR 整句相同的單位，其對齊的 JP、EN 依配對彙整，附單位 key 與數量）、keyword 名稱的說明，以及每個候選 JP/EN 譯名（條目現有的 jp、en，最常見的官方名稱，與 `--candidates` 給的）在含此詞的對齊單位中的數量（n）、命中（hits）與比率；n 只計有非空對齊文字的單位（去掉標籤與空白後為空的不算），名稱比對的是單位去頭尾空白後的整句 KR；n ≥ 10 且比率 ≥ 0.6 才 ok。用來填 `glossary/terms.yaml` 的 `jp`、`en`、`meaning` 與 `sources`。`--candidates` 是 YAML：`<KR>: {jp: [...], en: [...]}`。
- 用法：看 → 複製項目的文字（`ID「摘要」→ `，單筆或一組）→ 貼進 Claude Code 的對話，用 `/review-page` 說明決定。
  - 沒問題的，skill 執行 `bin/lct accept` 把規則建議與註記記進 `rules/declined.yaml`、檢查警告記進 `rules/accepted.yaml`（確認過的項目預設隱藏，建置報告也不再計入確認過的警告）。
  - 要改的，skill 寫調整（`lct adjust add`）或改規則並附重翻要求檔，讓譯文真的改變，新譯文出來前項目標「待重翻」。
  - 不清楚的，它會先問。
  - 修改都留在 `main` 工作樹，由你 commit 並 push `main`，CI 重建後網站更新。
- 需要本機有 `.work/raw` 與 `.work/tw`；ID 不是目前項目時（worktree 比網頁舊，或項目已解決）整批不寫入，更新 worktree 後再試。
- `bin/lct accept --dry-run ID...` 唯讀，印出 ID 在審核頁上的那一項；整組 ID 另印 `open_units`（頁面的 N 與目前的數），組不在目前審核資料的 ID 列在 `unknown_term_groups`，有不是目前項目的 ID 或 N 不符時 exit 1。
- `bin/lct accept --tidy` 刪除 `rules/accepted.yaml` 裡已不對應任何警告的條目，執行前同樣要先把 `.work/raw` 與 `.work/tw` 更新到最新，否則會刪掉仍然有效的條目，輸出會列出它對照的建置（`raw_tag`、`tw_commit`）。
- KR／JP／EN 原文著作權屬 Project Moon，本站非商業、供翻譯校對使用。

網站的檔案目錄與首頁：

- 首頁有族群表：每個檔案族群（`guides/routing.yaml`）的詞彙檢查是 `hard` 還是 `warn`、收哪些節點與檔數。
- 側欄是依路徑與檔名分出的群組樹：頂層節點（如 `StoryData/`、`Skills`）、子節點，再依檔名分出章節、角色、活動標記；上方有章節、角色、活動三個下拉選單，選了就只列符合的檔案，群組有命中時全部展開。

網站另有「詞彙」頁（`#/g`）：列出每個詞彙表條目、它的範圍、出現與用了譯名的單位數、應用率，可搜尋文字、依族群篩選、只看有沒用譯名的或「可考慮退回」的（`#/g?only=hard`），每欄表頭都可排序（再點一次反向）。點進條目（`#/g/<a>/<b>`）看它的範圍、各群組的用字，與每個命中的單位：沒用譯名的在前，每次顯示 100 個，可依族群、分群、章節、角色、活動、來源與文字篩選；「複製這些單位」複製篩選後全部未確認的單位（`ID「TW」→ `），貼給 `/review-page` 說明決定，改為退回的流程見上面詞彙範圍的 ID 與決定。

網站的「預覽」頁（`#/pv/<檔案路徑>?u=<單位指標>`，檔案頁每個單位旁的「預覽」連結）顯示一個單位若單獨送翻會組成的請求：依哪些指引、詞彙、keywords 與上下文，以及它現有的譯文在今天的送出檢查下會是退回、收下但有警告或收下；內容與 `bin/lct batch preview <key>` 相同，唯讀，頁首標明模擬所依的規則 commit，「全文」分頁可複製送給翻譯者的完整文字。

網站還有「更新」頁（頁首的「更新」，`#/u`），列出每個 Release 和它前一個 Release 比出的變動，最新的在最上面：日期、比對的是哪一版、是遊戲更新（附 raw tag）還是重建，以及新增、KR 修改、刪除、只有譯文改變各幾個，另標出有幾個其他檔案有變；最舊的 Release 前面沒有東西可比，那一列寫「第一版，沒有可比對的上一版」，沒有連結。點進一個 Release（`#/u/<tag>`）看逐條變動：KR 修改與只有譯文改變會把舊的與新的並列，差異處標色，也標出譯文來源前後的變化（例如 `paratranz` → `claude`）；JP／EN 是目前檔案裡的（不是當時的），展開才載入，單位已不在時會寫明；可依種類、來源與檔案篩選，預設隱藏不需翻譯的單位（KR 與 TW 相同的複製單位），取消勾選才看到。篩選列右側的「卡片｜表格」切換顯示方式，表格適合一次掃過很多筆，選擇會記在瀏覽器，重新整理後仍保持。篩選條件在網址裡，可以把網址直接貼給別人。資料來自 `tw` 上的 `.meta/history/<tag>.json`（見「運作方式」的表）。列表以 Release 的 tag 為準（`lct site` 讀 `tw` 的 tag，不是讀紀錄）：每個 Release 都有一列；紀錄還沒補上的 Release 列出名字與日期，沒有數字也沒有連結，等下一次建置補上紀錄才有；tag 沒有成為 Release 的紀錄（例如放棄的準備中版本）不列出。

`pages.yml` 在 `update.yml` 或 `rebuild.yml` 於 `main` 上成功結束後執行（也可以手動執行）：取得 `raw`、`tw` 與 tag 並建 worktree → `go build` 出 `lct` → `bin/lct site --out site/public/data` 輸出網站資料 → 在 `site/` 執行 `npm ci`、`npm run build` → 部署 `site/dist`。`lct site` 失敗時（例如 `raw` 與 `tw` 不是同一個 snapshot）整個執行失敗，不會用缺的或舊的資料部署，網站維持上一次部署的樣子；`TestWorkflowContract` 要求 `pages.yml` 的任何一步都不能被略過或忽略失敗。

### 一次性設定

網址是 `https://<owner>.github.io/<repo>/`，上面是這個 repo 的。fork 或搬到別的 repo 時，部署前要先做一次：到 Settings → Pages，把 Source 選為「GitHub Actions」。沒有這項設定，`pages.yml` 設定 Pages 的那一步會失敗，網站不會部署。`pages.yml` 不使用 secrets 或 variables。手動執行時請選 `main` 分支。

## GitHub Actions 設定

fork 或搬到別的 repo 時，要設定下面的 secrets 與 variables，並確認 workflow 有寫入權限。每次建置，`tw` 都照常 commit 與推送，但語言包沒變就不發 Release（見下面「細節」）。

### Secrets 與 variables

設定位置：repo 的 Settings → Secrets and variables → Actions。Secrets 在「Secrets」分頁，variables 在「Variables」分頁。

| 名稱 | 種類 | 用途 | 必填 | 預設值 | 怎麼取得 |
|---|---|---|---|---|---|
| `ANTHROPIC_API_KEY` | Secret | Claude 的 API key；只有翻譯那一步（`pipeline.yml`）和 `rules-proposal.yml` 的 action 那一步讀得到 | 要翻譯、要自動開規則 PR 時必填。沒有時仍會 plan、build 與發布，只是不翻譯，審核 Issue 會列出「翻譯沒有執行」；`rules-proposal.yml` 有新建議時會失敗 | 無 | 在 Claude Console 建立 API key；費用記在 Console 的帳戶，不是 GitHub |
| `LIMBUS_STATUS_URL` | Secret | 狀態 API 的網址，用來查最新的遊戲更新（token）。網址是第三方服務的，可能更換，所以放 secret；不會出現在記錄裡 | `update.yml` 必填；`rebuild.yml` 用不到 | 無 | 狀態 API 的完整網址（回傳最新的 `latest_token`），向維護者取得 |
| `LCT_MODEL` | Variable | 翻譯用的模型，傳給 `lct translate`；`rules-proposal.yml` 也用它作 action 的 `--model` | 要翻譯、要自動開規則 PR 時必填；沒有時與沒有 API key 一樣，不翻譯，審核 Issue 會列出「翻譯沒有執行」 | 無 | 填 Claude 的模型 ID，可用的 ID 見 Claude Console 或官方文件 |
| `RULES_REVIEW_IN_CI` | Variable | 設成 `true` 時，`rules-proposal.yml` 在 CI 以 action 審查建議並開規則 PR | 選填；不是 `true` 時 CI 只數出待處理的建議，留給人在本機執行 `/rules-review`，維持成功 | 無（在本機審查） | 填 `true`。action 用的是 GitHub Actions 的用量，只適用 Agent SDK、API 等的促銷額度不能用 |
| `LCT_MAX_USD` | Variable | 每次建置翻譯花費的上限（美元），到了就停止翻譯 | 否 | 30 | 自行決定；手動執行時可用 `max_usd` 輸入蓋過 |
| `LIMBUS_CDN_TEMPLATE` | Variable | 下載官方語言包的 CDN 網址範本，含 `{token}` 與 `{lang}` | 否 | 同 `lct.yaml` 的 `cdn_template`：`https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip` | CDN 換了位置才需要設定 |

### Workflow permissions

Settings → Actions → General → Workflow permissions 選「Read and write permissions」。各 workflow 也自己宣告了所需的權限：`contents: write`（推 `raw`、`tw`、tag 與建立 Release）與 `issues: write`（開審核 Issue）；`rules-proposal.yml` 另需 `pull-requests: write`（開規則 PR）與 `id-token: write`（action 向 Claude GitHub App 換 token）。`pages.yml` 只讀 repo，需要的是 `contents: read`、`pages: write` 與 `id-token: write`（部署到 Pages）。

### 各 workflow 何時執行

| workflow | 何時執行 | 做什麼 |
|---|---|---|
| `update.yml` | 每週四 11:35（UTC+8，cron `35 3 * * 4`）；收到 `repository_dispatch`（類型 `limbus_update`）；手動 | 向狀態 API 查最新的 token。沒有新 token、沒有待譯、沒有要套用的重翻或改寫檔，`tw` 上次建置用的就是目前的 `raw`，也沒有還欠著的 Release、沒有只保存了進度卻還沒建置的 commit，就結束。否則：下載並 snapshot 到 `raw` → plan → 翻譯 → 把翻譯進度推到 `tw` → build → 推 `tw` → 語言包有變才建立 Release → 有需要人處理的項目時開審核 Issue |
| `rebuild.yml` | push 到 `main` 且動到 `rules/**`、`glossary/**`、`guides/**` 或 `lct.yaml`；手動 | 不查狀態 API，用目前的 `raw` 重做：plan → 翻譯（規則改動後新增的待譯）→ 把翻譯進度推到 `tw` → build → 推 `tw` → 語言包有變才建立 Release → 審核 Issue |
| `rules-proposal.yml` | `update.yml` 或 `rebuild.yml` 在 `main` 上成功結束後；手動（沒有輸入） | 選規則 PR 的分支（有開著的規則 PR 就用它的，沒有就用 `rules-proposal/<日期>`）→ `bin/lct suggest` 數出還沒在任何規則 PR、也沒被不採用的建議，沒有就結束 → 有的話，`RULES_REVIEW_IN_CI` 是 `true` 才以 `anthropics/claude-code-action@v1` 執行 `/rules-review` skill（否則只在記錄留下待處理的數量，由人在本機執行），修改 `guides/`、`glossary/`、`rules/`，開規則 PR 或更新開著的那個 → 確認 PR 存在，把網址寫在 summary |
| `pages.yml` | `update.yml` 或 `rebuild.yml` 在 `main` 上成功結束後；手動（沒有輸入） | 取得 `raw`、`tw` 與 tag 並建 worktree → build `lct` → `bin/lct site --out site/public/data` 輸出網站資料 → 在 `site/` 執行 `npm ci`、`npm run build` → `actions/configure-pages`、`actions/upload-pages-artifact`（`site/dist`）、`actions/deploy-pages`。任何一步失敗，整個執行就失敗，不部署。網址與要先做的設定見「網站」 |
| `ci.yml` | 每次 push 與 PR | `go vet`、`go test`、`lct rules lint`。PR 改到 `guides/`、`glossary/` 或 `rules/` 時另外執行 `bin/lct impact`，把影響評估寫在該次執行的 summary（見「收到規則 PR 怎麼處理」）。另有 `site` job：在 `site/` 執行 `npm ci`、`npm run typecheck`、`npm test`、`npm run build` |

`update.yml` 與 `rebuild.yml` 都呼叫 `pipeline.yml`（共同的流程，不單獨執行）。`rules-proposal.yml` 與 `pages.yml` 在它們成功結束後才執行，失敗或被取消的不執行，其他分支上的執行也不算。細節：

- 翻譯最多進行 5 小時。每一輪執行到第 100 個回合後就不再領新批次，送出手上那批就結束（`--soft-turns 100`），接著開下一輪；130 個回合是硬上限（`--max-turns 130`），用完也是正常的結束。每輪結束時，這一輪領了卻沒送出的批次：結果檔已寫好的直接替它送出（照樣經過驗證），其餘馬上釋放，下一輪就能再領，不必等鎖定的 2 小時過去；開始翻譯前也先這樣處理一次，清掉上次中斷留下的批次。Actions 的記錄每輪印出開始時的花費與待譯數、結束時用了幾個回合與花費；每領一批印出批次 ID、單位數與檔案，每送出一批印出接受、沿用、退回（其中轉人工審核）、沒送出的單位數，每輪結束印出這輪共領了幾批、幾個單位與送出的結果，翻譯結束再印出所有輪的合計（`lct translate` 執行期間會建立 `work/batches/.log.jsonl`，`lct batch next` 與 `lct batch submit` 把每一批寫進去，`lct translate` 邊跑邊讀；平常手動執行批次指令時沒有這個檔，也不會記錄）；單輪另有 90 分鐘的上限，只用來擋住卡死的執行（被它終止算錯誤，整個翻譯迴圈就此結束）。到了花費上限或時間就停止，沒翻完的留在待譯清單，下次繼續，打包時暫以 EN 或 KR 顯示。翻譯失敗只發警告，已翻的照常 build 與發布。
- 領批次的順序與合批：`BattleKeywords*` 與 `Bufs*` 是關鍵字檔，先發，之後的批次才看得到它們定下的譯名；關鍵字批次只併同一族群的關鍵字檔。非劇情的小檔會與同族群的其他小檔合成一批，單位數上限是 `lct.yaml` 的 `batch.max_units`；劇情批次仍然一個檔一批。批次愈合愈少，輪數與花費隨之下降。
- 詞彙檢查與暫定詞：送出時，`lct batch submit` 依 `glossary/` 檢查譯文有沒有用詞條規定的譯名（詞條預設整詞比對，後接助詞也算，`match: substring` 才比對長詞裡的用法；`files`、`not_files` 限定或排除檔案，項目可寫檔名樣式、`family:<id>`（路由判為該族群的檔）、`group:<樣式>`（檔的節點）、`chapter:<數字>`、`character:<id>` 或 `event:<樣式>`（檔的章節、角色、活動標記）；`also` 列 `zh` 以外同樣正確的譯法，譯文含 `zh` 或任一 `also` 就算用了；同一處 KR 被另一個適用條目的更長命中完全蓋住時，那一處不算，長詞條優先；細節見 `glossary/README.md`）；漏用的 `term-missing` 預設只是內容警告，送出時照樣收下；只有詞條寫了 `check: hard` 才改成退回，而且 `guides/routing.yaml` 標 `glossary: warn` 的族群只警告不退回，`forbidden-term` 在這些族群照舊退回；`check: hard` 的詞條，該單位的註記（`note`）寫了這個詞條的 KR（例如說明為何在此不照詞條譯）就收下。暫定詞沒有 `check`，只警告。詞彙表沒有的詞，譯者可在結果檔的 `terms` 登記暫定譯名；暫定詞存在 `tw` 的 `.meta/terms.jsonl`，同一個 KR 以第一個登記的譯法為準，之後的批次會把它放進詞彙表欄位，送出時同樣檢查，審核頁也列為「暫定詞」供人決定。暫定詞只在送出批次時比對：整詞比對（KR 要以獨立的詞出現才算），KR 至少兩個字，不限檔案；對已有的譯文不產生警告。CI 的 `/rules-review` 不處理暫定詞，由人用 `term-decision` 或 `review-page` 決定。暫定詞不是規則，規則仍只在 `guides/`、`glossary/`、`rules/`。
- 翻譯時每一輪（Claude Code 的一次執行，最多 100 個回合）結束，`lct translate --push-each-run` 就把已翻的內容 commit 並推到 `tw`（commit 訊息 `translate: keep the progress of <raw tag>`，沒有變動就不 commit；推送失敗只發警告，下一輪再推）。所以 runner 中途掛掉，最多只損失正在進行的那一輪。翻譯一結束（包括失敗、被取消或逾時），再把剩下的（最後一輪、`.meta/translate.json`）commit 並推上去；沒有變動就不 commit，但照樣推一次，把前面推送失敗、還留在本地的 commit 推上去（沒有新東西時推送什麼都不做）。這次推送緊接在最後一輪的推送之後，GitHub 偶爾會拒絕（`missing necessary objects`：接收的伺服器還沒看到前一次推送），所以 pipeline 每次推 `tw` 失敗都會在 2、4、8、16 秒後重試，最後一次也失敗才讓步驟失敗。之後 build、發布或推送失敗，已付費翻好的內容也不會丟掉；build 與發布的 commit 接在它們後面。
- 「沒事可做」還要看三件事，任何一件不成立就不算沒事可做：
  - `tw` 上次建置的 raw tag（`tw` 的 `.meta/build.json` 的 `raw_tag`）是不是目前 `raw` 的 tag；`tw` 落後 `raw`（例如上次翻完卻沒建置成功），或讀不到那個檔，這次會補建置與發布。
  - 有沒有還欠著的 Release：`bin/lct publish --check` 只讀 `tw` 的 git 狀態（HEAD、tag、檔案樹），`tw` 的語言包和最新的 Release 不同、HEAD 又沒有 Release 的 tag（上次 `tw` 推上去了，Release 卻沒建成），這次會補建立。
  - `tw` 最後一個 commit 的標題是不是以 `translate: keep the progress of` 開頭：保存進度之後 build 失敗，語言包仍和最新的 Release 相同，上一項看不出來，靠這一項補建置。
- Claude Code CLI 在單獨的一個步驟安裝，那一步沒有 `ANTHROPIC_API_KEY`；key 只在執行 `lct translate` 的那一步。安裝失敗時不翻譯，照常 build 與發布，審核 Issue 列出「翻譯沒有執行」與原因。執行時設定 `DISABLE_AUTOUPDATER=1`，CLI 不會在執行中更新自己。每次執行都帶 `--disallowedTools "Read(./.git/**)"`：翻譯讀得到遊戲原文與譯者備註，那些文字誰都能寫進去，而 checkout 的 token 在 `.git/config`，所以 `.git/` 不讓它讀（同 `rules-proposal.yml`）。
- 語言包沒變就不發 Release：只有語言包的內容和最新一個 `zh-TW-*` Release 不同時才建立。比較這次的 `tw` commit 與該 Release 的 `LimbusCompany_Data/` 檔案樹（git blob），不含 `Info/version.json`（它記錄規則 commit、建置時間、模型與花費，每次建置都不同）。相同就不建立，`tw` 照常 commit 與推送，該次執行的 summary 會寫「語言包與 `<tag>` 相同，這次不建立 Release」。
- 比對紀錄 `.meta/history/<tag>.json`（`tw` 上，給網站的更新頁用）：每個 Release 和它前一個 Release 比出的變動，含新增、原文修改、刪除與只有譯文改變的字串，以及語言包裡其他有變的檔案。`bin/lct publish` 在 commit 之前寫：這次要發的 Release（tag 取 `bin/lct publish` 給的下一個）和最新的 Release 比，語言包有變才寫；沒有任何 Release 時不寫，最舊的 Release 前面沒有東西可比，也沒有紀錄。同一個 tag 還沒發之前重跑，紀錄會依當時的內容重寫。紀錄放在 `.meta/` 底下，不影響「語言包有沒有變」的判斷。已有的 Release 缺紀錄（這個功能之前發的）時，同一次 `bin/lct publish` 會逐一補上，每個和它前一個 Release 比，補齊後再跑就不再有新的；補齊用 `tw` 的 Release tag 與 `raw` 的 tag 讀當時的內容，Release 或它用的 raw tag 讀不到，那一個就補不成，只在 stderr 印 `lct: warning: history: …`、JSON 的 `history_warnings` 列出原因，不影響 commit 與 Release，等 tag 補回來再跑一次即可。`history` 列出這次寫入的紀錄的 tag。
- 審核 Issue 的標題是「審核：YYYY-MM-DD 更新」（日期取 raw tag 的日期）；沒有需要處理的項目時不開。同一個 raw tag 重跑時，內容會留言在還開著的同名 Issue，不另開新的。完整報表在該次執行的 summary。本機的 `bin/lct review`（或 `review` skill）從 `.work/tw` 印出同樣的項目。
- 所有會寫入 `raw`、`tw` 的執行共用 concurrency 群組 `lct-state`，一次只跑一個。GitHub 的群組最多一個執行中、一個等待中，新的等待會取代舊的等待；如果某次被取代，到 Actions 頁面手動執行一次即可。`pages.yml` 只讀 `raw`、`tw`，用自己的群組 `pages`（`cancel-in-progress: false`），一次部署一個；每次都用當時的分支建置，被取代的那次不會漏掉什麼。
- GitHub 只從預設分支執行排程，公開 repo 連續 60 天沒有活動會停用排程（到 Actions 頁面重新啟用）。排程在整點附近可能延遲。

### 規則 PR 要先設定的事（`rules-proposal.yml`）

1. **安裝 Claude GitHub App**：到 https://github.com/apps/claude 安裝，授權給這個 repo。action 以這個 App 的身分開 PR（`id-token: write` 讓它換到 App 的 token）。以 App 開的 PR 才會觸發 `ci.yml`，影響評估才會出現在 PR 上；用 workflow 自己的 `GITHUB_TOKEN` 開的 PR 不會觸發其他 workflow。
2. 設定 `ANTHROPIC_API_KEY` 與 `LCT_MODEL`（見上表）。
3. `rules-proposal` label 不必先建立：`lct proposal publish` 開第一個規則 PR 時會建立。
4. action 裡的 skill 以 `lct proposal publish` commit、推分支並開 PR，它需要 GitHub 的 token（環境變數 `GH_TOKEN` 或 `GITHUB_TOKEN`）。如果 action 那一步的記錄裡 `lct proposal publish` 回報拿不到 token（`gh needs a token`），改用 workflow 自己的 token：
   1. 在 `rules-proposal.yml` 執行 `anthropics/claude-code-action@v1` 的那一步加上 `env:`，內容是 `GH_TOKEN: ${{ github.token }}`。
   2. 到 Settings → Actions → General 勾選「Allow GitHub Actions to create and approve pull requests」。
   3. 這樣開出或更新的 PR 是 `github-actions` 的，不會觸發 `ci.yml`：每次 PR 開出或更新後，到 PR 頁面按「Close pull request」再按「Reopen pull request」，`ci.yml` 才會執行並附上影響評估。
   4. `go test ./...` 的 `TestWorkflowContract` 要求 action 那一步沒有 `GH_TOKEN`，同一個 commit 要拿掉 `cmd/lct/contract_test.go` 裡的這項檢查與它的測試案例，並把改了什麼記在這裡。
5. 擋住 token 外洩：action 讀得到遊戲原文與譯者備註，那些文字誰都能寫進去，所以 checkout 設了 `persist-credentials: false`（token 不留在 `.git/config`），`claude_args` 也用 `--disallowedTools "Read(./.git/**)"` 不讓它讀 `.git/`；`lct proposal publish` 另外拒絕含有 `ghp_`、`gho_`、`ghs_`、`ghu_`、`github_pat_`、`x-access-token`、`eC1hY2Nlc3MtdG9rZW46`（`x-access-token:` 的 base64）、`AUTHORIZATION: basic`、`sk-ant-` 的說明與規則檔修改。因此 action 之前的步驟以匿名方式 fetch（公開 repo 沒有問題）。如果 action 那一步的 `git push` 因為沒有憑證而失敗（記錄裡是 `could not read Username` 或 `Authentication failed`），把 checkout 那一步的 `persist-credentials: false` 刪掉（回到預設，token 留在 `.git/config`），`--disallowedTools "Read(./.git/**)"` 一定要留著；`TestWorkflowContract` 同樣要求 `persist-credentials: false`，同一個 commit 拿掉那項檢查與它的測試案例，並記在這裡。
6. 這次有沒有真的提出建議，由最後一步檢查：skill 把結果寫在 `work/proposal/outcome.txt`（`published`、`nothing: <理由>` 或 `failed: <原因>`）。`published` 時，規則 PR 的說明要有一個開始前不存在的建議，否則算錯誤（`lct proposal publish` 被拒絕時 action 不會失敗，不檢查的話已經開著的 PR 會讓這次顯示成功）；`nothing: …` 只在記錄與 summary 留下理由，維持成功，但 `nothing: 本機…` 是錯誤（skill 沒讀到 `lct proposal start` 寫的 `work/proposal/started.json`，以為在本機而沒有發布）；沒有這個檔或其他內容都是錯誤。沒有設定 `LCT_MODEL` 時 action 不執行，記錄有警告，結果記為 `nothing:`，維持成功；`RULES_REVIEW_IN_CI` 不是 `true` 時也不執行，記錄有 notice 寫明有幾個建議等人在本機執行 `/rules-review`，結果記為 `nothing:`，維持成功。

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
| 決定暫定詞 | `glossary/`，或 `rules/declined.yaml` | 採用：在 `glossary/terms.yaml` 新增該 KR 的詞條（`status: approved`，譯法可與暫定的不同；用 `term-decision` 或 `review-page` skill），詞彙表一有這個 KR，暫定詞就不再生效、也不再列出；不採用：`bin/lct accept --reason '<理由>' 'term-new/<KR>'`（ID 加單引號，KR 可能含空格；記進 `rules/declined.yaml`，該暫定詞不再出現在審核頁） |
| 改進翻譯規則 | `guides/`、`glossary/`、`rules/fields*.yaml` | 改檔案；`glossary/` 的譯名是預設譯名，`check: hard` 的條目缺譯名才退回。規則改了不會自動重翻舊譯文，要重翻必須另外要求。CI 也會提出建議，見「收到規則 PR 怎麼處理」 |

合併到 `main` 後，`rebuild.yml` 會自動執行。重翻要求與改寫檔每個只套用一次（套用過的記在 `tw` 的 `.meta/applied.json`）。

## 收到規則 PR 怎麼處理

每次 `update.yml` 或 `rebuild.yml` 成功之後，`rules-proposal.yml` 會看建置與 plan 發現了什麼規則還沒回答的事：同一句 KR 的不同譯法、新說話者、新欄位、新檔案、譯者的備註、詞彙候選，以及 key 已移動或過期的人工調整。有新建議就由 `rules-review` skill 改 `guides/`、`glossary/`、`rules/`，開一個規則 PR：

- 分支 `rules-proposal/<日期>`（同一天用過就接 `-2`、`-3`），label `rules-proposal`，標題「規則建議：YYYY-MM-DD」，目標分支 `main`。
- 說明逐項列出建議的內容、理由、證據（key、KR、現行譯文、出現次數），最後是影響評估。要重翻的範圍附在 `rules/requeue/*.yaml`。
- 每次執行最多處理 8 個建議（依 `lct suggest` 的順序），其餘的下次執行再處理；譯者備註只在發現它的那次建置之後提出一次。
- 同一時間最多一個開著的規則 PR。已有的話，新建議加進同一個 PR（分支多一個 commit，說明多幾個項目），不另開。每次加之前，`lct proposal start` 會把 `main` 合併進這個分支（程式、skill 與規則就是 `main` 目前的樣子）；合併有衝突就停下並列出衝突的檔案，到分支上把 `main` 合併進去、解掉衝突再重跑。
- 每個項目的第一行是 `<!-- suggestion:<id> -->`，`lct` 靠它知道 PR 有哪些建議；這一行只放標記，標記夾在其他文字中間就不算。編輯說明時不要在這一行加字；除非整個項目都不要，否則不要刪這一行。
- 「需要人決定」一節是 skill 沒有把握的項目：只有標記、一行理由與證據，沒有改任何檔。採用就在 PR 分支補上修改，並在合併前把該項的一節移到「需要人決定」標題之上（留在這一節的項目，`lct proposal start` 不算已處理，會再提出）；不要就照下面第 2 點記進 `rules/declined.yaml`；沒處理就合併的話，這一節的項目都還沒決定，下次執行會再提出（不一致譯法也一樣）。
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

- 提交前執行 `go vet ./... && go test ./...`；動到 `site/` 時另外執行 `cd site && npm ci && npm run typecheck && npm test && npm run build`（Node 版本見 `site/.nvmrc`）。
- 看某個單位單獨送翻會得到什麼批次（族群、指引、詞彙、上下文、待譯原因）以及它現有的譯文在送出檢查下的結果：`bin/lct batch preview <key>`（加 `--json` 輸出 `batch`、`state`、`pending`、`record`、`notes`、`check`）。唯讀，不寫租約、嘗試次數或紀錄；`copy` 紀錄與不在 KR 的 key 會回報錯誤（exit 1）。
- 本機預覽網站：先 `bin/lct site --out site/public/data` 輸出資料（要有 `.work/raw` 與 `.work/tw`），再 `cd site && npm run build && npm run preview`。`site/public/data/`、`site/dist/`、`site/node_modules/` 不進版控。
- 用 Claude Code 開啟本 repo，可以使用 `.claude/skills/` 的 skill：`translate-pending`（翻譯待譯清單）、`adjust`（人工調整）、`weekly-update`（在本機演練每週更新，結果只留在 `out/`）、`rules-review`（把 `lct suggest` 的建議整理成規則修改）、`review`（依 `lct review` 帶你逐項處理審核 Issue 的項目）、`review-page`（處理你從審核頁貼上的項目 ID）、`term-decision`（逐個決定詞彙表的候選條目）。模型由環境變數 `LCT_MODEL` 提供。
- 本機的 `.work/` 與 CI 的不是同一份。`raw`、`tw` 分支與 Release 只由 CI 寫入，不要 push；改動一律走 `main`。

設計與實作計畫見 `docs/superpowers/specs/` 與 `docs/superpowers/plans/`。
