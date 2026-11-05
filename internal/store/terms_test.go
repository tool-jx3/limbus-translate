package store

import (
	"path/filepath"
	"testing"
)

func TestAppendTermsAndLoad(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	if terms, err := s.LoadTerms(); err != nil || len(terms) != 0 {
		t.Fatalf("LoadTerms of a store without terms = %v, %v, want none", terms, err)
	}
	first := []ProvisionalTerm{
		{KR: "우레", ZH: "雷鳴", Key: "A.json|/dataList/0/content", Batch: "b-1", RawTag: "raw-1", Time: "2026-11-05T01:02:03Z"},
		{KR: "우레", ZH: "驚雷", Key: "A.json|/dataList/1/content", Batch: "b-1", RawTag: "raw-1", Time: "2026-11-05T01:02:03Z", Conflict: true},
	}
	added, err := s.AppendTerms(first)
	if err != nil || added != 2 {
		t.Fatalf("AppendTerms = %d, %v, want 2", added, err)
	}
	want := `{"kr":"우레","zh":"雷鳴","key":"A.json|/dataList/0/content","batch":"b-1","raw_tag":"raw-1","time":"2026-11-05T01:02:03Z"}` + "\n" +
		`{"kr":"우레","zh":"驚雷","key":"A.json|/dataList/1/content","batch":"b-1","raw_tag":"raw-1","time":"2026-11-05T01:02:03Z","conflict":true}` + "\n"
	if got := readFile(t, filepath.Join(root, ".meta", "terms.jsonl")); got != want {
		t.Errorf("terms.jsonl =\n%s\nwant\n%s", got, want)
	}
	// The same KR and ZH again is not stored twice, even from another unit.
	again := []ProvisionalTerm{first[0], {KR: "우레", ZH: "雷鳴", Key: "B.json|/dataList/0/content", Batch: "b-2"}, {KR: "번개", ZH: "閃電", Key: "B.json|/dataList/1/content", Batch: "b-2"}}
	if added, err := s.AppendTerms(again); err != nil || added != 1 {
		t.Fatalf("AppendTerms again = %d, %v, want 1", added, err)
	}
	got, err := s.LoadTerms()
	if err != nil || len(got) != 3 || got[2].KR != "번개" {
		t.Errorf("terms = %+v, %v", got, err)
	}
	if added, err := s.AppendTerms(nil); err != nil || added != 0 {
		t.Errorf("AppendTerms(nil) = %d, %v", added, err)
	}
	if _, err := s.AppendTerms([]ProvisionalTerm{{KR: "x", ZH: "y", Key: "nowhere"}}); err == nil {
		t.Error("AppendTerms accepted a key that is no unit key")
	}
}
