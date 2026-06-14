package cli

import (
	"bytes"
	"strings"
	"testing"
)

type sample struct {
	BVID  string `json:"bvid"`
	Title string `json:"title"`
	Views int64  `json:"view_count"`
	URL   string `json:"url"`
}

func renderOut(t *testing.T, format Format, fields []string, recs ...any) string {
	t.Helper()
	var buf bytes.Buffer
	o, err := NewOutput(&buf, format, fields, false, "", false, false, 0)
	if err != nil {
		t.Fatalf("NewOutput: %v", err)
	}
	for _, r := range recs {
		if err := o.Emit(r); err != nil {
			t.Fatalf("Emit: %v", err)
		}
	}
	if err := o.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.String()
}

func TestJSONLOneRecordPerLine(t *testing.T) {
	out := renderOut(t, FormatJSONL, nil,
		sample{BVID: "BV1", Title: "a", Views: 1},
		sample{BVID: "BV2", Title: "b", Views: 2},
	)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], `"bvid":"BV1"`) {
		t.Errorf("line 0 = %q", lines[0])
	}
}

func TestJSONArray(t *testing.T) {
	out := renderOut(t, FormatJSON, nil, sample{BVID: "BV1"})
	if !strings.HasPrefix(strings.TrimSpace(out), "[") || !strings.HasSuffix(strings.TrimSpace(out), "]") {
		t.Fatalf("json output is not an array: %q", out)
	}
}

func TestJSONEmptyIsEmptyArray(t *testing.T) {
	out := strings.TrimSpace(renderOut(t, FormatJSON, nil))
	if out != "[]" {
		t.Fatalf("empty json = %q, want []", out)
	}
}

func TestCSVHeaderAndFields(t *testing.T) {
	out := renderOut(t, FormatCSV, []string{"bvid", "title"}, sample{BVID: "BV1", Title: "hi", Views: 9})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if lines[0] != "bvid,title" {
		t.Fatalf("header = %q, want bvid,title", lines[0])
	}
	if lines[1] != "BV1,hi" {
		t.Fatalf("row = %q, want BV1,hi", lines[1])
	}
}

func TestURLFormatPicksURLField(t *testing.T) {
	out := strings.TrimSpace(renderOut(t, FormatURL, nil, sample{BVID: "BV1", URL: "https://x/BV1"}))
	if out != "https://x/BV1" {
		t.Fatalf("url output = %q, want the url field", out)
	}
}

func TestTSVUsesTabs(t *testing.T) {
	out := renderOut(t, FormatTSV, []string{"bvid", "title"}, sample{BVID: "BV1", Title: "hi"})
	if !strings.Contains(out, "BV1\thi") {
		t.Fatalf("tsv row not tab-separated: %q", out)
	}
}
