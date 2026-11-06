package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

func runPreview(t *testing.T, root string, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", jsonOut)
	err := runBatchPreview(ctx, args)
	return out.String(), err
}

func TestBatchPreviewJSON(t *testing.T) {
	root, _ := batchRepo(t)
	out, err := runPreview(t, root, true, batchKey0)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &top); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	var keys []string
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if got := strings.Join(keys, ","); got != "batch,check,notes,pending,record,state" {
		t.Errorf("keys = %s", got)
	}
	var b struct{ ID string }
	if err := json.Unmarshal(top["batch"], &b); err != nil || b.ID != "preview" {
		t.Errorf("batch id = %q, %v", b.ID, err)
	}
}

func TestBatchPreviewText(t *testing.T) {
	root, st := batchRepo(t)
	const kr = "특정 추출 시작"
	writeTestFile(t, filepath.Join(root, ".work", "raw"), "kr/KR_Term.json", `{"dataList":[{"id":1,"content":"`+kr+`"}]}`)
	const key = "Term.json|/dataList/0/content"
	if err := st.SaveRecords(map[string]store.Record{
		key: {Key: key, KRSha: units.KRSha(kr), ZH: "開始特定提取", Origin: store.OriginParatranz, Since: "raw-20261029"},
	}); err != nil {
		t.Fatal(err)
	}
	out, err := runPreview(t, root, false, key)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"family", "guides/", "특정 추출 → 特定提取", "check", "paratranz"} {
		if !strings.Contains(out, want) {
			t.Errorf("text lacks %q:\n%s", want, out)
		}
	}
}

func TestBatchPreviewCopyIsUserError(t *testing.T) {
	root, st := batchRepo(t)
	if err := st.SaveRecords(map[string]store.Record{
		batchKey1: {Key: batchKey1, KRSha: units.KRSha("뫼르소가 말했다"), ZH: "뫼르소가 말했다", Origin: store.OriginCopy, Since: "raw-20261029"},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := runPreview(t, root, true, batchKey1)
	if !isUserError(err) || !strings.Contains(err.Error(), "copy") {
		t.Errorf("err = %v, want a user error that mentions copy", err)
	}
}

func TestBatchPreviewUnknownKey(t *testing.T) {
	root, _ := batchRepo(t)
	for _, args := range [][]string{{"Nope.json|/dataList/0/content"}, {"Sample.json|/dataList/9/content"}, {"bad"}, {}, {batchKey0, batchKey1}} {
		if _, err := runPreview(t, root, true, args...); !isUserError(err) {
			t.Errorf("args %v: err = %v, want a user error", args, err)
		}
	}
}

func readMeta(t *testing.T, root string) map[string]string {
	t.Helper()
	got := map[string]string{}
	dir := filepath.Join(root, ".work", "tw", ".meta")
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		got[p] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBatchPreviewReadOnly(t *testing.T) {
	root, _ := batchRepo(t)
	before := readMeta(t, root)
	for _, key := range []string{batchKey0, batchKey1} {
		if _, err := runPreview(t, root, true, key); err != nil {
			t.Fatal(err)
		}
		if _, err := runPreview(t, root, false, key); err != nil {
			t.Fatal(err)
		}
	}
	after := readMeta(t, root)
	if len(before) != len(after) {
		t.Fatalf("files %d -> %d", len(before), len(after))
	}
	for p, b := range before {
		if after[p] != b {
			t.Errorf("%s changed", p)
		}
	}
}

func TestBatchPreviewKeywordSpace(t *testing.T) {
	root, st := keywordRepo(t)
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	recs[statusKey] = store.Record{Key: statusKey, KRSha: units.KRSha(statusKR), ZH: "使目標增加2級破裂強度", Origin: store.OriginClaude, Since: "raw-20261029"}
	if err := st.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	out, err := runPreview(t, root, true, statusKey)
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ Check struct{ ZH string } }
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Check.ZH != "使目標增加2級破裂 強度" {
		t.Errorf("check.zh = %q, want the space after 破裂", got.Check.ZH)
	}
}
