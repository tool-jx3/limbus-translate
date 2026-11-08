# 良秀縮略語研究的統計腳本

`../report.md` 的數字由這些腳本算出。輸入是 `lct site --out DIR` 寫出的網站資料（`DIR/files/<路徑>.json`，每個單位有 KR／JP／EN／TW 與 model、teller、speaker）。

## 用法

在本目錄執行，`LCT_SITE_DATA` 指向網站資料（預設 `site/public/data`）。中間結果（`*.pkl`、`*.txt`）寫在目前目錄，不進版控。

```bash
export LCT_SITE_DATA=/path/to/site/data
python3 -I build_occ.py   # 找出良秀台詞中的縮略語（occ.pkl）
python3 -I build_occ2.py  # 改用修正後的樣式重抓，並加入手動列出的不加點縮略語（occ2.pkl）
python3 -I build_occ3.py  # 同上，另以 trim.py 修剪誤抓的片段（覆寫 occ2.pkl）
python3 -I apply_ov.py    # 套用人工修正（overrides.py）→ occ3.pkl
python3 -I group.py       # 依縮略語分組、計數
python3 -I expand.py      # 找同場景的展開（exp.pkl）
python3 -I table.py       # 產生初版第 3 節的表（table.md；第 2 輪起改用下面兩個）
python3 -I tokens.py      # 每處縮略語在 JP／EN／繁中裡對應的片段（tok.pkl）
python3 -I table2.py      # 產生報告第 3 節的表（table2.md：3.1 出現兩次以上、3.3 只出現一次）
```

## 第 2 輪：JP／EN 異寫與類別

- `tokens.py`：每行抓出 JP「・」片段、EN 大寫縮寫、繁中「．」片段，依位置與該行的 KR 縮略語配對；數目對不上就留空。再以 `tokens_ov.py` 覆蓋。EN 只差空格、末點的寫法視為同一種（`S.H` → `S.H.`）；接 이다 的 `다.` 併入本體（`MERGE`），借用阿賴耶的流行語另列（`BORROWED`）
- `tokens_ov.py`：人工讀過整行後填的片段與修正後的 KR 邊界，鍵是（單位, 縮略語）。同一行有兩個以上縮略語、片段數對不上、或自動抓到前後文的都在這裡（145 處）
- `table2.py`：依 tok.pkl 分組，列出每個縮略語 JP、EN、繁中的所有寫法與次數；`CLS` 是讀過場景後判定的類別（同義異寫／異義／中文不一）與提案編號，`NOTE` 是類別的附註（判定只靠 JP／EN、KR 沒有根據的標「未確認」「待確認」），`EXP` 是出現兩次以上的劇中展開。沒列在 `CLS` 而 JP、EN、繁中有變化的，執行時會報錯

## 輔助

- `common.py`：良秀的 model 清單與縮略語的樣式（加點與不加點）；`load.py`：讀網站資料
- `dump_ryo.py`、`dump_ara.py`：列出良秀、阿賴耶加點的台詞
- `others.py`：其他角色模仿或評論縮略語的台詞
- `scan1.py`、`occ.py`：初步掃描與共用函式
- `scene.py 路徑 起 迄 [語言]`：印出一段場景（例：`scene.py StoryData/S940B.json 10 40 kjez`）

腳本是研究當時的工作版本，數字以 raw-20261105、tw `008ead4` 為準。
