package bili

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// A refusal that names what to do next is a different experience from one that
// names a wall, and the folder listing is the case that earns it. What it says
// changed while this was being written: reading a folder by its id was the way
// around the refusal until the folder read started withholding its contents
// too, so the advice now says that rather than sending people down it.
func TestARefusedFolderListingSaysWhatToDoNext(t *testing.T) {
	const base = "https://api.bilibili.com/x/v3/fav/folder/created/list-all"
	st, _, err := classify(result{
		body:        capture(t, "refused_silent_fav_list_all.json"),
		status:      http.StatusOK,
		contentType: "application/json",
		base:        base,
	})
	if st != StatusRefusedSilent {
		t.Fatalf("the folder listing classified as %s", st)
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("no APIError for a refusal: %v", err)
	}
	if ae.Endpoint != base {
		t.Errorf("endpoint = %q, want the listing URL", ae.Endpoint)
	}
	for _, want := range []string{"used to be the way around this", "logged-in cookie"} {
		if !strings.Contains(ae.Hint, want) {
			t.Errorf("the hint does not say %q: %s", want, ae.Hint)
		}
	}
}

// The advice lives in the matrix so there is one table rather than two that can
// disagree, which is only true if the error actually reads from it.
func TestAdviceComesFromTheMatrix(t *testing.T) {
	// The hint an endpoint with nothing extra to say produces.
	plain := refusedSilentError("https://api.bilibili.com/x/nothing-in-the-matrix").Hint
	var checked int
	for _, r := range Matrix() {
		if r.Advice == "" {
			if h := refusedSilentError(r.Base).Hint; h != plain {
				t.Errorf("%s carries no advice but says more than the plain refusal: %s", r.Name, h)
			}
			continue
		}
		checked++
		if h := refusedSilentError(r.Base).Hint; !strings.Contains(h, r.Advice) {
			t.Errorf("%s: the matrix advice never reaches the error: %s", r.Name, h)
		}
	}
	if checked == 0 {
		t.Fatal("no row in the matrix carries advice, so this test proves nothing")
	}
}

// The default user agent is a browser string these endpoints accept, and
// replacing it is the most common self-inflicted -352. A caller staring at a
// risk refusal has no way to know the flag they set three shells ago is why.
func TestARiskRefusalNamesAReplacedUserAgent(t *testing.T) {
	const base = "https://api.bilibili.com/x/web-interface/ranking/v2"
	body := capture(t, "risk_352_ranking.json")
	const said = "--user-agent"

	_, _, err := classify(result{body: body, status: http.StatusOK, contentType: "application/json", base: base, uaOverridden: true})
	if err == nil || !strings.Contains(err.Error(), said) {
		t.Errorf("a -352 under a replaced UA does not mention it: %v", err)
	}

	_, _, err = classify(result{body: body, status: http.StatusOK, contentType: "application/json", base: base})
	if err == nil {
		t.Fatal("a -352 is a refusal")
	}
	if strings.Contains(err.Error(), said) {
		t.Errorf("a -352 under the default UA blames a flag nobody set: %v", err)
	}

	// The same refusal arrives as an HTML interstitial behind a 412, and the
	// advice is worth exactly as much there.
	err = riskError(result{base: base, uaOverridden: true})
	if !strings.Contains(err.Error(), said) {
		t.Errorf("the 412 form does not mention the user agent: %v", err)
	}
}

// Only a risk refusal gets the user agent sentence. A not found is not going to
// start existing because the header changed.
func TestOnlyRiskRefusalsBlameTheUserAgent(t *testing.T) {
	_, _, err := classify(result{
		body:         capture(t, "not_found_audio.json"),
		status:       http.StatusOK,
		contentType:  "application/json",
		base:         "https://www.bilibili.com/audio/music-service-c/web/song/info",
		uaOverridden: true,
	})
	if err == nil || strings.Contains(err.Error(), "--user-agent") {
		t.Errorf("a not found blamed the user agent: %v", err)
	}
}

// routedTransport answers by URL path, which is what a paginated endpoint needs:
// the client bootstraps buvid, fetches the WBI keys from nav, and only then asks
// for a page, and the three want different bodies.
type routedTransport struct {
	pages []string
	page  int
}

func (rt *routedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(r.URL.Path, "finger/spi"):
		return jsonResponse(r, 200, "application/json", []byte(`{"code":0,"data":{"b_3":"stub3","b_4":"stub4"}}`)), nil
	case strings.Contains(r.URL.Path, "web-interface/nav"):
		body := `{"code":0,"data":{"isLogin":false,"wbi_img":{"img_url":"https://i0.hdslb.com/bfs/wbi/aaaa.png","sub_url":"https://i0.hdslb.com/bfs/wbi/bbbb.png"}}}`
		return jsonResponse(r, 200, "application/json", []byte(body)), nil
	}
	body := rt.pages[min(rt.page, len(rt.pages)-1)]
	rt.page++
	return jsonResponse(r, 200, "application/json", []byte(body)), nil
}

func feedClient(t *testing.T, pages ...string) *Client {
	t.Helper()
	c := NewClient(Config{CacheDir: t.TempDir(), CacheTTL: time.Hour, Timeout: 5 * time.Second})
	c.hc.Transport = &routedTransport{pages: pages}
	return c
}

// drainFeed reads a dynamics feed to the end and returns what it produced.
func drainFeed(t *testing.T, c *Client) (int, error) {
	t.Helper()
	var n int
	for _, err := range c.Dynamics(context.Background(), "946974", ListOptions{}) {
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

const feedItem = `{"id_str":"1","type":"DYNAMIC_TYPE_AV","modules":{"module_author":{"mid":946974,"name":"a"}}}`

// The item count is not the signal for the end of a feed. Reading it that way
// makes a refused page and an exhausted feed the same event, and both come back
// as a short list that looks complete.
func TestAFeedThatPromisesMoreAndSendsNoneHasRefused(t *testing.T) {
	c := feedClient(t,
		`{"code":0,"data":{"has_more":true,"offset":"page2","items":[`+feedItem+`]}}`,
		`{"code":0,"data":{"has_more":true,"offset":"page3","items":[]}}`,
	)
	n, err := drainFeed(t, c)
	if n != 1 {
		t.Errorf("kept %d items, want the 1 the first page carried", n)
	}
	if Kind(err) != StatusRefusedSilent {
		t.Fatalf("a page that broke its own promise came back as %v", err)
	}
	if !strings.Contains(err.Error(), "said there was more") {
		t.Errorf("the refusal does not say what happened: %v", err)
	}
}

// The control. A feed that says it is done is done, and calling that a refusal
// would put an exit code on every complete read.
func TestAFeedThatSaysItIsDoneIsNotARefusal(t *testing.T) {
	c := feedClient(t,
		`{"code":0,"data":{"has_more":true,"offset":"page2","items":[`+feedItem+`]}}`,
		`{"code":0,"data":{"has_more":false,"offset":"","items":[]}}`,
	)
	n, err := drainFeed(t, c)
	if err != nil {
		t.Fatalf("the end of a feed is not an error: %v", err)
	}
	if n != 1 {
		t.Errorf("kept %d items, want 1", n)
	}
}

// A first page carrying nothing has not paginated anywhere. This endpoint
// refuses anonymous callers far more often than a creator posts nothing at all,
// so the empty first page is reported rather than returned as an empty feed.
func TestAnEmptyFirstPageIsARefusalAndNotAnEmptyFeed(t *testing.T) {
	c := feedClient(t, `{"code":0,"data":{"has_more":false,"offset":"","items":[]}}`)
	n, err := drainFeed(t, c)
	if n != 0 {
		t.Errorf("emitted %d items from an empty page", n)
	}
	if Kind(err) != StatusRefusedSilent {
		t.Fatalf("an empty first page came back as %v", err)
	}
	if !strings.Contains(err.Error(), "cookie") {
		t.Errorf("the refusal does not say what would change it: %v", err)
	}
}

// The folder read is the same defect one level down: the endpoint answers with
// the folder's metadata and withholds its contents, so the payload rule sees a
// payload and the caller sees an empty folder. media_count is what settles it.
func TestAWithheldFolderIsNotAnEmptyFolder(t *testing.T) {
	for _, tc := range []struct {
		name       string
		page       int
		emitted    int
		hasMore    bool
		mediaCount int
		refused    bool
	}{
		{"a folder holding items that sent none", 1, 0, true, 145, true},
		{"the same without the has_more promise", 1, 0, false, 145, true},
		{"a folder that is genuinely empty", 1, 0, false, 0, false},
		{"paged to the end", 3, 40, false, 40, false},
		{"a page past the end the caller asked for", 9, 0, false, 40, false},
		{"a promise broken part way through", 2, 20, true, 145, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := emptyFolderPage(tc.page, tc.emitted, tc.hasMore, tc.mediaCount)
			if tc.refused && err == nil {
				t.Fatal("a withheld folder came back as an empty one")
			}
			if !tc.refused && err != nil {
				t.Fatalf("an honest empty page came back as a refusal: %v", err)
			}
			if err != nil && err.Status != StatusRefusedSilent {
				t.Errorf("status = %s, want refused_silent", err.Status)
			}
		})
	}
}

// The count is in the message because it is the evidence. A reader who is told
// the folder holds 145 items and sent none does not have to take the tool's
// word for the classification.
func TestAWithheldFolderSaysHowManyItemsItHolds(t *testing.T) {
	err := emptyFolderPage(1, 0, false, 145)
	if err == nil {
		t.Fatal("a folder holding 145 items sent none and that passed")
	}
	if !strings.Contains(err.Error(), "145") {
		t.Errorf("the message does not carry the count: %v", err)
	}
	if err.Endpoint != favItemsBase {
		t.Errorf("endpoint = %q, want the folder contents URL", err.Endpoint)
	}
}
