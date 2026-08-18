package cli

import (
	"encoding/csv"
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/bilibili-cli/bili"
)

func i64(v int64) *int64 { return &v }

// Every record carries its provenance and no record spends a column on it. The
// two are only compatible because the envelope is tagged out of the table and
// the csv while staying in the json, and that is worth a test rather than a
// comment: a tag that stops working fails silently and widely.
func TestTheEnvelopeIsNotAColumn(t *testing.T) {
	rec := bili.User{
		Mid: 946974, Name: "a creator", FollowerCount: i64(17),
		Envelope: &bili.Envelope{Endpoint: "x/space/wbi/acc/info", Status: bili.StatusOK},
	}
	header := strings.Split(strings.Split(strings.TrimSpace(renderOut(t, FormatCSV, nil, rec)), "\n")[0], ",")
	if slices.Contains(header, "envelope") {
		t.Errorf("the envelope took a csv column: %v", header)
	}
	if !slices.Contains(header, "mid") {
		t.Fatalf("the header looks wrong, so this test is not proving anything: %v", header)
	}

	// Hidden by default is not the same as gone. Somebody who wants to see
	// where a record came from asks for it by name.
	asked := strings.TrimSpace(renderOut(t, FormatCSV, []string{"mid", "envelope"}, rec))
	if !strings.Contains(asked, "acc/info") {
		t.Errorf("--fields envelope did not reach the envelope: %s", asked)
	}

	if out := renderOut(t, FormatJSONL, nil, rec); !strings.Contains(out, `"endpoint":"x/space/wbi/acc/info"`) {
		t.Errorf("the json lost the envelope: %s", out)
	}
}

// A count nobody was told prints as an empty cell, not as a zero. In a csv this
// is the whole point: a zero in a column of counts is arithmetic to whatever
// reads it next, and an empty field is not.
func TestAWithheldCountIsAnEmptyCellAndNotAZero(t *testing.T) {
	rec := bili.User{
		Mid: 946974, Name: "a creator", FollowerCount: i64(17), TotalView: nil,
		Envelope: &bili.Envelope{
			Endpoint: "x/space/wbi/acc/info",
			Status:   bili.StatusOK,
			Missed:   map[string]string{"total_view": "x/space/upstat refused_silent: code 0 with no payload"},
		},
	}
	out := renderOut(t, FormatCSV, []string{"follower_count", "total_view", "total_like"}, rec)
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("the csv does not parse: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want a header and one record: %q", len(rows), out)
	}
	if rows[1][0] != "17" {
		t.Errorf("follower_count rendered as %q, want 17", rows[1][0])
	}
	for i, name := range []string{"total_view", "total_like"} {
		if got := rows[1][i+1]; got != "" {
			t.Errorf("%s rendered as %q, want an empty cell", name, got)
		}
	}
}
