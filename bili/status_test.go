package bili

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// capture loads a stored response from testdata.
func capture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read capture %s: %v", name, err)
	}
	return b
}

// TestClassifyEveryStoredCapture runs the classifier over one real response per
// state. The captures are what bilibili actually sent, which is the only useful
// thing to test a classifier against.
func TestClassifyEveryStoredCapture(t *testing.T) {
	cases := []struct {
		file   string
		status int
		ctype  string
		base   string
		want   Status
	}{
		{"ok_relation_stat.json", 200, "application/json", "https://api.bilibili.com/x/relation/stat", StatusOK},
		{"ok_fav_page_past_end.json", 200, "application/json", "https://api.bilibili.com/x/v3/fav/resource/list", StatusOK},
		{"empty_tags.json", 200, "application/json", "https://api.bilibili.com/x/tag/archive/tags", StatusEmpty},
		{"refused_silent_upstat.json", 200, "application/json", "https://api.bilibili.com/x/space/upstat", StatusRefusedSilent},
		{"refused_silent_fav_list_all.json", 200, "application/json", "https://api.bilibili.com/x/v3/fav/folder/created/list-all", StatusRefusedSilent},
		{"risk_352_ranking.json", 200, "application/json", "https://api.bilibili.com/x/web-interface/ranking/v2", StatusRisk},
		{"risk_412.html", 412, "text/html", "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space", StatusRisk},
		{"forbidden_reply.json", 200, "application/json", "https://api.bilibili.com/x/v2/reply/wbi/main", StatusForbidden},
		{"not_found_audio.json", 200, "application/json", "https://www.bilibili.com/audio/music-service-c/web/song/info", StatusNotFound},
		{"rate_509.json", 200, "application/json", "https://api.bilibili.com/x/article/view", StatusRate},
	}

	seen := map[Status]bool{}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			got, _, apiErr := classify(result{
				body:        capture(t, tc.file),
				status:      tc.status,
				contentType: tc.ctype,
				base:        tc.base,
			})
			if got != tc.want {
				t.Fatalf("classified as %s, want %s", got, tc.want)
			}
			if got.Refused() != (apiErr != nil) {
				t.Fatalf("%s: Refused() is %v but the error is %v", got, got.Refused(), apiErr)
			}
			if apiErr != nil && apiErr.Endpoint != tc.base {
				t.Errorf("error does not name the endpoint that refused: %q", apiErr.Endpoint)
			}
		})
		seen[tc.want] = true
	}

	// Every state has a capture behind it. The one that matters is refused_silent,
	// which is not in anybody's documentation and would otherwise be a claim
	// rather than a measurement.
	for _, st := range []Status{StatusOK, StatusEmpty, StatusRefusedSilent, StatusRisk, StatusForbidden, StatusNotFound, StatusRate} {
		if !seen[st] {
			t.Errorf("no stored capture for %s", st)
		}
	}
}

// TestTheRiskInterstitialIsNeverParsedAsJSON is the specific failure the
// ordering in classify exists to prevent. The body is a real HTML page and a
// JSON decoder meeting it produces a parse error, which sends the reader
// looking for a bug in the decoder instead of at a refusal.
func TestTheRiskInterstitialIsNeverParsedAsJSON(t *testing.T) {
	body := capture(t, "risk_412.html")
	if len(body) != 3400 {
		t.Fatalf("the stored interstitial is %d bytes, expected the 3400 byte capture", len(body))
	}

	var out map[string]any
	st, err := decodeResult(result{
		body:        body,
		status:      412,
		contentType: "text/html; charset=utf-8",
		base:        "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space",
	}, &out)

	if st != StatusRisk {
		t.Fatalf("classified as %s, want %s", st, StatusRisk)
	}
	if err == nil {
		t.Fatal("no error for an intercepted response")
	}
	if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "decode") {
		t.Errorf("the error is about parsing rather than about the interception: %v", err)
	}
	if out != nil {
		t.Error("the payload was decoded into out despite the response being a refusal")
	}
}

// TestPayloadRule is the per endpoint table the classifier's last step reads.
// It is a test rather than a comment on purpose. The day upstat starts
// answering, this fails and somebody gets to delete a row, which is the correct
// way for a workaround to end.
func TestPayloadRule(t *testing.T) {
	cases := []struct {
		base string
		want bool
	}{
		{"https://api.bilibili.com/x/space/upstat", true},
		{"https://api.bilibili.com/x/v3/fav/folder/created/list-all", true},
		{"https://api.bilibili.com/x/relation/stat", true},
		{"https://api.bilibili.com/x/web-interface/view", true},
		{"https://api.bilibili.com/x/v2/dm/web/seg.so", true},
		// Not in the matrix, so it has no rule and can never be called a
		// silent refusal.
		{"https://api.bilibili.com/x/web-interface/nav", false},
		{"https://example.invalid/nothing", false},
	}
	for _, tc := range cases {
		if got := carriesPayload(tc.base); got != tc.want {
			t.Errorf("carriesPayload(%s) = %v, want %v", tc.base, got, tc.want)
		}
	}

	// Every endpoint the rule says yes to has a reason on the row, because the
	// rule is what turns an empty response into a refusal in front of a user
	// and that is not a thing to assert without saying why.
	for _, r := range Matrix() {
		if r.Payload && r.expect().Refused() && r.Note == "" {
			t.Errorf("%s is recorded as carrying a payload and as not answering, without saying why", r.Name)
		}
	}
}

// TestEmptyPayloadShapes pins which payloads count as nothing.
func TestEmptyPayloadShapes(t *testing.T) {
	for body, want := range map[string]bool{
		"null":       true,
		"{}":         true,
		"  {}  ":     true,
		"":           true,
		"[]":         false, // an answer with no entries, not a refusal
		`{"a":1}`:    false,
		`[{"a":1}]`:  false,
		`{"a":null}`: false,
	} {
		if got := isEmptyPayload([]byte(body)); got != want {
			t.Errorf("isEmptyPayload(%q) = %v, want %v", body, got, want)
		}
	}
}

// A dry run prints the requests and invents the responses. The payload rule
// would call every one of them a silent refusal, since the body it invents
// carries no payload by design, and the user would be told an endpoint refused
// a request that was never sent.
func TestADryRunIsNeverClassifiedAsARefusal(t *testing.T) {
	base := "https://api.bilibili.com/x/v3/fav/folder/created/list-all"
	if !carriesPayload(base) {
		t.Fatalf("%s should carry a payload, so this test is testing the wrong endpoint", base)
	}
	res := result{body: dryRunBody, status: 200, contentType: "application/json", base: base, dryRun: true}
	st, _, apiErr := classify(res)
	if st != StatusEmpty {
		t.Errorf("dry run classified as %q, want %q", st, StatusEmpty)
	}
	if apiErr != nil {
		t.Errorf("dry run produced an error: %v", apiErr)
	}
}

func TestRefusedAndCacheableAgree(t *testing.T) {
	for st, refused := range map[Status]bool{
		StatusOK:            false,
		StatusEmpty:         false,
		StatusRefusedSilent: true,
		StatusRisk:          true,
		StatusForbidden:     true,
		StatusNotFound:      true,
		StatusRate:          true,
		StatusNetwork:       true,
		StatusError:         true,
	} {
		if st.Refused() != refused {
			t.Errorf("%s.Refused() = %v, want %v", st, st.Refused(), refused)
		}
		if st.Cacheable() == refused {
			t.Errorf("%s is both %v refused and %v cacheable", st, st.Refused(), st.Cacheable())
		}
	}
}

// stubTransport serves one canned response to every request except the buvid
// bootstrap, which every client makes before its first real call.
type stubTransport struct {
	status int
	ctype  string
	body   []byte
	hits   int
}

func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Path, "finger/spi") {
		return jsonResponse(r, 200, "application/json", []byte(`{"code":0,"data":{"b_3":"stub3","b_4":"stub4"}}`)), nil
	}
	s.hits++
	return jsonResponse(r, s.status, s.ctype, s.body), nil
}

func jsonResponse(r *http.Request, status int, ctype string, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{ctype}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    r,
	}
}

func stubClient(t *testing.T, st *stubTransport) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	c := NewClient(Config{CacheDir: dir, CacheTTL: time.Hour, Retries: 3, Timeout: 5 * time.Second})
	c.hc.Transport = st
	return c, dir
}

func cachedFiles(t *testing.T, dir string) int {
	t.Helper()
	n, _ := CacheStats(dir)
	return n
}

// TestRefusalsAreNeverCached is the rule that keeps a five minute problem from
// becoming an hour long one. Every refusal on this API is a statement about
// this request from this address right now, and a cached -352 makes the tool
// look broken long after the site has stopped objecting.
func TestRefusalsAreNeverCached(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
		base string
	}{
		{"risk", "risk_352_ranking.json", "https://api.bilibili.com/x/web-interface/ranking/v2"},
		{"refused_silent", "refused_silent_fav_list_all.json", "https://api.bilibili.com/x/v3/fav/folder/created/list-all"},
		{"forbidden", "forbidden_reply.json", "https://api.bilibili.com/x/v2/reply/wbi/main"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, dir := stubClient(t, &stubTransport{status: 200, ctype: "application/json", body: capture(t, tc.file)})
			var out map[string]any
			if err := c.getJSON(context.Background(), tc.base, nil, &out); err == nil {
				t.Fatal("a refusal came back as success")
			}
			if n := cachedFiles(t, dir); n != 0 {
				t.Errorf("%d files written to the cache for a refusal", n)
			}
		})
	}
}

// TestAnswersAreCached is the control. A rule that never caches anything would
// also pass the test above.
func TestAnswersAreCached(t *testing.T) {
	base := "https://api.bilibili.com/x/relation/stat"
	stub := &stubTransport{status: 200, ctype: "application/json", body: capture(t, "ok_relation_stat.json")}
	c, dir := stubClient(t, stub)

	var out map[string]any
	if err := c.getJSON(context.Background(), base, nil, &out); err != nil {
		t.Fatalf("getJSON: %v", err)
	}
	if cachedFiles(t, dir) != 1 {
		t.Fatalf("%d cached files after one answer, want 1", cachedFiles(t, dir))
	}
	if out["mid"] == nil {
		t.Error("the payload was not decoded")
	}

	before := stub.hits
	var again map[string]any
	if err := c.getJSON(context.Background(), base, nil, &again); err != nil {
		t.Fatalf("second getJSON: %v", err)
	}
	if stub.hits != before {
		t.Error("the second call went to the network instead of reading the cache")
	}
}

// TestAStaleCachedRefusalIsIgnored covers the cache written by a version that
// did not know a refusal from an answer. Replaying it for the rest of the TTL
// would be the worst of both.
func TestAStaleCachedRefusalIsIgnored(t *testing.T) {
	base := "https://api.bilibili.com/x/relation/stat"
	stub := &stubTransport{status: 200, ctype: "application/json", body: capture(t, "ok_relation_stat.json")}
	c, dir := stubClient(t, stub)

	// Plant a refusal under the key this request will use.
	newCache(dir, time.Hour).put(buildURL(base, nil), capture(t, "risk_352_ranking.json"))

	var out map[string]any
	if err := c.getJSON(context.Background(), base, nil, &out); err != nil {
		t.Fatalf("getJSON: %v", err)
	}
	if stub.hits == 0 {
		t.Error("the planted refusal was replayed instead of being re-asked")
	}
	if out["mid"] == nil {
		t.Error("the payload was not decoded")
	}
}

// TestA412IsNotRetried is the difference between a refusal and a busy server.
// Risk control saying no to this address is not improved by asking it four more
// times in the next second.
func TestA412IsNotRetried(t *testing.T) {
	stub := &stubTransport{status: 412, ctype: "text/html", body: capture(t, "risk_412.html")}
	c, _ := stubClient(t, stub)

	var out map[string]any
	err := c.getJSON(context.Background(), "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space", nil, &out)
	if err == nil {
		t.Fatal("an intercepted request came back as success")
	}
	var ae *APIError
	if !errors.As(err, &ae) || ae.Status != StatusRisk {
		t.Fatalf("error is %v, want a risk APIError", err)
	}
	if stub.hits != 1 {
		t.Errorf("the 412 was requested %d times, want 1", stub.hits)
	}
}

// TestA429IsRetried is the other side of the same decision. A server that said
// it was busy is worth asking again.
func TestA429IsRetried(t *testing.T) {
	stub := &stubTransport{status: 429, ctype: "application/json", body: []byte(`{}`)}
	c, _ := stubClient(t, stub)
	c.cfg.Rate = time.Millisecond

	var out map[string]any
	if err := c.getJSON(context.Background(), "https://api.bilibili.com/x/relation/stat", nil, &out); err == nil {
		t.Fatal("a 429 came back as success")
	}
	if stub.hits < 2 {
		t.Errorf("the 429 was requested %d times, want it retried", stub.hits)
	}
}
