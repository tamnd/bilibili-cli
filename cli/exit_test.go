package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/tamnd/bilibili-cli/bili"
)

// The table in the README is the contract. This is it in code, and the two are
// meant to be read side by side.
func TestExitCodeForEveryState(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, ExitOK},
		{ErrNoResults, ExitEmpty},
		{&bili.APIError{Status: bili.StatusRisk}, ExitRisk},
		{&bili.APIError{Status: bili.StatusForbidden}, ExitRisk},
		{&bili.APIError{Status: bili.StatusEmpty}, ExitEmpty},
		{&bili.APIError{Status: bili.StatusRefusedSilent}, ExitRefused},
		{&bili.APIError{Status: bili.StatusNetwork}, ExitNetwork},
		{&bili.APIError{Status: bili.StatusRate}, ExitRate},
		{&bili.APIError{Status: bili.StatusNotFound}, ExitNotFound},
		{&bili.APIError{Status: bili.StatusError}, ExitUsage},
		{errors.New("unknown flag: --nope"), ExitUsage},
		{context.DeadlineExceeded, ExitNetwork},
	}
	for _, tc := range cases {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}

// Every failure leaves through the wrapper that puts the request in front of
// it, so an exit code read from an unwrapped error would be read from an error
// that never reaches main.
func TestExitCodeSurvivesTheWrapping(t *testing.T) {
	inner := &bili.APIError{Status: bili.StatusRefusedSilent, Message: "code 0 with no payload"}
	wrapped := &targetError{cmd: "favorites", args: []string{"946974"}, err: inner}
	if got := ExitCode(wrapped); got != ExitRefused {
		t.Errorf("ExitCode(wrapped) = %d, want %d", got, ExitRefused)
	}
	if got := ExitCode(fmt.Errorf("outer: %w", wrapped)); got != ExitRefused {
		t.Errorf("ExitCode(twice wrapped) = %d, want %d", got, ExitRefused)
	}
}

// Three things: what was asked, which endpoint answered, and what it said.
func TestAFailureNamesTheRequest(t *testing.T) {
	inner := &bili.APIError{
		Status:   bili.StatusRefusedSilent,
		Message:  "code 0 with no payload",
		Hint:     "this endpoint refuses anonymous callers",
		Endpoint: "https://api.bilibili.com/x/v3/fav/folder/created/list-all",
	}
	msg := (&targetError{cmd: "favorites", args: []string{"946974"}, err: inner}).Error()
	for _, want := range []string{"favorites 946974", "x/v3/fav/folder/created/list-all", "refused"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q does not say %q", msg, want)
		}
	}
}

// A run fed five hundred ids on stdin has a failure worth reading and a prefix
// nobody wants.
func TestALongArgumentListIsSummarized(t *testing.T) {
	got := summarizeArgs([]string{"BV1", "BV2", "BV3", "BV4", "BV5"})
	if got != "BV1 BV2 BV3 and 2 more" {
		t.Errorf("summarizeArgs = %q", got)
	}
	if got := summarizeArgs([]string{"BV1", "BV2"}); got != "BV1 BV2" {
		t.Errorf("summarizeArgs = %q, want the args verbatim", got)
	}
}

func TestOneFailureInManyIsNotAFailedRun(t *testing.T) {
	a := &App{quiet: true}
	ids := []string{"BV1", "BV2", "BV3"}
	got, err := runEach(a, ids, func(id string) ([]string, error) {
		if id == "BV2" {
			return nil, &bili.APIError{Status: bili.StatusRefusedSilent, Message: "no"}
		}
		return []string{id}, nil
	})
	if err != nil {
		t.Fatalf("a run that partly worked should not fail: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("kept %d records, want the 2 that worked", len(got))
	}
}

// A status becomes the run's exit code only when it covers every target.
func TestAStatusOnlyBecomesTheRunWhenItCoversEveryTarget(t *testing.T) {
	a := &App{quiet: true}
	refused := func(string) ([]string, error) {
		return nil, &bili.APIError{Status: bili.StatusRefusedSilent, Message: "no"}
	}
	_, err := runEach(a, []string{"a", "b"}, refused)
	if got := ExitCode(err); got != ExitRefused {
		t.Errorf("every target refused, exit = %d, want %d", got, ExitRefused)
	}

	var n int
	mixed := func(string) ([]string, error) {
		n++
		if n == 1 {
			return nil, &bili.APIError{Status: bili.StatusRefusedSilent, Message: "no"}
		}
		return nil, &bili.APIError{Status: bili.StatusNotFound, Message: "gone"}
	}
	_, err = runEach(a, []string{"a", "b"}, mixed)
	if got := ExitCode(err); got != ExitUsage {
		t.Errorf("targets failed differently, exit = %d, want %d", got, ExitUsage)
	}
}

func TestARunThatWroteNothingIsEmptyAndNotAnError(t *testing.T) {
	var buf bytes.Buffer
	out, err := NewOutput(&buf, FormatJSONL, nil, false, "", false, false, 0)
	if err != nil {
		t.Fatalf("NewOutput: %v", err)
	}
	if got := noResults(&App{}, out); !errors.Is(got, ErrNoResults) {
		t.Errorf("noResults on an empty run = %v, want ErrNoResults", got)
	}
	if got := ExitCode(noResults(&App{}, out)); got != ExitEmpty {
		t.Errorf("exit = %d, want %d", got, ExitEmpty)
	}
	if err := out.Emit(sample{BVID: "BV1"}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if got := noResults(&App{}, out); got != nil {
		t.Errorf("noResults after a record = %v, want nil", got)
	}
}

// A dry run writes nothing by design, and reporting that as an empty result
// would say something about the site when nothing was asked of it.
func TestADryRunIsNotAnEmptyResult(t *testing.T) {
	var buf bytes.Buffer
	out, err := NewOutput(&buf, FormatJSONL, nil, false, "", false, false, 0)
	if err != nil {
		t.Fatalf("NewOutput: %v", err)
	}
	if got := noResults(&App{dryRun: true}, out); got != nil {
		t.Errorf("noResults in a dry run = %v, want nil", got)
	}
}
