package bili

import (
	"context"
	"testing"
)

func TestResolveClassify(t *testing.T) {
	c := NewClient(DefaultConfig())
	ctx := context.Background()
	cases := []struct {
		in   string
		kind IDKind
		want func(*Identity) bool
	}{
		{"BV1GJ411x7h7", KindVideo, func(id *Identity) bool { return id.BVID == "BV1GJ411x7h7" && id.AID == 80433022 }},
		{"https://www.bilibili.com/video/BV1GJ411x7h7", KindVideo, func(id *Identity) bool { return id.BVID == "BV1GJ411x7h7" }},
		{"av170001", KindVideo, func(id *Identity) bool { return id.AID == 170001 && id.BVID == "BV17x411w7KC" }},
		{"https://space.bilibili.com/2", KindUser, func(id *Identity) bool { return id.Mid == 2 }},
		{"ss33802", KindBangumi, func(id *Identity) bool { return id.SeasonID == 33802 }},
		{"ep331204", KindEpisode, func(id *Identity) bool { return id.EpID == 331204 }},
		{"md28229233", KindMedia, func(id *Identity) bool { return id.MediaID == 28229233 }},
		{"au1", KindAudio, func(id *Identity) bool { return id.SID == 1 }},
		{"cv7018872", KindArticle, func(id *Identity) bool { return id.CVID == 7018872 }},
		{"ml123", KindFavorite, func(id *Identity) bool { return id.FavID == 123 }},
		{"https://live.bilibili.com/5440", KindLive, func(id *Identity) bool { return id.RoomID == 5440 }},
		{"https://t.bilibili.com/1211848387244064839", KindDynamic, func(id *Identity) bool { return id.Dynamic == "1211848387244064839" }},
	}
	for _, tc := range cases {
		id, err := c.Resolve(ctx, tc.in)
		if err != nil {
			t.Errorf("Resolve(%q): %v", tc.in, err)
			continue
		}
		if id.Kind != tc.kind {
			t.Errorf("Resolve(%q).Kind = %q, want %q", tc.in, id.Kind, tc.kind)
		}
		if !tc.want(id) {
			t.Errorf("Resolve(%q) fields wrong: %+v", tc.in, id)
		}
	}
}

func TestResolveUnknown(t *testing.T) {
	c := NewClient(DefaultConfig())
	if _, err := c.Resolve(context.Background(), "this is not an id"); err == nil {
		t.Fatal("Resolve of garbage returned nil error")
	}
}

func TestNormalizeBV(t *testing.T) {
	if got := normalizeBV("bv1GJ411x7h7"); got != "BV1GJ411x7h7" {
		t.Fatalf("normalizeBV = %q, want BV1GJ411x7h7", got)
	}
}
