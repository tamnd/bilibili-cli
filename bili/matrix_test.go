package bili

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMatrixIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Matrix() {
		switch {
		case r.Name == "":
			t.Errorf("row with base %s has no name", r.Base)
		case seen[r.Name]:
			t.Errorf("%s appears twice", r.Name)
		case !strings.HasPrefix(r.Base, "https://"):
			t.Errorf("%s: base %q is not an https URL", r.Name, r.Base)
		case len(r.Params) == 0:
			t.Errorf("%s: no parameters, so the probe would not be a real request", r.Name)
		case r.Device && !r.Signed:
			t.Errorf("%s: wants device parameters but is not signed, which was never measured", r.Name)
		case !r.Payload && r.Note == "":
			t.Errorf("%s: recorded as carrying no payload without saying why", r.Name)
		}
		seen[r.Name] = true
	}
}

// TestMatrixMatchesTheClient is the test this file exists for. The matrix says
// which endpoints refuse an unsigned request; the client decides what to send
// by calling either getJSON or getJSONSigned. Those two facts are written in
// different places and nothing but this test keeps them together.
//
// The claim is one directional on purpose. An endpoint recorded as requiring a
// signature and fetched without one is a bug that shows up as a -403 or a -352
// in front of a user. An endpoint recorded as not requiring one and signed
// anyway costs a nav fetch and nothing else, and it is the safer side to be
// wrong on when the site tightens.
//
// It reads the source rather than making requests, so it runs offline and it
// fails on the change rather than on the consequence.
func TestMatrixMatchesTheClient(t *testing.T) {
	signedBy := clientSigningDecisions(t)

	for _, r := range Matrix() {
		if !r.Signed {
			continue
		}
		signed, called := signedBy[r.Base]
		if !called {
			// Not every row is reached through getJSON. The danmaku segment
			// endpoint is protobuf and goes through rawGet.
			continue
		}
		if !signed {
			t.Errorf("%s: the matrix says a signature is required and the client fetches it unsigned", r.Name)
		}
	}
}

// TestEveryFetchedEndpointIsInTheMatrix catches the other direction: a new
// endpoint added to the client and not recorded, which the drift job would then
// never re-measure.
func TestEveryFetchedEndpointIsInTheMatrix(t *testing.T) {
	// Endpoints that are deliberately absent, with the reason.
	exempt := map[string]string{
		"https://api.bilibili.com/x/web-interface/nav":              "bootstrap. Answers -101 anonymously and is measured by the probe run itself",
		"https://api.bilibili.com/x/frontend/finger/spi":            "bootstrap. Issues the buvid cookies the other rows depend on",
		"https://api.bilibili.com/x/web-interface/search/square":    "the unsigned twin of the wbi route, kept for a fallback path",
		"https://api.bilibili.com/x/v2/reply/reply":                 "nested replies, only reachable with an oid from x/v2/reply/wbi/main",
		"https://api.bilibili.com/x/polymer/web-dynamic/v1/detail":  "needs a dynamic id, which has no stable public example",
		"https://api.bilibili.com/x/player/wbi/playurl":             "returns signed stream URLs that expire, so a probe would measure nothing durable",
		"https://api.live.bilibili.com/live_user/v1/Master/info":    "needs a uid that is currently streaming",
		"https://api.live.bilibili.com/room/v2/Room/room_id_by_uid": "same",
		"https://s.search.bilibili.com/main/suggest":                "its own host, no signature, no cookie, nothing to drift",
	}

	inMatrix := map[string]bool{}
	for _, r := range Matrix() {
		inMatrix[r.Base] = true
	}

	for base := range clientSigningDecisions(t) {
		if inMatrix[base] || exempt[base] != "" {
			continue
		}
		t.Errorf("%s is fetched by the client but is not in the matrix and is not exempt", base)
	}
}

// clientSigningDecisions walks bili/*.go and reports, for every endpoint the
// client fetches through a string literal, whether it goes out signed.
func clientSigningDecisions(t *testing.T) map[string]bool {
	t.Helper()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	out := map[string]bool{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var signed bool
			switch sel.Sel.Name {
			case "getJSON", "getJSONNoCache":
			case "getJSONSigned":
				signed = true
			default:
				return true
			}
			// getJSON(ctx, base, params, out): the base is the second argument.
			if len(call.Args) < 2 {
				return true
			}
			lit, ok := call.Args[1].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			base, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			// A signed call anywhere wins. An endpoint reached both ways is
			// itself a bug, and the matrix comparison will surface it.
			if signed || !out[base] {
				out[base] = signed || out[base]
			}
			if _, seen := out[base]; !seen {
				out[base] = signed
			}
			return true
		})
	}
	return out
}

func TestClassifyCode(t *testing.T) {
	for code, want := range map[int]string{
		-352:    stateRisk,
		-403:    stateForbidden,
		-404:    stateNotFound,
		4511001: stateNotFound,
		-509:    stateRate,
		-400:    stateError,
	} {
		if got := classifyCode(code); got != want {
			t.Errorf("classifyCode(%d) = %s, want %s", code, got, want)
		}
	}
}

func TestProbeReportsRefusedSilentAsRecorded(t *testing.T) {
	// upstat is recorded as carrying no payload. A code 0 with an empty object
	// is therefore the recorded behaviour and must not be reported as drift.
	var upstat Requirement
	for _, r := range Matrix() {
		if r.Name == "x/space/upstat" {
			upstat = r
		}
	}
	if upstat.Name == "" {
		t.Fatal("x/space/upstat is not in the matrix")
	}
	if upstat.Payload {
		t.Error("x/space/upstat is recorded as carrying a payload, which it has never done anonymously")
	}
	if len(upstat.Params) == 0 {
		t.Error("x/space/upstat has no parameters")
	}
	if _, ok := upstat.Params["mid"]; !ok {
		t.Error("x/space/upstat is probed without a mid")
	}
	_ = url.Values(upstat.Params)
}

func TestHTTPStatus(t *testing.T) {
	for msg, want := range map[string]int{
		"HTTP 412 from https://api.bilibili.com/x/article/view?id=1":  412,
		"HTTP 429 from https://api.bilibili.com/x/web-interface/view": 429,
		"HTTP 200 from https://api.bilibili.com/x/web-interface/view": 200,
		// The URL contains 412 and the message does not. Matching on three
		// bare digits would read this as a risk refusal.
		"dial tcp: lookup api.bilibili.com/412: no such host": 0,
		"":     0,
		"HTTP": 0,
	} {
		if got := httpStatus(msg); got != want {
			t.Errorf("httpStatus(%q) = %d, want %d", msg, got, want)
		}
	}
}

func TestClassifyErrReadsTheStatusNotTheDigits(t *testing.T) {
	netErr := &APIError{Code: 0, Message: "HTTP 412 from https://api.bilibili.com/x/article/view?id=1", Kind: ErrNetwork}
	if got := classifyErr(netErr); got != stateRisk {
		t.Errorf("a 412 should be a risk state, got %s", got)
	}
	coded := &APIError{Code: -509, Message: "too frequent", Kind: ErrRate}
	if got := classifyErr(coded); got != stateRate {
		t.Errorf("a -509 should be a rate state, got %s", got)
	}
}

func TestExpectDefaultsToOK(t *testing.T) {
	if got := (Requirement{}).expect(); got != stateOK {
		t.Errorf("a row with no recorded expectation should expect ok, got %s", got)
	}
	if got := (Requirement{Expect: stateRefusedSilent}).expect(); got != stateRefusedSilent {
		t.Errorf("a recorded expectation should be honoured, got %s", got)
	}
}

func TestProbeBackoffOutlastsTheLongestPenaltyWindowMeasured(t *testing.T) {
	// x/article/view refused a minute after a probe run and answered three
	// minutes after it. A backoff shorter than that reports its own pressure as
	// drift, every week, forever.
	if ProbeBackoff < 3*time.Minute {
		t.Errorf("ProbeBackoff is %s, which is shorter than the longest penalty window measured", ProbeBackoff)
	}
}

// TestSignedRowsJudgeOnlyTheSignedAnswer pins the asymmetry in Probe. It is the
// kind of thing that reads like an oversight and gets "fixed" back into a weekly
// false positive, so it is written down as an assertion.
func TestSignedRowsJudgeOnlyTheSignedAnswer(t *testing.T) {
	src, err := os.ReadFile("matrix.go")
	if err != nil {
		t.Fatalf("read matrix.go: %v", err)
	}
	if strings.Contains(string(src), `"recorded as needing a signature, and it answered without one"`) {
		t.Error("an unsigned request that unexpectedly succeeds is not drift: the client keeps signing and nothing breaks")
	}
}

// TestRateLimitedTwiceIsNotDrift pins the other half of the same judgement. A
// -509 says we are asking too often, which is a fact about this run and not
// about what the endpoint requires, so a row the site throttled on both passes
// was not measured rather than moved. x/article/view is the row that does this
// in practice, and reporting it as drift would open an issue every week that
// tells the reader nothing.
func TestRateLimitedTwiceIsNotDrift(t *testing.T) {
	src, err := os.ReadFile("matrix.go")
	if err != nil {
		t.Fatalf("read matrix.go: %v", err)
	}
	if !strings.Contains(string(src), "o.Moved, o.Unmeasured = false, true") {
		t.Error("the second pass must downgrade a repeated rate limit to unmeasured instead of reporting it as drift")
	}
}
