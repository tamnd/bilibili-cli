package bvconv

import "testing"

// av170001 <-> BV17x411w7KC is the canonical worked example from the public
// description of the BV encoding, so it pins the algorithm, not just round-trips.
func TestKnownVector(t *testing.T) {
	if got := ToBV(170001); got != "BV17x411w7KC" {
		t.Fatalf("ToBV(170001) = %q, want BV17x411w7KC", got)
	}
	got, err := ToAV("BV17x411w7KC")
	if err != nil {
		t.Fatalf("ToAV: %v", err)
	}
	if got != 170001 {
		t.Fatalf("ToAV(BV17x411w7KC) = %d, want 170001", got)
	}
}

func TestRoundTrip(t *testing.T) {
	for _, aid := range []int64{1, 2, 170001, 80433022, 116726591198647} {
		bv := ToBV(aid)
		back, err := ToAV(bv)
		if err != nil {
			t.Fatalf("ToAV(%q): %v", bv, err)
		}
		if back != aid {
			t.Errorf("round trip %d -> %q -> %d", aid, bv, back)
		}
	}
}

func TestToAVRejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "BV", "notabvid", "BV0000000000"} {
		if _, err := ToAV(in); err == nil {
			t.Errorf("ToAV(%q) = nil error, want error", in)
		}
	}
}
