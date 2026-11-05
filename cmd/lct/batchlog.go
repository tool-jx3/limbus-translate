package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/tool-jx3/limbus-translate/internal/cli"
)

// batchLogFile is made by lct translate for as long as it runs: while it is
// there, lct batch next and lct batch submit add a line (a batchEvent) to it
// for every batch they hand out and take in, and lct translate reports those
// as the runs go (batchWatch). Without it, which is how a person runs the
// batch commands, they log nothing.
const batchLogFile = batchDir + "/.log.jsonl"

// batchEvent is a line of the batch log.
type batchEvent struct {
	// Event is "next" (lct batch next handed out the batch) or "submit" (lct
	// batch submit took it in).
	Event string `json:"event"`
	Batch string `json:"batch"`
	File  string `json:"file,omitempty"`
	// Units is the number of units of the batch.
	Units int `json:"units"`
	// What a submit did with them: counts of batch.SubmitResult.
	Accepted    int `json:"accepted,omitempty"`
	Reused      int `json:"reused,omitempty"`
	Rejected    int `json:"rejected,omitempty"`
	NeedsReview int `json:"needs_review,omitempty"`
	Released    int `json:"released,omitempty"`
}

// submitEvent is the event of a submit of a batch of units units and file.
func submitEvent(id, file string, units int, rep *submitReport) batchEvent {
	return batchEvent{
		Event: "submit", Batch: id, File: file, Units: units,
		Accepted: len(rep.Accepted), Reused: len(rep.Reused), Rejected: len(rep.Rejected),
		NeedsReview: len(rep.NeedsReview), Released: len(rep.Released),
	}
}

// logBatchEvent adds ev to the batch log when lct translate keeps one, and
// does nothing otherwise. A log that cannot be written is only warned about:
// the batch was handed out or taken in all the same.
func logBatchEvent(ctx *cli.Context, ev batchEvent) {
	f, err := os.OpenFile(filepath.Join(ctx.Root, filepath.FromSlash(batchLogFile)), os.O_WRONLY|os.O_APPEND, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err == nil {
		var line []byte
		if line, err = json.Marshal(ev); err == nil {
			_, err = f.Write(append(line, '\n'))
		}
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "lct: warning: the batch log of lct translate (%s): %v\n", batchLogFile, err)
	}
}

// batchTally counts the batches of a round, or of every round.
type batchTally struct {
	// Batches and Units were handed out.
	Batches, Units int
	// Submitted batches, and what was done with their units.
	Submitted, Accepted, Reused, Rejected, NeedsReview, NotSubmitted int
	// Released are units of batches a run left that lct translate released
	// after it.
	Released int
}

func (t *batchTally) add(o batchTally) {
	t.Batches += o.Batches
	t.Units += o.Units
	t.Submitted += o.Submitted
	t.Accepted += o.Accepted
	t.Reused += o.Reused
	t.Rejected += o.Rejected
	t.NeedsReview += o.NeedsReview
	t.NotSubmitted += o.NotSubmitted
	t.Released += o.Released
}

func (t batchTally) String() string {
	return fmt.Sprintf("%s, %s; %d submitted: %s; %d released after the run",
		counted(t.Batches, "batch", "batches"), counted(t.Units, "unit", "units"), t.Submitted,
		submitCounts(t.Accepted, t.Reused, t.Rejected, t.NeedsReview, t.NotSubmitted), t.Released)
}

// counted is n with the word for one or for more.
func counted(n int, one, more string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, more)
}

func submitCounts(accepted, reused, rejected, needsReview, notSubmitted int) string {
	return fmt.Sprintf("%d accepted, %d reused, %d rejected (%d need review), %d not submitted",
		accepted, reused, rejected, needsReview, notSubmitted)
}

// batchWatch is the reading side of the batch log, for lct translate: it
// reports every batch a run hands out and takes in as the run goes, and what
// the batches of each round and of every round came to.
type batchWatch struct {
	mu   sync.Mutex
	path string
	log  io.Writer
	// off is where the next line of the log begins.
	off          int64
	round, total batchTally
	rounds       int
}

// newBatchWatch makes the batch log at path, empty: whatever an earlier lct
// translate left in it is not of these runs.
func newBatchWatch(path string, log io.Writer) (*batchWatch, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		return nil, err
	}
	return &batchWatch{path: path, log: log}, nil
}

// close removes the log, so that the batch commands log nothing once lct
// translate is over.
func (w *batchWatch) close() {
	if err := os.Remove(w.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(w.log, "removing %s: %v\n", w.path, err)
	}
}

// read reports the lines added to the log since the last read, as events of
// round. A line that is not whole yet is left for the next read.
func (w *batchWatch) read(round int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fi, err := os.Stat(w.path)
	if err != nil || fi.Size() <= w.off {
		if err != nil {
			fmt.Fprintf(w.log, "round %d: reading the batch log: %v\n", round, err)
		}
		return
	}
	f, err := os.Open(w.path)
	if err != nil {
		fmt.Fprintf(w.log, "round %d: reading the batch log: %v\n", round, err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.NewSectionReader(f, w.off, fi.Size()-w.off))
	if err != nil {
		fmt.Fprintf(w.log, "round %d: reading the batch log: %v\n", round, err)
		return
	}
	end := bytes.LastIndexByte(data, '\n')
	if end < 0 {
		return
	}
	w.off += int64(end + 1)
	for _, line := range bytes.Split(data[:end], []byte("\n")) {
		var ev batchEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			fmt.Fprintf(w.log, "round %d: a line of the batch log is not an event: %s\n", round, line)
			continue
		}
		w.event(round, ev)
	}
}

func (w *batchWatch) event(round int, ev batchEvent) {
	switch ev.Event {
	case "next":
		w.round.Batches++
		w.round.Units += ev.Units
		fmt.Fprintf(w.log, "round %d: batch %s: %s of %s\n", round, ev.Batch, counted(ev.Units, "unit", "units"), ev.File)
	case "submit":
		w.submitted(ev)
		fmt.Fprintf(w.log, "round %d: batch %s submitted: %s\n", round, ev.Batch,
			submitCounts(ev.Accepted, ev.Reused, ev.Rejected, ev.NeedsReview, ev.Released))
	default:
		fmt.Fprintf(w.log, "round %d: the batch log has an event of the unknown kind %q\n", round, ev.Event)
	}
}

func (w *batchWatch) submitted(ev batchEvent) {
	w.round.Submitted++
	w.round.Accepted += ev.Accepted
	w.round.Reused += ev.Reused
	w.round.Rejected += ev.Rejected
	w.round.NeedsReview += ev.NeedsReview
	w.round.NotSubmitted += ev.Released
}

// settled counts what lct translate did after a run with a batch the run
// left: ev when it submitted the batch, released the units it freed.
func (w *batchWatch) settled(ev *batchEvent, released int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if ev != nil {
		w.submitted(*ev)
	}
	w.round.Released += released
}

// endRound reports what the batches of round came to, and starts the next.
func (w *batchWatch) endRound(round int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.log, "round %d: %s\n", round, w.round)
	w.total.add(w.round)
	w.round = batchTally{}
	w.rounds++
}

// summary is what the batches of every round came to.
func (w *batchWatch) summary() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return fmt.Sprintf("translate: %s, %s", counted(w.rounds, "round", "rounds"), w.total)
}
