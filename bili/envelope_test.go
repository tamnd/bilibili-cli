package bili

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

// pathRouter answers by the tail of the request path, which is what a record
// assembled from four endpoints needs: acc/info has to answer while upstat
// refuses, and a sequential stub cannot express that.
type pathRouter map[string]string

func (p pathRouter) RoundTrip(r *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(r.URL.Path, "finger/spi"):
		return jsonResponse(r, 200, "application/json", []byte(`{"code":0,"data":{"b_3":"stub3","b_4":"stub4"}}`)), nil
	case strings.Contains(r.URL.Path, "web-interface/nav"):
		body := `{"code":0,"data":{"isLogin":false,"wbi_img":{"img_url":"https://i0.hdslb.com/bfs/wbi/aaaa.png","sub_url":"https://i0.hdslb.com/bfs/wbi/bbbb.png"}}}`
		return jsonResponse(r, 200, "application/json", []byte(body)), nil
	}
	for suffix, body := range p {
		if strings.HasSuffix(r.URL.Path, suffix) {
			return jsonResponse(r, 200, "application/json", []byte(body)), nil
		}
	}
	return jsonResponse(r, 404, "application/json", []byte(`{"code":-404,"message":"nothing here"}`)), nil
}

func routedClient(t *testing.T, routes pathRouter) *Client {
	t.Helper()
	c := NewClient(Config{CacheDir: t.TempDir(), CacheTTL: time.Hour, Timeout: 5 * time.Second})
	c.hc.Transport = routes
	return c
}

// The bodies a user record is assembled from, with the counts left out so each
// test can supply the one it is about.
const (
	accInfoBody = `{"code":0,"data":{"mid":946974,"name":"a creator"}}`
	relStatBody = `{"code":0,"data":{"follower":17,"following":3}}`
	// upstat answering code 0 with an empty object, which is what it does to
	// anonymous callers and is a refusal rather than a creator with no views.
	upstatRefused = `{"code":0,"data":{}}`
	upstatZero    = `{"code":0,"data":{"archive":{"view":0},"likes":0}}`
	arcSearchBody = `{"code":0,"data":{"list":{"vlist":[]},"page":{"count":924,"pn":1,"ps":1}}}`
)

func userFor(t *testing.T, routes pathRouter) *User {
	t.Helper()
	u, err := routedClient(t, routes).User(context.Background(), "946974")
	if err != nil {
		t.Fatalf("the user record failed: %v", err)
	}
	return u
}

// The defect this milestone is about. upstat refuses to say what a creator's
// view and like totals are, and printing that as 0 states a fact nobody
// measured, in the same breath and the same shape as the counts that were.
func TestARefusedCountIsAbsentRatherThanZero(t *testing.T) {
	u := userFor(t, pathRouter{
		"acc/info":      accInfoBody,
		"relation/stat": relStatBody,
		"upstat":        upstatRefused,
		"arc/search":    arcSearchBody,
	})
	if u.TotalView != nil || u.TotalLike != nil {
		t.Fatalf("a refused count came back as a number: view=%v like=%v", u.TotalView, u.TotalLike)
	}
	for _, f := range []string{"total_view", "total_like"} {
		why, ok := u.Envelope.Missed[f]
		if !ok {
			t.Errorf("%s is missing and the envelope does not say why", f)
			continue
		}
		if !strings.Contains(why, "upstat") {
			t.Errorf("%s: the reason does not name the endpoint that refused: %s", f, why)
		}
	}
	// The counts that did answer are untouched by the one that did not.
	if u.FollowerCount == nil || *u.FollowerCount != 17 {
		t.Errorf("follower_count = %v, want 17", u.FollowerCount)
	}
}

// The other half of the same fact. A creator whose totals really are zero has
// to be able to say so, or the fix above has only moved the lie.
func TestACountOfZeroIsNotAMissingCount(t *testing.T) {
	u := userFor(t, pathRouter{
		"acc/info":      accInfoBody,
		"relation/stat": relStatBody,
		"upstat":        upstatZero,
		"arc/search":    arcSearchBody,
	})
	if u.TotalView == nil || *u.TotalView != 0 {
		t.Fatalf("a real zero came back as %v", u.TotalView)
	}
	if _, ok := u.Envelope.Missed["total_view"]; ok {
		t.Error("a count that answered was recorded as missed")
	}
}

// The json is where the difference has to survive, because that is what people
// pipe into something else. A withheld count is not in the object at all, and a
// zero is in it as a zero.
func TestTheJSONTellsAZeroFromAnAbsentCount(t *testing.T) {
	// The key has to be checked rather than the text, because the name of a
	// withheld field appears in the envelope's missed map either way.
	marshalled := func(upstat string) map[string]any {
		t.Helper()
		b, err := json.Marshal(userFor(t, pathRouter{
			"acc/info": accInfoBody, "relation/stat": relStatBody,
			"upstat": upstat, "arc/search": arcSearchBody,
		}))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	if v, ok := marshalled(upstatRefused)["total_view"]; ok {
		t.Errorf("a withheld count is still a key in the json, as %v", v)
	}
	v, ok := marshalled(upstatZero)["total_view"]
	if !ok {
		t.Fatal("a real zero is not in the json at all")
	}
	if v != float64(0) {
		t.Errorf("total_view = %v, want 0", v)
	}
}

// video_count is on none of the endpoints the user record used to read. The
// listing endpoint has to know it in order to paginate, so it comes from there.
func TestTheUploadCountComesFromTheListingEndpoint(t *testing.T) {
	u := userFor(t, pathRouter{
		"acc/info":      accInfoBody,
		"relation/stat": relStatBody,
		"upstat":        upstatRefused,
		"arc/search":    arcSearchBody,
	})
	if u.VideoCount == nil {
		t.Fatalf("video_count is absent: %v", u.Envelope.Missed)
	}
	if *u.VideoCount != 924 {
		t.Errorf("video_count = %d, want the 924 the listing reported", *u.VideoCount)
	}
}

// And when the listing refuses, which it does to anonymous callers often
// enough, the count is absent rather than zero like everything else here.
func TestARefusedListingLeavesTheUploadCountAbsent(t *testing.T) {
	u := userFor(t, pathRouter{
		"acc/info":      accInfoBody,
		"relation/stat": relStatBody,
		"upstat":        upstatRefused,
		"arc/search":    `{"code":-352,"message":"风控校验失败"}`,
	})
	if u.VideoCount != nil {
		t.Fatalf("a refused listing produced a count of %d", *u.VideoCount)
	}
	if why := u.Envelope.Missed["video_count"]; !strings.Contains(why, "arc/search") {
		t.Errorf("video_count is missing and the envelope does not name the listing: %q", why)
	}
}

// The envelope is the thing that makes the absences above readable, so it has
// to be on every record and not only on the one that needed it first.
func TestEveryRecordTypeCarriesAnEnvelope(t *testing.T) {
	for _, rec := range []any{
		Video{}, User{}, Comment{}, Danmaku{}, Dynamic{}, LiveRoom{},
		Bangumi{}, Audio{}, Article{}, Favorite{}, Stream{}, Suggestion{},
	} {
		rt := reflect.TypeOf(rec)
		f, ok := rt.FieldByName("Envelope")
		if !ok {
			t.Errorf("%s has no envelope", rt.Name())
			continue
		}
		if f.Type != reflect.TypeOf((*Envelope)(nil)) {
			t.Errorf("%s.Envelope is %s", rt.Name(), f.Type)
		}
		// Hidden from the table and the csv, kept in the json. Provenance is
		// worth having on every record and worth a column on none of them.
		if got := f.Tag.Get("table"); got != "-" {
			t.Errorf("%s.Envelope would take a table column, tag is %q", rt.Name(), got)
		}
		if got := f.Tag.Get("json"); got != "envelope,omitempty" {
			t.Errorf("%s.Envelope json tag is %q", rt.Name(), got)
		}
	}
}

// The envelope says which endpoint answered in the same short form the
// requirement matrix uses, so the two can be read against each other.
func TestTheEndpointNameMatchesTheMatrix(t *testing.T) {
	for _, r := range Matrix() {
		if got := endpointName(r.Base); got != r.Name {
			t.Errorf("endpointName(%q) = %q, want %q", r.Base, got, r.Name)
		}
	}
}

// A record built without an envelope is still a record, and nothing that writes
// to one should have to check first.
func TestRecordingAMissOnNoEnvelopeIsHarmless(t *testing.T) {
	var e *Envelope
	e.miss("total_view", "upstat said nothing")
	if e.clone() != nil {
		t.Error("cloning nothing produced something")
	}
}

// One request can produce many records, and a note that belongs to one of them
// must not appear on the rest.
func TestACloneDoesNotShareItsMisses(t *testing.T) {
	base := &Envelope{Endpoint: "x/v2/reply/wbi/main", Status: StatusOK}
	base.miss("replies", "the sub-reply endpoint refused")

	other := base.clone()
	other.miss("location", "not carried on this comment")

	if _, ok := base.Missed["location"]; ok {
		t.Error("a note written on the clone reached the original")
	}
	if base.Missed["replies"] == "" {
		t.Error("the clone lost what the original already knew")
	}
	if other.Missed["replies"] == "" {
		t.Error("the clone did not carry the original's notes")
	}
}

// The reason a field is missing is read next to the value that is not there, so
// it is one clause naming the endpoint and the state rather than the paragraph
// of advice the same refusal prints on stderr.
func TestAMissedFieldSaysWhichEndpointAndWhatState(t *testing.T) {
	note := refusalNote(&APIError{
		Status:   StatusRefusedSilent,
		Message:  "code 0 with no payload",
		Hint:     "this endpoint always carries a payload when it answers",
		Endpoint: upstatBase,
	})
	for _, want := range []string{"x/space/upstat", "refused_silent", "code 0 with no payload"} {
		if !strings.Contains(note, want) {
			t.Errorf("the note does not say %q: %s", want, note)
		}
	}
	if strings.Contains(note, "always carries a payload") {
		t.Errorf("the note carried the stderr advice into every field: %s", note)
	}
}
