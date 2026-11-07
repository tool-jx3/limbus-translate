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
python3 -I table.py       # 產生報告第 3 節的表
```

## 輔助

- `common.py`：良秀的 model 清單與縮略語的樣式（加點與不加點）；`load.py`：讀網站資料
- `dump_ryo.py`、`dump_ara.py`：列出良秀、阿賴耶加點的台詞
- `others.py`：其他角色模仿或評論縮略語的台詞
- `scan1.py`、`occ.py`：初步掃描與共用函式
- `scene.py 路徑 起 迄 [語言]`：印出一段場景（例：`scene.py StoryData/S940B.json 10 40 kjez`）

腳本是研究當時的工作版本，數字以 raw-20261105、tw `008ead4` 為準。
