package bili

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// dryRunOut is where DryRun prints the requests it would make.
var dryRunOut io.Writer = os.Stdout

// lsid builds a browser-like b_lsid token from a seed: HEX_HEX where the second
// half is the seed in milliseconds.
func lsid(seed int64) string {
	a := strconv.FormatInt((seed%0xFFFFFF)|0x100000, 16)
	b := strconv.FormatInt(seed*1000, 16)
	return strings.ToUpper(a + "_" + b)
}

// uuidLike builds a _uuid token shaped like the one bilibili's web bootstrap
// sets. Exact randomness is unnecessary; only the shape matters to risk control.
func uuidLike(seed int64) string {
	h := strconv.FormatInt(seed, 16)
	for len(h) < 12 {
		h = "0" + h
	}
	return strings.ToUpper(h[:8] + "-" + h[8:12] + "-" + h[len(h)-4:] + "-" + h[:4] + "-" + h + "infoc")
}

// Client is a typed, signed, rate-limited bilibili web API client.
type Client struct {
	cfg       Config
	hc        *http.Client
	wbi       wbiKeys
	cache     *cache
	nowFn     func() time.Time
	buvidOnce sync.Once

	mu   sync.Mutex
	next time.Time

	cookieMu sync.RWMutex
	cookies  map[string]string // name -> value, sent on every request
}

// NewClient builds a client from cfg, filling defaults for zero fields.
func NewClient(cfg Config) *Client {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Lang == "" {
		cfg.Lang = "zh-CN"
	}
	jar, _ := cookiejar.New(nil)
	tr := &http.Transport{}
	if cfg.Proxy != "" {
		if pu, err := url.Parse(cfg.Proxy); err == nil {
			tr.Proxy = http.ProxyURL(pu)
		}
	}
	c := &Client{
		cfg:     cfg,
		hc:      &http.Client{Timeout: cfg.Timeout, Jar: jar, Transport: tr},
		nowFn:   time.Now,
		cookies: map[string]string{},
	}
	if !cfg.NoCache && cfg.CacheDir != "" {
		c.cache = newCache(cfg.CacheDir, cfg.CacheTTL)
	}
	c.applyCookies()
	return c
}

func (c *Client) now() time.Time { return c.nowFn() }

// applyCookies parses the configured cookie header into the per-request cookie
// map. Cookies are sent via an explicit Cookie header rather than the cookie jar
// because the jar scopes host-only cookies to www.bilibili.com, which would never
// reach the api.bilibili.com / s.search.bilibili.com hosts the API lives on.
func (c *Client) applyCookies() {
	if c.cfg.Cookie == "" {
		return
	}
	for part := range strings.SplitSeq(c.cfg.Cookie, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		c.setCookie(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
}

func (c *Client) setCookie(name, value string) {
	c.cookieMu.Lock()
	c.cookies[name] = value
	c.cookieMu.Unlock()
}

func (c *Client) hasCookie(name string) bool {
	c.cookieMu.RLock()
	defer c.cookieMu.RUnlock()
	return c.cookies[name] != ""
}

// cookieHeader renders the current cookies into a Cookie header value.
func (c *Client) cookieHeader() string {
	c.cookieMu.RLock()
	defer c.cookieMu.RUnlock()
	if len(c.cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(c.cookies))
	for k, v := range c.cookies {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

// ensureBuvid acquires an anonymous buvid cookie once so requests look like a
// real browser even without a login. Several risk-controlled endpoints (e.g.
// ranking/v2, popular/series) return -352 without it.
func (c *Client) ensureBuvid(ctx context.Context) {
	c.buvidOnce.Do(func() {
		if c.hasCookie("buvid3") {
			return
		}
		// getJSONNoCache already unwraps the envelope, so decode the data payload
		// directly (b_3 / b_4 live at the top level of data).
		var spi struct {
			B3 string `json:"b_3"`
			B4 string `json:"b_4"`
		}
		if err := c.getJSONNoCache(ctx, "https://api.bilibili.com/x/frontend/finger/spi", nil, &spi); err != nil {
			return
		}
		if spi.B3 != "" {
			c.setCookie("buvid3", spi.B3)
			c.setCookie("buvid4", spi.B4)
		}
		// b_nut (a unix timestamp) and b_lsid (a session id) accompany buvid3 in a
		// real browser. Stricter endpoints (article/view, dynamic detail) return
		// -352 for a buvid that arrives without them.
		now := c.now().Unix()
		c.setCookie("b_nut", strconv.FormatInt(now, 10))
		c.setCookie("b_lsid", lsid(now))
		c.setCookie("_uuid", uuidLike(now))
	})
}

func (c *Client) throttle(ctx context.Context) error {
	if c.cfg.Rate <= 0 {
		return nil
	}
	c.mu.Lock()
	now := c.now()
	wait := c.next.Sub(now)
	if c.next.Before(now) {
		c.next = now.Add(c.cfg.Rate)
	} else {
		c.next = c.next.Add(c.cfg.Rate)
	}
	c.mu.Unlock()
	if wait <= 0 {
		return nil
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// dryRunBody is returned for every request when DryRun is set: a valid empty
// success envelope so decoders yield no records rather than erroring.
var dryRunBody = []byte(`{"code":0,"message":"dry-run","data":null,"result":null}`)

// result is one HTTP response as it arrived, before anything is decoded.
//
// The status line and the content type are carried alongside the body because
// classification needs them and because collapsing them into an error string
// loses the only evidence that separates an interception from a bad gateway.
type result struct {
	body        []byte
	status      int
	contentType string
	base        string // endpoint without query parameters, for the payload rule

	// dryRun marks a body this client synthesized rather than received. There
	// is nothing in it to classify, and every state except empty would be a
	// statement about a site that was never asked.
	dryRun bool
}

// rawGet performs a GET with retries and returns the decompressed body. It is
// the shape the surfaces that are not JSON still want: the article HTML page,
// the legacy danmaku XML, the suggest endpoint on its own host.
func (c *Client) rawGet(ctx context.Context, rawURL string, headers map[string]string) ([]byte, error) {
	res, err := c.fetch(ctx, rawURL, headers)
	if err != nil {
		return nil, err
	}
	if res.status < 200 || res.status >= 300 {
		st, known := statusForHTTP(res.status)
		return nil, httpError(st, known, res)
	}
	return res.body, nil
}

// fetch performs a GET with retries and returns the whole response.
//
// Which statuses are retried is a decision, not an oversight. A 429 or a 5xx is
// worth asking again, because the server said it was busy. An HTTP 412 is risk
// control saying no to this request from this address, and asking four more
// times in quick succession is the one response guaranteed to make it worse, so
// it returns immediately.
func (c *Client) fetch(ctx context.Context, rawURL string, headers map[string]string) (result, error) {
	// The payload rule is keyed by the endpoint, so the query string comes off
	// once here rather than at every call site.
	base, _, _ := strings.Cut(rawURL, "?")
	if c.cfg.DryRun {
		_, _ = fmt.Fprintf(dryRunOut, "GET %s\n", rawURL)
		return result{body: dryRunBody, status: http.StatusOK, contentType: "application/json", base: base, dryRun: true}, nil
	}
	var last error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			d := c.cfg.Rate * time.Duration(attempt*attempt+1)
			select {
			case <-ctx.Done():
				return result{}, ctx.Err()
			case <-time.After(d):
			}
		}
		if err := c.throttle(ctx); err != nil {
			return result{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return result{}, err
		}
		req.Header.Set("User-Agent", c.cfg.UserAgent)
		req.Header.Set("Referer", Referer)
		// Note: no Origin header. Browsers omit Origin on same-site GETs, and some
		// risk-controlled endpoints (e.g. ranking/v2) reject requests that carry it
		// with code -352.
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Encoding", "gzip")
		if ck := c.cookieHeader(); ck != "" {
			req.Header.Set("Cookie", ck)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			last = fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, e := time.ParseDuration(ra + "s"); e == nil {
					select {
					case <-ctx.Done():
						return result{}, ctx.Err()
					case <-time.After(secs):
					}
				}
			}
			continue
		}
		body, err := readBody(resp)
		_ = resp.Body.Close()
		if err != nil {
			last = err
			continue
		}
		res := result{
			body:        body,
			status:      resp.StatusCode,
			contentType: resp.Header.Get("Content-Type"),
			base:        base,
		}
		// Everything that is left is handed back for classification, including
		// the non 2xx statuses. A 412 carries the HTML interstitial that says
		// what happened, and throwing the body away to build an error string
		// would discard the only useful part of the response.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if _, known := statusForHTTP(resp.StatusCode); !known {
				last = fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
				continue
			}
		}
		return res, nil
	}
	return result{}, &APIError{
		Code:    0,
		Message: last.Error(),
		Hint:    "request failed after retries",
		Status:  StatusNetwork,
	}
}

func readBody(resp *http.Response) ([]byte, error) {
	var r io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gz.Close() }()
		r = gz
	}
	return io.ReadAll(io.LimitReader(r, 64<<20))
}

// envelope is the standard bilibili response wrapper. Most endpoints carry their
// payload in "data"; the pgc/* (bangumi) endpoints use "result" instead.
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	TTL     int             `json:"ttl"`
	Data    json.RawMessage `json:"data"`
	Result  json.RawMessage `json:"result"`
}

// payload returns the data field, falling back to result for pgc endpoints.
func (e envelope) payload() json.RawMessage {
	if len(e.Data) > 0 && string(e.Data) != "null" {
		return e.Data
	}
	return e.Result
}

// buildURL composes a base URL with query parameters.
func buildURL(base string, params url.Values) string {
	if len(params) == 0 {
		return base
	}
	return base + "?" + params.Encode()
}

// getJSON fetches base+params, classifies the response, and decodes the payload
// into out. It uses the on-disk cache when enabled.
func (c *Client) getJSON(ctx context.Context, base string, params url.Values, out any) error {
	c.ensureBuvid(ctx)
	full := buildURL(base, params)
	cacheable := c.cache != nil && !c.cfg.DryRun

	if cacheable {
		if b, ok := c.cache.get(full); ok {
			// A cache written by this version holds answers only. One written
			// by an older version can hold a refusal, and the useful thing to
			// do with it is ignore it and ask again rather than replay it for
			// the rest of the TTL.
			st, err := decodeResult(result{body: b, status: http.StatusOK, contentType: "application/json", base: base}, out)
			if !st.Refused() && err == nil {
				return nil
			}
		}
	}

	res, err := c.fetch(ctx, full, nil)
	if err != nil {
		return err
	}
	st, err := decodeResult(res, out)
	if cacheable && st.Cacheable() {
		c.cache.put(full, res.body)
	}
	return err
}

// getJSONNoCache always hits the network and never caches.
func (c *Client) getJSONNoCache(ctx context.Context, base string, params url.Values, out any) error {
	res, err := c.fetch(ctx, buildURL(base, params), nil)
	if err != nil {
		return err
	}
	_, err = decodeResult(res, out)
	return err
}

// getJSONSigned WBI-signs the params then behaves like getJSON.
func (c *Client) getJSONSigned(ctx context.Context, base string, params url.Values, out any) error {
	signed, err := c.signWBI(ctx, params)
	if err != nil {
		return err
	}
	return c.getJSON(ctx, base, signed, out)
}

// decodeResult classifies res and, only if it answered, decodes the payload
// into out. The classification is returned as well as the error, because the
// caller needs to know whether the response was worth caching and an error
// value alone cannot say that: an empty result is not an error and a
// refused_silent one is.
func decodeResult(res result, out any) (Status, error) {
	st, env, apiErr := classify(res)
	if apiErr != nil {
		return st, apiErr
	}
	payload := env.payload()
	if out == nil || len(payload) == 0 {
		return st, nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return StatusError, fmt.Errorf("decode data: %w", err)
	}
	return st, nil
}

// decodeEnvelope classifies and decodes a body that arrived over a 200 with a
// JSON content type. It is the shape callers that already hold bytes want.
func decodeEnvelope(body []byte, out any) error {
	_, err := decodeResult(result{body: body, status: http.StatusOK, contentType: "application/json"}, out)
	return err
}

// Raw fetches base+params and returns the untouched response body (the full
// envelope). Used by --raw.
func (c *Client) Raw(ctx context.Context, base string, params url.Values, sign bool) ([]byte, error) {
	res, err := c.rawResult(ctx, base, params, sign)
	if err != nil {
		return nil, err
	}
	// --raw hands back the bytes untouched, but not silently: a caller piping
	// this into jq should be told the server sent an HTML interception rather
	// than being left to work it out from the parse error.
	if res.status < 200 || res.status >= 300 {
		st, known := statusForHTTP(res.status)
		return nil, httpError(st, known, res)
	}
	if isHTML(res.contentType) {
		return nil, riskError(res)
	}
	return res.body, nil
}

// rawResult fetches base+params and returns the whole unclassified response.
//
// Unlike Raw it does not turn a non 2xx into an error, because its caller is
// the probe, whose entire job is to look at what came back and name it.
func (c *Client) rawResult(ctx context.Context, base string, params url.Values, sign bool) (result, error) {
	c.ensureBuvid(ctx)
	if sign {
		signed, err := c.signWBI(ctx, params)
		if err != nil {
			return result{}, err
		}
		params = signed
	}
	return c.fetch(ctx, buildURL(base, params), nil)
}

// Nav returns login state and the current WBI keys. The nav endpoint returns
// code -101 when anonymous but still carries the WBI keys in data, so this
// decodes data regardless of code.
func (c *Client) Nav(ctx context.Context) (*Nav, error) {
	body, err := c.rawGet(ctx, "https://api.bilibili.com/x/web-interface/nav", nil)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	var n Nav
	if len(env.Data) > 0 {
		_ = json.Unmarshal(env.Data, &n)
	}
	if env.Code != 0 && env.Code != -101 {
		return &n, apiError(env.Code, env.Message)
	}
	return &n, nil
}

// SetNow overrides the clock (testing).
func (c *Client) SetNow(f func() time.Time) { c.nowFn = f }
