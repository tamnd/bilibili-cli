package bili

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// These two keys are the published WBI example keys; the mixin they derive is a
// fixed, documented value, so this pins the permutation table, not a round trip.
const (
	testImgKey = "7cd084941338484aae1ad9425b84077c"
	testSubKey = "4932caff0ff746eab6f01bf08b70ac45"
	testMixin  = "ea1db124af3c7062474693fa704f4ff8"
)

func TestMixinKey(t *testing.T) {
	got := mixinKey(testImgKey, testSubKey)
	if got != testMixin {
		t.Fatalf("mixinKey = %q, want %q", got, testMixin)
	}
	if len(got) != 32 {
		t.Fatalf("mixin key length = %d, want 32", len(got))
	}
}

func TestKeyStem(t *testing.T) {
	u := "https://i0.hdslb.com/bfs/wbi/7cd084941338484aae1ad9425b84077c.png"
	if got := keyStem(u); got != testImgKey {
		t.Fatalf("keyStem = %q, want %q", got, testImgKey)
	}
}

// signWBI must be deterministic for a fixed clock and mixin key. The expected
// w_rid was computed independently from the same algorithm inputs.
func TestSignWBIDeterministic(t *testing.T) {
	c := NewClient(DefaultConfig())
	c.SetNow(func() time.Time { return time.Unix(1700000000, 0) })
	c.wbi.mixin = testMixin
	c.wbi.fetched = c.now()

	signed, err := c.signWBI(context.Background(), url.Values{"foo": {"bar"}, "baz": {"1"}})
	if err != nil {
		t.Fatalf("signWBI: %v", err)
	}
	if got := signed.Get("wts"); got != "1700000000" {
		t.Fatalf("wts = %q, want 1700000000", got)
	}
	if got := signed.Get("w_rid"); got != "0c5f11a238916d4556aeff87fbbca276" {
		t.Fatalf("w_rid = %q, want 0c5f11a238916d4556aeff87fbbca276", got)
	}
}

func TestWBIEscapeSpace(t *testing.T) {
	// WBI uses %20 for spaces, not the + that url.QueryEscape would produce.
	if got := wbiEscape("a b"); got != "a%20b" {
		t.Fatalf("wbiEscape = %q, want a%%20b", got)
	}
}

func TestStripWBIChars(t *testing.T) {
	if got := stripWBIChars("a!b'c(d)e*f"); got != "abcdef" {
		t.Fatalf("stripWBIChars = %q, want abcdef", got)
	}
}

func TestAddDeviceParams(t *testing.T) {
	v := addDeviceParams(url.Values{})
	for _, k := range []string{"dm_img_list", "dm_img_str", "dm_cover_img_str", "dm_img_inter"} {
		if v.Get(k) == "" {
			t.Errorf("addDeviceParams missing %s", k)
		}
	}
}

// A cached mixin key means no nav request, and that has to hold on the injected
// clock as well as the real one.
//
// It did not. The freshness window was measured with time.Since while the
// stored moment came from the injected clock, so a caller who pinned the clock
// got a cache that never hit and a nav fetch on every signed call. The test
// above hid it by passing anyway: it reached the real API, got a real key, and
// checked a signature computed from the key it had set rather than the one it
// used. Denying the network is what surfaced it.
func TestACachedMixinKeyCostsNoRequest(t *testing.T) {
	c := NewClient(DefaultConfig())
	c.SetNow(func() time.Time { return time.Unix(1700000000, 0) })
	c.hc.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("the client asked for %s and a cached key needs no requests", r.URL)
	})
	c.wbi.mixin = testMixin
	c.wbi.fetched = c.now()

	got, err := c.ensureWBI(context.Background())
	if err != nil {
		t.Fatalf("ensureWBI: %v", err)
	}
	if got != testMixin {
		t.Errorf("ensureWBI = %q, want the cached %q", got, testMixin)
	}
}

// And a key older than the window is refetched, so the fix above did not turn
// the freshness check off.
func TestAStaleMixinKeyIsRefetched(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Retries = 0 // the refetch is expected to fail, and waiting out four backoffs proves nothing
	c := NewClient(cfg)
	c.SetNow(func() time.Time { return time.Unix(1700000000, 0) })
	c.hc.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("asked for %s", r.URL)
	})
	c.wbi.mixin = testMixin
	c.wbi.fetched = c.now().Add(-7 * time.Hour)

	if _, err := c.ensureWBI(context.Background()); err == nil {
		t.Error("a key seven hours old was served from the cache")
	}
}

// roundTripperFunc is a transport that answers with whatever the function says,
// which here is always a failure. A test that must not reach the network is
// better served by a transport that cannot than by trusting that it will not.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
