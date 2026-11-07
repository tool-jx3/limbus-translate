package main

import (
	"path/filepath"
	"strings"
	"testing"
)

type evidenceCheckOut struct {
	Lang      string  `json:"lang"`
	Candidate string  `json:"candidate"`
	N         int     `json:"n"`
	Hits      int     `json:"hits"`
	Rate      float64 `json:"rate"`
	OK        bool    `json:"ok"`
}

type evidenceItemOut struct {
	KR    string `json:"kr"`
	ZH    string `json:"zh"`
	JP    string `json:"jp"`
	EN    string `json:"en"`
	Names []struct {
		JP    string   `json:"jp"`
		EN    string   `json:"en"`
		Keys  []string `json:"keys"`
		Count int      `json:"count"`
	} `json:"names"`
	Descs []struct {
		Key string `json:"key"`
		KR  string `json:"kr"`
		JP  string `json:"jp"`
		EN  string `json:"en"`
	} `json:"descs"`
	Checks []evidenceCheckOut `json:"checks"`
}

func evidenceRepo(t *testing.T) string {
	t.Helper()
	root := cleanRules(t)
	copyFile(t, filepath.Join(repoRoot, "lct.yaml"), filepath.Join(root, "lct.yaml"))
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 파열, zh: 破裂, status: approved, en: Rupture}\n- {kr: 출혈, zh: 流血, status: approved}\n- {kr: 후보, zh: 候選, status: candidate}\n")
	writeTestFile(t, root, ".work/raw/meta/source.json", `{"tag": "raw-20261029", "token": "t", "created_at": "2026-10-29 02:55:23", "fetched_at": "2026-10-29T00:00:00Z", "files": {}}`)
	writeTestFile(t, root, ".work/raw/kr/KR_Bufs.json", `{"dataList":[{"id":"Burst","name":"파열","desc":"파열 설명"},{"id":"Bleed","name":"출혈","desc":"출혈 설명"}]}`)
	writeTestFile(t, root, ".work/raw/en/EN_Bufs.json", `{"dataList":[{"id":"Burst","name":"Rupture","desc":"Rupture desc"},{"id":"Bleed","name":"Bleed","desc":"Bleed desc"}]}`)
	writeTestFile(t, root, ".work/raw/jp/JP_Bufs.json", `{"dataList":[{"id":"Burst","name":"破裂","desc":"破裂説明"},{"id":"Bleed","name":"出血","desc":"出血説明"}]}`)
	t.Chdir(root)
	return root
}

func TestEvidenceCommandJSON(t *testing.T) {
	root := evidenceRepo(t)
	writeTestFile(t, root, "cands.yaml", "파열:\n  jp: [破裂]\n  en: [Burst]\n")

	var items []evidenceItemOut
	mustLct(t, &items, "glossary", "evidence", "--candidates", "cands.yaml")
	if len(items) != 2 || items[0].KR != "파열" || items[1].KR != "출혈" {
		t.Fatalf("items = %+v, want the two approved entries", items)
	}
	it := items[0]
	if it.ZH != "破裂" || it.EN != "Rupture" {
		t.Errorf("item = %+v", it)
	}
	if len(it.Names) != 1 || it.Names[0].JP != "破裂" || it.Names[0].EN != "Rupture" || it.Names[0].Count != 1 ||
		len(it.Names[0].Keys) != 1 || it.Names[0].Keys[0] != "Bufs.json|/dataList/0/name" {
		t.Errorf("names = %+v", it.Names)
	}
	if len(it.Descs) != 1 || it.Descs[0].Key != "Bufs.json|/dataList/0/desc" || it.Descs[0].JP != "破裂説明" || it.Descs[0].EN != "Rupture desc" {
		t.Errorf("descs = %+v", it.Descs)
	}
	var got []string
	for _, c := range it.Checks {
		got = append(got, c.Lang+":"+c.Candidate)
		if c.OK || c.N != 2 {
			t.Errorf("check %+v: want N 2 (name and desc) and not ok", c)
		}
	}
	if strings.Join(got, ",") != "JP:破裂,EN:Rupture,EN:Burst" {
		t.Errorf("checks = %v", got)
	}

	// --kr filters; the shape of the output is the same list.
	var one []evidenceItemOut
	mustLct(t, &one, "glossary", "evidence", "--kr", "출혈")
	if len(one) != 1 || one[0].KR != "출혈" || one[0].Names[0].EN != "Bleed" {
		t.Errorf("--kr 출혈 = %+v", one)
	}
}

func TestEvidenceCommandText(t *testing.T) {
	evidenceRepo(t)
	code, stdout, stderr := lct(t, "glossary", "evidence", "--kr", "파열")
	if code != 0 || stderr != "" {
		t.Fatalf("exit code %d, stderr %q", code, stderr)
	}
	for _, want := range []string{"파열", "破裂", "Rupture", "Bufs.json|/dataList/0/name", "Bufs.json|/dataList/0/desc", "破裂説明", "N=2", "Hits=2"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("text output lacks %q:\n%s", want, stdout)
		}
	}
}

func TestEvidenceCommandUserErrors(t *testing.T) {
	root := evidenceRepo(t)
	writeTestFile(t, root, "bad.yaml", "없음:\n  en: [x]\n")
	writeTestFile(t, root, "typo.yaml", "파열:\n  english: [x]\n")
	for _, args := range [][]string{
		{"--kr", "없음"},
		{"--candidates", "missing.yaml"},
		{"--candidates", "bad.yaml"},
		{"--candidates", "typo.yaml"},
		{"extra"},
	} {
		if code, _, stderr := lct(t, append([]string{"glossary", "evidence"}, args...)...); code != 1 || stderr == "" {
			t.Errorf("glossary evidence %v: exit code %d (want 1), stderr %q", args, code, stderr)
		}
	}
}
