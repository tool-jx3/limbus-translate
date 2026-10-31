import json, os, sys, collections, re
root = sys.argv[1]
base = os.path.join(root, "kr", "LocalizeTemp_kr")
load = lambda p: json.loads(open(p, "rb").read().decode("utf-8-sig"))
sd = os.path.join(base, "StoryData")
g = collections.Counter(); gchars = collections.Counter(); ex = collections.defaultdict(list)
models = collections.Counter(); fieldsets = collections.Counter()
for f in os.listdir(sd):
    stem = f[3:-5] if f.startswith("KR_") else f[:-5]
    m = re.match(r"([A-Za-z]+)(\d)?", stem); key = (m.group(1) + (m.group(2) or "")) if m else stem
    data = load(os.path.join(sd, f)); dl = data.get("dataList", []) if isinstance(data, dict) else []
    g[key] += 1; gchars[key] += sum(len(x.get("content", "")) for x in dl if isinstance(x, dict))
    if len(ex[key]) < 3: ex[key].append(stem)
    for x in dl:
        if isinstance(x, dict):
            if x.get("model"): models[x["model"]] += 1
            fieldsets[tuple(sorted(k for k in x if k != "id"))] += 1
print("StoryData groups (prefix+first digit):")
for k, n in sorted(g.items(), key=lambda x: -gchars[x[0]])[:20]: print(f"  {k:6s} files={n:4d} chars={gchars[k]:8d} e.g. {', '.join(ex[k])}")
print("\nTop StoryData `model` speakers:", ", ".join(f"{k}({v})" for k, v in models.most_common(30)))
print("\nStoryData item field sets:", fieldsets.most_common(8))
pv = os.path.join(base, "PersonalityVoiceDlg"); who = collections.Counter()
for f in os.listdir(pv):
    m = re.match(r"(?:KR_)?Voice_([A-Za-z]+)_", f); who[m.group(1) if m else f] += 1
print("\nPersonalityVoiceDlg files per character:", dict(who.most_common()))
ann = os.path.join(base, "BattleAnnouncerDlg"); a = collections.Counter()
for f in os.listdir(ann):
    m = re.match(r"(?:KR_)?Announcer_([A-Za-z]+)", f); a[m.group(1) if m else f] += 1
print("\nAnnouncers:", ", ".join(f"{k}({v})" for k, v in a.most_common(25)))
smc = load(os.path.join(base, "KR_ScenarioModelCodes-AutoCreated.json"))
print("\nScenarioModelCodes sample:", json.dumps(smc["dataList"][:3], ensure_ascii=False))
f = sorted(os.listdir(pv))[0]; d = load(os.path.join(pv, f))
print("\nPersonalityVoiceDlg sample", f, json.dumps(d["dataList"][:2], ensure_ascii=False)[:500])
