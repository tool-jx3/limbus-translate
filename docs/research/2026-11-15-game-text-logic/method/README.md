# 方法：怎麼從遊戲程式讀出這些規則

這份報告的結論都來自 2026-11-12 版的 `GameAssembly.dll`（Windows x64，165,384,192 bytes）與 `global-metadata.dat`（50,227,844 bytes）。遊戲更新後，位址、enum 成員、甚至規則本身都可能改變；重做時照下面的步驟，腳本都在這個資料夾。

## 腳本

| 腳本 | 用途 |
|---|---|
| [`md_decrypt.py`](md_decrypt.py) | 把加密的 `global-metadata.dat` 還原成標準格式。種子、替換表、位移常數是從 `GameAssembly.dll` 的載入函式自動讀出來的，沒有寫死 |
| [`lift.sh`](lift.sh) | 呼叫 il2csharp：輸出全部型別的宣告，或反編譯指定的型別 |
| [`who.py`](who.py) | 誰呼叫某個方法、某個方法依序呼叫了什麼、哪些方法用到某個字串常值 |
| [`disasm.py`](disasm.py) | 反組譯，標出呼叫目標的方法名稱 |
| [`lit.py`](lit.py) | 在全部字串常值裡搜尋 |
| [`cond.py`](cond.py) | 把反編譯輸出去掉執行期雜訊，方便閱讀 |
| [`kwsim.py`](kwsim.py) | 重現關鍵字自動連結的判斷；比較 KR 與譯文的連結差異 |
| [`packcheck.py`](packcheck.py) | 把譯文與 KR 逐欄位比對結構問題（[04-other.md](../04-other.md) 的數字來源） |
| [`fingerprint.py`](fingerprint.py) | 把報告依賴的程式事實抽成一份「指紋」（`fingerprints/<日期>.json`），遊戲更新後比對哪些變了、要重看哪幾章；`watch.json` 是要盯的方法清單 |
| [`filelist.py`](filelist.py) | 官方檔案清單（`RemoteLocalizeFileList.json`）和語言包的檔案互相比對：哪些檔不會被載入、清單有而資料夾沒有的檔、同一張表的重複 id |
| [`catalog.py`](catalog.py) | 解讀 Addressables 的目錄（`catalog_S1.json`）：每個資源的型別，在哪個 bundle 還是在遊戲本體 |
| [`unityres.py`](unityres.py) | 讀遊戲本體的 `resources.assets` 與場景檔（`level*`、`sharedassets*.assets`）：文字資源（啟動用清單與文字、換行規則）、樣式表、下載設定、靜態標籤對應的文字 id、任一個文字元件的字級／縮字／換行／框的大小（戰鬥氣泡、劇情對話框） |
| `re_common.py` | 共用：工作目錄、PE 檔讀取、方法位址索引 |

前六個與 `fingerprint.py` 的 `watch`、`make` 需要遊戲檔案與 `pip install iced-x86 numpy`；`kwsim.py`、`packcheck.py` 只需要 `raw`、`tw` 分支的資料；`filelist.py`、`catalog.py` 各需要一份遊戲下載的資料，`unityres.py` 需要遊戲安裝資料夾 `LimbusCompany_Data` 裡的 `globalgamemanagers.assets`，加上要讀的 `resources.assets`、`level*` 或 `sharedassets*.assets`（腳本開頭寫了用法），都不需要額外的套件。腳本一律用 `python3 -I` 執行。遊戲的檔案與下載的資料都不放進版控。

## 步驟

```bash
export RE_WORK=~/re-work          # 工作目錄，不要放在版控裡
mkdir -p $RE_WORK
# 放進兩個遊戲檔案：
#   <遊戲>/GameAssembly.dll
#   <遊戲>/LimbusCompany_Data/il2cpp_data/Metadata/global-metadata.dat
git clone https://github.com/HZBHZB1234/il2csharp $RE_WORK/il2csharp
pip install -r $RE_WORK/il2csharp/requirements.txt numpy
```

```bash
python3 -I md_decrypt.py          # → $RE_WORK/global-metadata.std.dat（約 10 秒）
```

```bash
./lift.sh decls                   # → $RE_WORK/decls：25,318 個型別的宣告、script.json、stringliteral.json（約 1.5 分鐘）
```

之後依問題找型別、讀內容、對照反組譯：

```bash
python3 -I lit.py '<ruby|noparse'                                  # 從字串常值找線索
python3 -I who.py literal 4153                                     # 哪些方法用到第 4153 號字串常值
python3 -I who.py callers TMProStringMatcher.GetAppliedAllRichText # 誰呼叫它
./lift.sh 'UI.Utility.TMProStringMatcher' matcher                  # 反編譯這個型別 → $RE_WORK/lift/matcher
python3 -I cond.py $RE_WORK/lift/matcher/Assembly-CSharp/UI/Utility/TMProStringMatcher.cs 1527 1800
python3 -I disasm.py 0x28649c8 0x2864a60                           # 對照反組譯確認判斷式
```

```bash
python3 -I kwsim.py diff .work/raw/kr .work/tw/LimbusCompany_Data/Lang/zh-TW
python3 -I packcheck.py .work/raw/kr .work/tw/LimbusCompany_Data/Lang/zh-TW 5
```

## 遊戲更新之後：比對程式有沒有變

報告是針對一個版本寫的。`fingerprint.py` 把報告依賴的事實抽出來存檔，新版出來後再抽一次，兩份一比就知道哪些結論要重看。

指紋裡有三種東西：

- **類別的欄位**：`TextDataSet` 的每張表、檔案清單的每個欄位、每個 `TextData_*` 類別的欄位（名稱、型別、位移）。多了欄位，代表原本不讀的 key 可能變成會讀了。
- **enum 的成員**：例如 `BUFF_UNIQUE_KEYWORD`（會自動連結的狀態 id 名單）。
- **方法的摘要**：`watch.json` 列的每個方法依序呼叫了什麼、用到哪些字串常值、讀了哪些浮點常數、比較了哪些數值，以及去掉位址之後的機器碼雜湊。例如關鍵字後接字元的清單 `・「」『』.,/&;`、氣泡的 3 秒與 6 秒，都會出現在摘要裡。

`watch.json` 的方法來自三處：報告內文引用的位址、內文以「類別.方法」寫出的名稱、腳本裡 `EXTRA` 列的名稱。它以名稱記錄，所以新版不必重做。

```bash
# 新版的兩個遊戲檔案放進新的工作目錄，照上面的步驟跑 md_decrypt.py 與 ./lift.sh decls，然後：
python3 -I fingerprint.py make fingerprints/2026-11-19.json
python3 -I fingerprint.py diff fingerprints/2026-11-12.json fingerprints/2026-11-19.json
```

`diff` 印出 Markdown：類別欄位的增減、enum 成員的增減、行為有變的方法（呼叫、字串或常數不同）、找不到的方法、只有機器碼不同的方法，並列出要重看的章節。沒有差異時 exit 0，有差異 exit 1。

限制：

- 方法改名或被內嵌會列在「找不到」，要人工看。
- 別的類別多了欄位會讓位移改變，機器碼雜湊就不同；呼叫、字串、常數都沒變時多半不影響結論，所以另外列一節。
- 虛擬呼叫與委派不在「呼叫」裡。
- 報告新增了引用之後，用舊版的工作目錄重跑 `python3 -I fingerprint.py watch` 更新清單。
- 這套比對只在 2026-11-12 這一版上測試過（自己比自己、以及人為修改過的指紋），還沒有真的比過兩個版本。

資料面的變化（新的檔案種類、新檔案、新的 JSON 欄位、新的條目 id、第一次出現的文字標記）不需要遊戲程式，由 `lct` 從 `raw` 分支的快照算：`bin/lct rawdiff [--from <raw tag>] [--to <raw tag>] [--lang kr|jp|en]`，網站的「資料變化」頁（`#/d`）列出每個快照的結果（JP、EN 另外列出和 KR 不同的變化）。兩邊互相對照著看：資料多了一個欄位（例如 2026-11-12 的 `excludeAutoLink`），指紋的類別欄位通常也會多一個，這時就要回頭讀那個類別的程式。

## 1. 還原 metadata

遊戲用 Unity 6000.3.12f1 的 IL2CPP（metadata v39）。`global-metadata.dat` 經過改造，一般的 IL2CPP 工具讀不了：

- 檔頭不是標準的 `0xFAB11BAF` 與版本號，而是 0x324 bytes 的自訂表：67 組 `(大小, 筆數, 位移)`，其中 31 組是真的資料表，其餘是混淆用的假資料；順序被打亂，每張表的實際位移還要再加上一個寫死在程式裡的常數。
- 檔頭與 7 張表（字串常值索引、字串常值、識別字、屬性、方法、欄位、組件）各用一組種子加密：xorshift64（位移 13、7、17）產生狀態，取狀態低 4 個 byte 互相 XOR 後查一張 256 bytes 的替換表，得到的 byte 與資料 XOR。
- 解密程式在 `GameAssembly.dll` 裡，緊接在「開啟 `Metadata\global-metadata.dat`」之後。檔名是在堆疊上用立即值逐段組出來的，所以搜不到字串；`md_decrypt.py` 直接搜尋那道 `mov dword [rsp+20h], 'glob'` 的機器碼來找這個函式，再反組譯它讀出種子、替換表位址、每張加密表在檔頭的欄位位置與位移常數。
- 沒有加密的表，位移常數散在各個讀取函式裡，不必逐一找：各表在檔案裡仍照標準順序緊接排列（4 bytes 對齊），用 7 張加密表當定位點，其餘依大小推算，再用內容檢查（名稱索引是否都落在字串開頭）確認。這一版只有「欄位封送大小」那張表前面多 4 bytes。
- 解密後依標準 v39 的順序重寫檔頭，就得到一般工具能讀的 metadata：51,653 個型別、322,172 個方法、32,350 個字串常值。

**改版後可能要手動調整的**是 `md_decrypt.py` 裡的兩個常數：`TABLES`（哪一組檔頭欄位是哪一張表）與 `GAPS`（表之間多出來的 bytes）。腳本發現對不上時會停下來說明；`python3 -I md_decrypt.py probe` 會印出解密後的檔頭，每組的「大小÷筆數」與位移順序足以重新對應（做法寫在腳本開頭）。如果連載入函式的形狀都變了（找不到那道指令、或加密方式換了），就要重新讀那個函式。

## 2. 反編譯

用 [HZBHZB1234/il2csharp](https://github.com/HZBHZB1234/il2csharp)（r11，commit ef2f543）：

- 先輸出 `Assembly-CSharp` 全部型別的宣告，用來找型別、欄位、方法與位址。
- 需要讀方法內容的型別個別反編譯（每次約 1 分鐘）。整個 `Assembly-CSharp` 一次反編譯會用光 16 GB 記憶體。
- 這個反編譯器的輸出**不能直接相信**：
  - 條件有時相反或變形（`flag >> 31`、`match == null` 其實是「有比對到」）。
  - 宣告裡 `const` 字串的內容是錯的；方法內容裡出現的字串常值才可信。
  - enum 的數值是錯的（成員名稱正確）。`LOCALIZE_LANGUAGE` 實際是 KR＝0、EN＝1、JP＝2。
  - 字串比較常被展開成三份重複的程式；泛型方法沒有內容，要從呼叫處找共用實作的位址再讀反組譯。
- 所以報告裡每一條會影響結論的判斷，都另外對照反組譯確認過；只看反編譯結果、沒有對照的地方標「推定」。
- `who.py` 只找得到直接呼叫。虛擬呼叫與委派找不到；內容相同的小函式（空函式、回傳固定值）會被編譯器合併成同一段機器碼，分不出是誰。

這次的調查分成四個主題平行進行（載入流程、標籤與 ruby、關鍵字的資料流、其他），每個主題的每一項結論都要附上方法名稱與位址，彙整時再抽查關鍵的幾項（英文為底、ruby 的轉換條件、劇情 `//`、選擇事件分頁）。抽查中發現並修正的一處：「未定義的 `[xxx]` 會造成 null 例外」是誤讀，實際是顏色表找不到時回傳第一個顏色、文字顯示 `Unknown`。

## 3. 用官方資料驗證

讀出規則後寫成 `kwsim.py`，拿 `raw` 分支的 KR／EN／JP 官方文字與 `tw` 分支的譯文檢查：

- 專案裡兩個「已在遊戲中確認」的案例（`rules/adjustments/2026-11-04-noparse.yaml`）都重現：「攻擊等級+1」被讀成 `AttackLevelAdder`、「加斥我援」的「援」被讀成 `CheerUpXichun`。
- KR 官方文字裡被 `<noparse>` 包住或後面接 `[TabExplain]` 的名稱，後一個字元是空格的 206 處、文字結尾 35 處、`,` 28 處、換行 6 處，都符合規則；有 9 處是 `)`，照規則不會連結（官方多包了）。
- JP 官方把「～」「始」「種」三個名稱標成 `excludeAutoLink`，KR／EN 沒有；可見這個欄位是各語言自己決定的。

## 沒有放進版控的東西

遊戲檔案、解密後的 metadata、反編譯輸出都只在工作目錄。報告裡只引用說明所需的方法名稱、位址與短片段。
