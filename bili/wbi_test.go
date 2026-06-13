package bili

import (
	"context"
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
