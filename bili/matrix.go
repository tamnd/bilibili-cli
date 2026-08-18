package bili

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The requirement matrix.
//
// What an endpoint needs in order to answer is a measured fact about the site,
// not a property of the URL. Two endpoints under the same /wbi/ path prefix can
// disagree about whether they want a signature, and an endpoint that answers a
// bare request today can start demanding one next month without anything in the
// test suite noticing. This file is where those facts are written down, and
// Probe is how they get re-measured.
//
// Every row was measured against the live API on 2026-08-19. The offline test
// in matrix_test.go asserts that the client's own signing decisions still agree
// with this table, so changing one without the other fails the build. The
// weekly drift job runs Probe against the real site and reports the rows that
// moved.

// Requirement records what one endpoint needs and what it is known to return.
//
// Buvid and Signed are two independent gates and the difference between them
// matters. A refusal measured against a request that has neither says only that
// one of them was missing, and the obvious reading, that it wanted a signature,
// is wrong at least once: ranking/v2 answers a completely unsigned request as
// long as the buvid cookies are present.
type Requirement struct {
	// Name is the short form used in reports and in error messages. It is the
	// path with the host and the leading slash removed.
	Name string

	// Base is the full endpoint URL.
	Base string

	// Params is a request against an object stable enough to still exist in a
	// year. Nothing here is user data.
	Params url.Values

	// Buvid records that the endpoint refuses a request without the anonymous
	// buvid cookies. It is documentation rather than a switch, because the
	// client acquires them once and then sends them everywhere.
	Buvid bool

	// Signed records that the endpoint refuses a request without a WBI
	// signature even when the buvid cookies are present. This is the field the
	// client's behaviour is checked against.
	Signed bool

	// Device records that the client also sends the dm_img_* browser
	// fingerprint parameters. Only acc/info and arc/search were measured as
	// actually requiring them.
	Device bool

	// Payload records that the endpoint carries a payload when it answers. An
	// endpoint with Payload true that returns code 0 carrying nothing has
	// refused, and this field is the only thing that makes that
	// distinguishable. See 02_extraction.md section 2.1.
	//
	// It is not the same question as Expect, and conflating the two is easy.
	// Payload is about the endpoint: does a real answer from here have
	// something in it. Expect is about today: what does a correct request get
	// right now. upstat has Payload true and Expect refused_silent, and both
	// are accurate. It carries view and like totals when it answers, and it
	// does not answer anonymously. That pair is exactly what makes its silence
	// nameable rather than something to shrug at.
	Payload bool

	// Expect is the state a correctly formed request produces today. It is
	// empty for the rows that answer, and set for the three that do not, so
	// that a probe reports drift when one of them starts answering as well as
	// when one of them stops.
	Expect Status

	// Note explains any row that would otherwise look like a mistake.
	Note string

	// Advice is what to tell the person who hit a refusal on this endpoint,
	// and it is only worth setting where there is something to say beyond "get
	// a cookie". The folder listing is the case that earns the field: the
	// listing is refused and the folder contents are not, so a caller who
	// knows a media_id can still read that folder. A refusal that names a way
	// forward is a different experience from one that names a wall.
	Advice string
}

// Matrix returns the requirement matrix. The objects it names are deliberately
// boring: the oldest article on the site, a partition that has existed since
// the beginning, a room number in the first thousand.
func Matrix() []Requirement {
	const (
		bvid = "BV1gtgE6AEmZ"
		aid  = "117085992648879"
		cid  = "40861174685"
		mid  = "946974"
		room = "1029"
	)

	return []Requirement{
		{
			Name:    "x/web-interface/view",
			Base:    "https://api.bilibili.com/x/web-interface/view",
			Params:  vals("bvid", bvid),
			Payload: true,
		},
		{
			Name:    "x/web-interface/archive/related",
			Base:    "https://api.bilibili.com/x/web-interface/archive/related",
			Params:  vals("bvid", bvid),
			Payload: true,
		},
		{
			Name:    "x/tag/archive/tags",
			Base:    "https://api.bilibili.com/x/tag/archive/tags",
			Params:  vals("bvid", bvid),
			Payload: true,
		},
		{
			Name:    "x/v2/reply/wbi/main",
			Base:    "https://api.bilibili.com/x/v2/reply/wbi/main",
			Params:  url.Values{"oid": {aid}, "type": {"1"}, "mode": {"3"}},
			Signed:  true,
			Payload: true,
			Note:    "answers -403 without a signature, which is a signing failure wearing a permission code",
		},
		{
			Name:    "x/v2/dm/web/seg.so",
			Base:    "https://api.bilibili.com/x/v2/dm/web/seg.so",
			Params:  url.Values{"type": {"1"}, "oid": {cid}, "pid": {aid}, "segment_index": {"1"}},
			Payload: true,
			Note:    "protobuf, not JSON. Probe only checks that bytes came back",
		},
		{
			Name:    "x/space/wbi/acc/info",
			Base:    "https://api.bilibili.com/x/space/wbi/acc/info",
			Params:  vals("mid", mid),
			Buvid:   true,
			Signed:  true,
			Device:  true,
			Payload: true,
		},
		{
			Name:    "x/space/wbi/arc/search",
			Base:    "https://api.bilibili.com/x/space/wbi/arc/search",
			Params:  url.Values{"mid": {mid}, "ps": {"5"}, "pn": {"1"}},
			Buvid:   true,
			Signed:  true,
			Device:  true,
			Payload: true,
		},
		{
			Name:    "x/relation/stat",
			Base:    "https://api.bilibili.com/x/relation/stat",
			Params:  vals("vmid", mid),
			Payload: true,
		},
		{
			Name:    "x/space/upstat",
			Base:    "https://api.bilibili.com/x/space/upstat",
			Params:  vals("mid", mid),
			Payload: true,
			Expect:  stateRefusedSilent,
			Note:    "answers code 0 with an empty object, signed or not. It carries a creator's view and like totals when it answers, which is why the empty object is a refusal. bili user used to print those totals as zero when this happened and now leaves them out and names this endpoint in the record's envelope",
			Advice:  "The creator's view and like totals come from here and nowhere else, so they are unavailable anonymously rather than zero",
		},
		{
			Name:    "x/polymer/web-dynamic/v1/feed/space",
			Base:    "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space",
			Params:  url.Values{"host_mid": {mid}, "features": {"itemOpusStyle"}, "platform": {"web"}, "web_location": {"333.999"}},
			Buvid:   true,
			Signed:  true,
			Device:  true,
			Payload: true,
			Note:    "the only endpoint measured that refuses with an HTTP 412 and an HTML body rather than a code. Measured again on 2026-08-19 answering 200 to an anonymous caller with has_more false and an empty items array, which is a refusal the payload rule cannot see: the payload is there and it is empty inside. The paging rule in dynamic.go is what names that one, so this row stays Expect empty and the probe reports it as answering",
		},
		{
			Name:    "x/web-interface/popular",
			Base:    "https://api.bilibili.com/x/web-interface/popular",
			Params:  url.Values{"ps": {"5"}, "pn": {"1"}},
			Payload: true,
		},
		{
			Name:    "x/web-interface/popular/series/one",
			Base:    "https://api.bilibili.com/x/web-interface/popular/series/one",
			Params:  vals("number", "1"),
			Buvid:   true,
			Payload: true,
			Note:    "refused -352 when first measured without the buvid cookies, and answers with them. This row is why the buvid and the signature are recorded separately",
		},
		{
			Name:    "x/web-interface/ranking/v2",
			Base:    "https://api.bilibili.com/x/web-interface/ranking/v2",
			Params:  url.Values{"rid": {"0"}, "type": {"all"}},
			Buvid:   true,
			Payload: true,
			Note:    "refused without the buvid cookies and answers unsigned with them, which is why the two gates are separate fields",
		},
		{
			Name:    "x/web-interface/wbi/search/type",
			Base:    "https://api.bilibili.com/x/web-interface/wbi/search/type",
			Params:  url.Values{"search_type": {"video"}, "keyword": {"lofi"}, "page": {"1"}},
			Payload: true,
			Note:    "wbi in the path and no signature required. This row is the reason the path is not the rule. The client signs it anyway, which is allowed",
		},
		{
			Name:    "x/web-interface/wbi/search/square",
			Base:    "https://api.bilibili.com/x/web-interface/wbi/search/square",
			Params:  vals("limit", "10"),
			Payload: true,
		},
		{
			Name:    "x/article/view",
			Base:    "https://api.bilibili.com/x/article/view",
			Params:  vals("id", "1"),
			Payload: true,
			Note:    "returns -509 under a fast probe cadence. That is a rate state and it clears by waiting",
		},
		{
			Name:    "x/v3/fav/resource/list",
			Base:    "https://api.bilibili.com/x/v3/fav/resource/list",
			Params:  url.Values{"media_id": {"1952291424"}, "ps": {"5"}, "pn": {"1"}, "platform": {"web"}},
			Payload: true,
			Note:    "answers, and the answer is not the whole of what was asked. Measured on 2026-08-19 returning info.media_count 145 with medias null and has_more true to an anonymous caller, which is the folder's metadata with its contents withheld. The payload rule cannot see that, since a payload is present; favorite.go compares the item count against media_count instead",
		},
		{
			Name:    "x/v3/fav/folder/created/list-all",
			Base:    "https://api.bilibili.com/x/v3/fav/folder/created/list-all",
			Params:  vals("up_mid", mid),
			Payload: true,
			Expect:  stateRefusedSilent,
			Note:    "answers code 0 with a null payload, signed or not. It carries the folder list when it answers, so the null is a refusal rather than a user with no folders. Reading a folder by media_id was a way around this when the row was first measured and no longer is: see the resource/list row",
			Advice:  "Reading the folder directly by its id used to be the way around this and is not any more, so this needs a logged-in cookie rather than a different command",
		},
		{
			Name:    "room/v1/Room/get_info",
			Base:    "https://api.live.bilibili.com/room/v1/Room/get_info",
			Params:  vals("room_id", room),
			Buvid:   true,
			Payload: true,
			Note:    "a different host, and the buvid it needs was issued by api.bilibili.com",
		},
		{
			Name:    "room/v3/area/getRoomList",
			Base:    "https://api.live.bilibili.com/room/v3/area/getRoomList",
			Params:  url.Values{"parent_area_id": {"1"}, "page": {"1"}, "page_size": {"5"}},
			Payload: true,
		},
		{
			Name:    "pgc/view/web/season",
			Base:    "https://api.bilibili.com/pgc/view/web/season",
			Params:  vals("season_id", "33802"),
			Payload: true,
			Note:    "puts its payload in result rather than data. An observer that only reads data records this as a refusal, which is how it was misread once already",
		},
		{
			Name:    "pgc/review/user",
			Base:    "https://api.bilibili.com/pgc/review/user",
			Params:  vals("media_id", "28229233"),
			Payload: true,
			Note:    "the md to ss resolver, and the other half of the result rather than data pair",
		},
		{
			Name:    "audio/music-service-c/web/song/info",
			Base:    "https://www.bilibili.com/audio/music-service-c/web/song/info",
			Params:  vals("sid", "1"),
			Payload: false,
			Expect:  stateNotFound,
			Note:    "answers 4511001 for every sid tried, including 1 and 999999999, so the code is a constant and not an echo. Payload is false because this surface has never been observed answering, so there is nothing measured to record",
		},
	}
}

// Observation is one row of the matrix as the site answered it just now.
type Observation struct {
	Requirement Requirement `json:"-"`
	Name        string      `json:"endpoint"`

	// Bare and WithSignature are the two columns. Bare is always measured.
	// WithSignature is only measured for rows recorded as needing one, because
	// a second live request against an endpoint that already answered buys
	// nothing and costs bilibili a request.
	Bare          Status `json:"bare"`
	WithSignature Status `json:"signed,omitempty"`

	// Retried records that the row refused once and was asked again after a
	// pause, which is the only way to tell a site change from probe pressure.
	Retried bool `json:"retried,omitempty"`

	// Unmeasured records that the site rate limited both attempts, so this run
	// learned nothing about the row either way. It is deliberately not Moved.
	// "The requirement changed" and "we could not find out" are different
	// claims, and only the first one is worth waking somebody up for.
	Unmeasured bool `json:"unmeasured,omitempty"`

	// Moved is set when the observation contradicts the recorded row.
	Moved  bool   `json:"moved"`
	Reason string `json:"reason,omitempty"`
}

// Probe measures one row against the live API once and reports whether the
// answer matches what is recorded. It never uses the cache: a cached answer
// would describe the site on the day it was cached, which is the one thing this
// is trying not to do.
//
// A single Probe cannot tell a site change from probe pressure. Use
// VerifyMatrix, which asks the refusing rows again.
func (c *Client) Probe(ctx context.Context, r Requirement) Observation {
	o := Observation{Requirement: r, Name: r.Name}

	params := r.Params
	if r.Device {
		params = addDeviceParams(params)
	}

	o.Bare = c.observe(ctx, r, params, false)

	// An endpoint recorded as gated is asked both ways, and only one of the two
	// answers can be drift.
	//
	// The signed request is the one the client actually makes, so a signed
	// request that stops answering is a user facing break and is reported.
	//
	// The unsigned request is measured for the report and not judged. An
	// endpoint that suddenly answers without a signature has not broken
	// anything: the safe response is to keep signing, which is what we already
	// do. It is also not a stable observation. feed/space refused an unsigned
	// request on one run and answered the identical one twenty minutes later,
	// so treating a single unsigned success as evidence that the gate is gone
	// would open an issue most weeks and mean nothing.
	if r.Signed {
		o.WithSignature = c.observe(ctx, r, params, true)
		if o.WithSignature != r.expect() {
			o.Moved, o.Reason = true, "recorded as answering when signed, and it did not"
		} else if o.Bare == stateOK {
			o.Reason = "answered without a signature this time, which the client does not rely on"
		}
		return o
	}

	if want := r.expect(); o.Bare != want {
		o.Moved = true
		o.Reason = "recorded as answering " + string(want) + " and it answered " + string(o.Bare)
	}
	return o
}

// VerifyMatrix walks the whole matrix and reports what moved.
//
// It is two passes, and the second one is the point. A run of two dozen
// requests from one address is the exact shape risk control exists to slow
// down, so a refusal partway through a run is at least as likely to be us as
// it is to be the site. x/article/view is the reliable demonstration: it
// answers a cold request and refuses inside a run, and its penalty window
// outlasts a one minute pause.
//
// So the refusing rows are collected, the run pauses once, and then they are
// asked again. Pausing once rather than once per row is what keeps this
// bounded: ten refusing rows and one refusing row cost the same wait, which
// means a site that has tightened everywhere still finishes inside the drift
// job's timeout instead of being killed halfway and reported as a partial run.
//
// progress, if non-nil, is called with each observation as it is measured, so a
// long run says something before it says everything.
func (c *Client) VerifyMatrix(ctx context.Context, rows []Requirement, progress func(Observation)) ([]Observation, error) {
	out := make([]Observation, len(rows))
	var recheck []int

	for i, r := range rows {
		o := c.Probe(ctx, r)
		out[i] = o
		if o.Moved {
			recheck = append(recheck, i)
		}
		if progress != nil {
			progress(o)
		}
	}

	if len(recheck) == 0 {
		return out, nil
	}
	if err := sleepCtx(ctx, ProbeBackoff); err != nil {
		return out, err
	}

	for _, i := range recheck {
		o := c.Probe(ctx, rows[i])
		o.Retried = true
		switch {
		case !o.Moved:
			// It answered the second time, so the first refusal was the run
			// and not the site. Keep the passing observation and say that it
			// took two attempts.
			o.Reason = "refused on the first pass and answered on the second, which is probe pressure rather than drift"
		case o.Bare == stateRate:
			// Rate limited twice. A -509 is the site saying we are asking too
			// often, which is a fact about our request volume and not about
			// what the endpoint requires. This run did not measure the row.
			//
			// x/article/view is the one that does this: its penalty window is
			// per address and cumulative, so a machine that has probed it a
			// few times recently cannot get an answer out of it at any pace.
			// Reporting that as drift would open an issue every week that says
			// nothing, which is how a drift job stops being read.
			o.Moved, o.Unmeasured = false, true
			o.Reason = "rate limited on both passes, so this run did not measure the row"
		default:
			o.Reason += ", and again after " + ProbeBackoff.String()
		}
		out[i] = o
		if progress != nil {
			progress(o)
		}
	}
	return out, nil
}

// ProbeBackoff is how long VerifyMatrix waits before asking the refusing rows
// again. Three minutes rather than one because x/article/view was measured
// refusing a minute after a run and answering three minutes after it, and a
// backoff shorter than the longest penalty window observed just reports that
// window as drift every week.
const ProbeBackoff = 3 * time.Minute

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// expect is the state this row should be in. Rows that answer say nothing and
// mean ok, which keeps the common case out of the table.
func (r Requirement) expect() Status {
	if r.Expect == "" {
		return stateOK
	}
	return r.Expect
}

// The response states. M2 moves these into the client itself and makes every
// request go through them; here they are what Probe reports.
// The probe reports the same seven states the client sorts live responses into,
// because they are the same question asked from two directions. status.go owns
// them; this file used to carry a second copy and the two could disagree.
const (
	stateOK            = StatusOK
	stateRefusedSilent = StatusRefusedSilent
	stateRisk          = StatusRisk
	stateForbidden     = StatusForbidden
	stateNotFound      = StatusNotFound
	stateRate          = StatusRate
	stateError         = StatusError
)

// observe makes one request and names what came back. It deliberately does not
// decode into a record type: the question here is what state the response is
// in, and answering that has to work for an endpoint whose payload shape this
// tool does not model.
// observe makes one request and reports the state it came back in.
//
// The classification is the client's own, not a second copy written for the
// probe. That matters more than it looks: a drift job that decides for itself
// what a refusal is can only tell you that the probe's idea of the site
// changed. This one tells you that the tool's idea of the site changed, which
// is the thing anybody actually wants to know.
func (c *Client) observe(ctx context.Context, r Requirement, params url.Values, sign bool) Status {
	res, err := c.rawResult(ctx, r.Base, params, sign)
	if err != nil {
		return classifyErr(err)
	}

	// The danmaku segment endpoint is protobuf. There is no envelope to read,
	// so the only questions available are whether it was intercepted and
	// whether any bytes came back.
	if strings.HasSuffix(r.Base, ".so") {
		st, _ := classifyBinary(res)
		return st
	}

	st, _, _ := classify(res)
	return st
}

func classifyCode(code int) Status {
	st, _ := statusForCode(code)
	return st
}

// classifyErr names the state behind a transport level failure. rawGet reports
// these as an APIError with a zero code and the HTTP status in the message,
// which is why this reads the status rather than the code.
func classifyErr(err error) Status {
	var ae *APIError
	if errors.As(err, &ae) {
		if ae.Status != "" && ae.Status != StatusError {
			return ae.Status
		}
		if ae.Code != 0 {
			return classifyCode(ae.Code)
		}
	}
	switch httpStatus(err.Error()) {
	case http.StatusPreconditionFailed:
		return stateRisk
	case http.StatusTooManyRequests:
		return stateRate
	case http.StatusNotFound:
		return stateNotFound
	case http.StatusForbidden:
		return stateForbidden
	}
	return stateError
}

// httpStatus pulls the code out of rawGet's "HTTP 412 from https://..." and
// returns 0 for anything else. Matching on the whole prefix rather than on the
// three digits matters: a URL can contain 412 and a query parameter often does.
func httpStatus(msg string) int {
	i := strings.Index(msg, "HTTP ")
	if i < 0 || len(msg) < i+8 {
		return 0
	}
	n, err := strconv.Atoi(msg[i+5 : i+8])
	if err != nil {
		return 0
	}
	return n
}
