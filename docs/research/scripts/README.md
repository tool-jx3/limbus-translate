# 調查用分析腳本

產生 `docs/research/2026-10-30-reference-survey.md` 與 `guides/` 裡統計數字的一次性 Python 腳本。之後會由 Go CLI 取代，這裡保留是為了能重算、核對。

## 資料目錄

所有腳本都吃同一個參數 `DATA`，目錄結構如下（下載後自行解壓）：

```text
DATA/
  raw/kr/LocalizeTemp_kr/   # localize_kr.zip
  raw/en/LocalizeTemp_en/   # localize_en.zip
  raw/jp/LocalizeTemp_jp/   # localize_jp.zip
  tw/para/                  # ParatranzUploader release 的 localize_files.zip
  tw/ltm/                   # LimbusTraditionalMandarin/storyline release 的 Hant.zip
```

下載網址：`https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{kr,en,jp}.zip`

## 執行順序

```bash
python3 -I fields3.py DATA       # 三語欄位比對 → 哪些欄位不翻
python3 -I families.py DATA      # 檔案族群與字數
python3 -I story.py DATA         # StoryData 分組、說話者、語音檔角色
python3 -I align.py DATA         # 產生 DATA/units.jsonl（逐欄位對齊 KR/EN/JP/繁中）
python3 -I coverage.py DATA/units.jsonl
python3 -I models.py DATA        # 說話者鍵值 → 繁中名稱
python3 -I charstats.py DATA     # 產生 DATA/charlines.json 並統計各角色
python3 -I charcheck.py DATA     # 核對原指引的具體說法
python3 -I formatcheck.py DATA   # 程式格式須知核對
python3 -I glosscheck.py DATA    # 譯名統一列表核對
python3 -I naming.py DATA        # 檔名規律核對
python3 -I examples.py DATA      # 挑選角色例句
```
