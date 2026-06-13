package bili

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// mixinKeyEncTab is the fixed permutation that mixes the two WBI keys.
var mixinKeyEncTab = [...]int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
	27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
	37, 48, 7, 16, 24, 55, 40, 61, 26, 17, 0, 1, 60, 51, 30, 4,
	22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36, 20, 34, 44, 52,
}

type wbiKeys struct {
	mu      sync.Mutex
	mixin   string
	fetched time.Time
	imgKey  string
	subKey  string
}

// mixinKey derives the 32-character mixin key from the two raw WBI keys.
func mixinKey(imgKey, subKey string) string {
	raw := imgKey + subKey
	var b strings.Builder
	for _, i := range mixinKeyEncTab {
		if i < len(raw) {
			b.WriteByte(raw[i])
		}
	}
	s := b.String()
	if len(s) > 32 {
		s = s[:32]
	}
	return s
}

// keyStem extracts the filename stem from a WBI image URL.
func keyStem(u string) string {
	base := path.Base(u)
	if dot := strings.LastIndexByte(base, '.'); dot >= 0 {
		return base[:dot]
	}
	return base
}

// ensureWBI fetches and caches the current WBI mixin key.
func (c *Client) ensureWBI(ctx context.Context) (string, error) {
	c.wbi.mu.Lock()
	if c.wbi.mixin != "" && time.Since(c.wbi.fetched) < 6*time.Hour {
		k := c.wbi.mixin
		c.wbi.mu.Unlock()
		return k, nil
	}
	c.wbi.mu.Unlock()

	nav, err := c.Nav(ctx)
	if err != nil && nav == nil {
		return "", err
	}
	img := keyStem(nav.WbiImg.ImgURL)
	sub := keyStem(nav.WbiImg.SubURL)
	mk := mixinKey(img, sub)

	c.wbi.mu.Lock()
	c.wbi.imgKey, c.wbi.subKey, c.wbi.mixin, c.wbi.fetched = img, sub, mk, time.Now()
	c.wbi.mu.Unlock()
	return mk, nil
}

// signWBI returns a copy of params with wts and w_rid added per the WBI scheme.
func (c *Client) signWBI(ctx context.Context, params url.Values) (url.Values, error) {
	mk, err := c.ensureWBI(ctx)
	if err != nil {
		return nil, err
	}
	signed := url.Values{}
	for k, v := range params {
		signed[k] = v
	}
	signed.Set("wts", strconv.FormatInt(c.now().Unix(), 10))

	// build the canonical query: sorted keys, values stripped of !'()*
	keys := make([]string, 0, len(signed))
	for k := range signed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var q strings.Builder
	for i, k := range keys {
		if i > 0 {
			q.WriteByte('&')
		}
		val := stripWBIChars(signed.Get(k))
		q.WriteString(wbiEscape(k))
		q.WriteByte('=')
		q.WriteString(wbiEscape(val))
	}
	sum := md5.Sum([]byte(q.String() + mk))
	signed.Set("w_rid", hex.EncodeToString(sum[:]))
	return signed, nil
}

// addDeviceParams adds the browser device-fingerprint parameters that the space
// endpoints (acc/info, arc/search) require to clear risk control (-352). The
// values mirror what a real desktop Chrome WebGL probe sends; the endpoints only
// check for their presence and plausibility, not exact device identity.
func addDeviceParams(v url.Values) url.Values {
	v.Set("dm_img_list", "[]")
	v.Set("dm_img_str", "V2ViR0wgMS4wIChPcGVuR0wgRVMgMi4wIENocm9taXVtKQ")
	v.Set("dm_cover_img_str", "QU5HTEUgKEludGVsLCBJbnRlbChSKSBVSEQgR3JhcGhpY3MgKDB4MDAwMDlCQzQpIERpcmVjdDNEMTEgdnNfNV8wIHBzXzVfMCwgRDNEMTEpR29vZ2xlIEluYy4gKEludGVsKQ")
	v.Set("dm_img_inter", `{"ds":[],"wh":[0,0,0],"of":[0,0,0]}`)
	return v
}

func wbiEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func stripWBIChars(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '!', '\'', '(', ')', '*':
			return -1
		}
		return r
	}, s)
}
