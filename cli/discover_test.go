package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/tamnd/bilibili-cli/bili"
)

// A gated edge is the ordinary case in a walk, so the notes have to survive as
// something countable rather than as prose printed once and forgotten.
func TestGatedEdgesAreCountedByState(t *testing.T) {
	var n walkNotes
	n.add(&bili.APIError{Status: bili.StatusRisk, Message: "no"})
	n.add(&bili.APIError{Status: bili.StatusRisk, Message: "no"})
	n.add(&bili.APIError{Status: bili.StatusRefusedSilent, Message: "nothing"})
	n.add(nil)

	if got := n.total(); got != 3 {
		t.Errorf("total = %d, want 3", got)
	}
	if got := n.summary(); got != "2 risk, 1 refused_silent" {
		t.Errorf("summary = %q", got)
	}
}

// A walk that reached anything past its seeds did what it was asked, however
// much of the graph was gated.
func TestAPartlyGatedWalkStillSucceeds(t *testing.T) {
	var n walkNotes
	n.add(&bili.APIError{Status: bili.StatusRisk, Message: "no"})
	if err := n.result(&App{quiet: true}, 12); err != nil {
		t.Errorf("a walk that reached 12 nodes failed: %v", err)
	}
}

// A walk where every edge was refused reached nothing, and exiting 0 there
// reports an empty graph as a complete one.
func TestAWalkThatReachedNothingExitsWithTheRefusal(t *testing.T) {
	var n walkNotes
	n.add(&bili.APIError{Status: bili.StatusRisk, Message: "no"})
	n.add(&bili.APIError{Status: bili.StatusRisk, Message: "no"})
	err := n.result(&App{quiet: true}, 0)
	if bili.Kind(err) != bili.StatusRisk {
		t.Fatalf("the refusal did not survive: %v", err)
	}
	if got := ExitCode(err); got != ExitRisk {
		t.Errorf("exit = %d, want %d", got, ExitRisk)
	}
}

// Refused in more than one way, so no single state describes the walk. Picking
// one of them would be a guess with an exit code attached to it.
func TestAWalkRefusedSeveralWaysIsNotAnyOneOfThem(t *testing.T) {
	var n walkNotes
	n.add(&bili.APIError{Status: bili.StatusRisk, Message: "no"})
	n.add(&bili.APIError{Status: bili.StatusNotFound, Message: "gone"})
	err := n.result(&App{quiet: true}, 0)
	if got := ExitCode(err); got != ExitUsage {
		t.Errorf("exit = %d, want %d", got, ExitUsage)
	}
	if !strings.Contains(err.Error(), "1 risk, 1 not_found") {
		t.Errorf("the counts are not in the message: %v", err)
	}
}

// A walk with nothing to report reports nothing.
func TestAWalkWithNoNotesIsSilent(t *testing.T) {
	var n walkNotes
	if err := n.result(&App{quiet: true}, 3); err != nil {
		t.Errorf("a clean walk failed: %v", err)
	}
	if err := n.result(&App{quiet: true}, 0); err != nil {
		t.Errorf("a walk of seeds only failed: %v", err)
	}
}

// The walk hands the error over, not its text, and this is what that buys: the
// classification is still there at the far end.
func TestAWalkNoteKeepsItsClassification(t *testing.T) {
	var got error
	opts := bili.WalkOptions{Note: func(err error) { got = err }}
	inner := &bili.APIError{Status: bili.StatusRefusedSilent, Message: "code 0 with no payload"}
	opts.Note(inner)
	if !errors.Is(got, inner) {
		t.Fatalf("the note arrived as %v", got)
	}
	if bili.Kind(got) != bili.StatusRefusedSilent {
		t.Errorf("kind = %s, want refused_silent", bili.Kind(got))
	}
}
