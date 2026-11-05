package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The watch reports the events of the log as they come, leaves a line that is
// not whole for the next read, counts what lct translate did with the batches
// a run left, and sums the rounds up.
func TestBatchWatchCountsWhatTheRunsAndTheSettlingDid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "work", "batches", ".log.jsonl")
	var log strings.Builder
	w, err := newBatchWatch(path, &log)
	if err != nil {
		t.Fatal(err)
	}
	add := func(text string) {
		t.Helper()
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(text); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}

	// A submit that is half written when the log is read.
	add(`{"event":"next","batch":"a","file":"A.json","units":40}` + "\n" + `{"event":"sub`)
	w.read(1)
	if got := log.String(); got != "round 1: batch a: 40 units of A.json\n" {
		t.Fatalf("after the first read the log is %q, want the batch handed out and nothing of the line that is not whole", got)
	}
	add(`mit","batch":"a","file":"A.json","units":40,"accepted":39,"released":1}` + "\n" + `{"event":"next","batch":"b","file":"B.json","units":1}` + "\n")
	w.read(1)
	// The run left b with its out file: lct translate submitted it.
	w.settled(&batchEvent{Event: "submit", Batch: "b", File: "B.json", Units: 1, Rejected: 1, NeedsReview: 1}, 0)
	w.endRound(1)
	// The next run left c without one: lct translate released it.
	// A batch of three files is told by its first.
	add(`{"event":"next","batch":"c","file":"C.json","files":3,"units":5}` + "\n")
	w.read(2)
	w.settled(nil, 5)
	w.endRound(2)

	want := strings.Join([]string{
		"round 1: batch a: 40 units of A.json",
		"round 1: batch a submitted: 39 accepted, 0 reused, 0 rejected (0 need review), 1 not submitted",
		"round 1: batch b: 1 unit of B.json",
		"round 1: 2 batches, 41 units; 2 submitted: 39 accepted, 0 reused, 1 rejected (1 need review), 1 not submitted; 0 released after the run",
		"round 2: batch c: 5 units of C.json and 2 more files",
		"round 2: 1 batch, 5 units; 0 submitted: 0 accepted, 0 reused, 0 rejected (0 need review), 0 not submitted; 5 released after the run",
	}, "\n") + "\n"
	if got := log.String(); got != want {
		t.Errorf("the log is\n%s\nwant\n%s", got, want)
	}
	if got, want := w.summary(), "translate: 2 rounds, 3 batches, 46 units; 2 submitted: 39 accepted, 0 reused, 1 rejected (1 need review), 1 not submitted; 5 released after the run"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	w.close()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the log is still there after close: %v", err)
	}
}
