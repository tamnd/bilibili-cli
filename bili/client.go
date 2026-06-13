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

// rawGet performs a GET with retries and returns the decompressed body.
func (c *Client) rawGet(ctx context.Context, rawURL string, headers map[string]string) ([]byte, error) {
	if c.cfg.DryRun {
		fmt.Fprintf(dryRunOut, "GET %s\n", rawURL)
		return dryRunBody, nil
	}
	var last error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			d := c.cfg.Rate * time.Duration(attempt*attempt+1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}
		if err := c.throttle(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
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
			resp.Body.Close()
			last = fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, e := time.ParseDuration(ra + "s"); e == nil {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(secs):
					}
				}
			}
			continue
		}
		body, err := readBody(resp)
		resp.Body.Close()
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			last = fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
			continue
		}
		return body, nil
	}
	return nil, &APIError{Code: 0, Message: last.Error(), Hint: "request failed after retries", Kind: ErrNetwork}
}

func readBody(resp *http.Response) ([]byte, error) {
	var r io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
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

// getJSON fetches base+params, unwraps the envelope, and decodes data into out.
// It uses the on-disk cache when enabled.
func (c *Client) getJSON(ctx context.Context, base string, params url.Values, out any) error {
	c.ensureBuvid(ctx)
	full := buildURL(base, params)
	if c.cache != nil && !c.cfg.DryRun {
		if b, ok := c.cache.get(full); ok {
			return decodeEnvelope(b, out)
		}
	}
	body, err := c.rawGet(ctx, full, nil)
	if err != nil {
		return err
	}
	if c.cache != nil && !c.cfg.DryRun {
		c.cache.put(full, body)
	}
	return decodeEnvelope(body, out)
}

// getJSONNoCache always hits the network and never caches.
func (c *Client) getJSONNoCache(ctx context.Context, base string, params url.Values, out any) error {
	body, err := c.rawGet(ctx, buildURL(base, params), nil)
	if err != nil {
		return err
	}
	return decodeEnvelope(body, out)
}

// getJSONSigned WBI-signs the params then behaves like getJSON.
func (c *Client) getJSONSigned(ctx context.Context, base string, params url.Values, out any) error {
	signed, err := c.signWBI(ctx, params)
	if err != nil {
		return err
	}
	return c.getJSON(ctx, base, signed, out)
}

func decodeEnvelope(body []byte, out any) error {
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if env.Code != 0 {
		return apiError(env.Code, env.Message)
	}
	payload := env.payload()
	if out == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

// Raw fetches base+params and returns the untouched response body (the full
// envelope). Used by --raw.
func (c *Client) Raw(ctx context.Context, base string, params url.Values, sign bool) ([]byte, error) {
	c.ensureBuvid(ctx)
	if sign {
		signed, err := c.signWBI(ctx, params)
		if err != nil {
			return nil, err
		}
		params = signed
	}
	return c.rawGet(ctx, buildURL(base, params), nil)
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
