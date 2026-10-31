import json, subprocess, sys, difflib
A, B = sys.argv[1], sys.argv[2]
names = subprocess.run(["git", "diff", "--name-only", "--diff-filter=M", A, B, "--", "kr/"], capture_output=True, text=True).stdout.split()
def load(rev, path):
    raw = subprocess.run(["git", "show", f"{rev}:{path}"], capture_output=True).stdout
    try: return json.loads(raw.decode("utf-8-sig"))
    except Exception: return None
def flat(v, path, out):
    if isinstance(v, dict):
        for k, x in v.items(): flat(x, path + [k], out)
    elif isinstance(v, list):
        for i, x in enumerate(v): flat(x, path + [i], out)
    elif isinstance(v, str): out.append(("/".join(str(p) for p in path), v))
found = []
for n in names[:int(sys.argv[3])]:
    a, b = load(A, n), load(B, n)
    if a is None or b is None: continue
    fa, fb = [], []; flat(a, [], fa); flat(b, [], fb)
    sa, sb = [v for _, v in fa], [v for _, v in fb]
    sm = difflib.SequenceMatcher(None, sa, sb, autojunk=False)
    ops = [o for o in sm.get_opcodes() if o[0] != "equal"]
    mid_ins = [o for o in ops if o[0] == "insert" and o[2] < len(sa)]
    if mid_ins:
        same_after = sum(1 for i, (pa, va) in enumerate(fa) if i < len(fb) and fb[i][1] != va)
        tail = sum(j2 - j1 for _, i1, i2, j1, j2 in [o for o in ops if o[0] == "equal"]) if False else None
        moved = sum(1 for tag, i1, i2, j1, j2 in sm.get_opcodes() if tag == "equal" and i1 != j1 for _ in range(i2 - i1))
        found.append((n, len(sa), len(sb), [(o[1], o[4] - o[3]) for o in mid_ins][:3], moved))
print(f"scanned {min(len(names), int(sys.argv[3]))} modified kr files; files with mid-file insertions: {len(found)}")
for f in found[:12]: print("  ", f[0], f"strings {f[1]}->{f[2]}", "inserted at (old pos, count):", f[3], "unchanged strings shifted:", f[4])
