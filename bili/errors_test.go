package bili

import (
	"fmt"
	"strings"
	"testing"
)

func TestAPIErrorMapping(t *testing.T) {
	cases := []struct {
		code int
		kind ErrKind
	}{
		{-101, ErrForbidden},
		{-352, ErrRisk},
		{-403, ErrForbidden},
		{-404, ErrNotFound},
		{-412, ErrRisk},
		{-509, ErrRate},
		{62002, ErrNotFound},
		{4511001, ErrNotFound},
		{22001, ErrForbidden},
		{-400, ErrGeneric},
		{99999, ErrGeneric},
	}
	for _, tc := range cases {
		e := apiError(tc.code, "msg")
		if e.Code != tc.code {
			t.Errorf("code = %d, want %d", e.Code, tc.code)
		}
		if e.Status != tc.kind {
			t.Errorf("apiError(%d).Status = %v, want %v", tc.code, e.Status, tc.kind)
		}
		if Kind(e) != tc.kind {
			t.Errorf("Kind(apiError(%d)) = %v, want %v", tc.code, Kind(e), tc.kind)
		}
	}
}

// The kind of an error and the state of its response used to be two enums that
// had to be kept in agreement by hand. They are one type now, and this is the
// test that says so, because the moment they are two again the drift starts.
func TestEveryKindIsAState(t *testing.T) {
	kinds := map[string]ErrKind{
		"ErrRisk":          ErrRisk,
		"ErrForbidden":     ErrForbidden,
		"ErrEmpty":         ErrEmpty,
		"ErrRefusedSilent": ErrRefusedSilent,
		"ErrNetwork":       ErrNetwork,
		"ErrRate":          ErrRate,
		"ErrNotFound":      ErrNotFound,
		"ErrGeneric":       ErrGeneric,
	}
	states := map[Status]bool{
		StatusRisk: true, StatusForbidden: true, StatusEmpty: true,
		StatusRefusedSilent: true, StatusNetwork: true, StatusRate: true,
		StatusNotFound: true, StatusError: true, StatusOK: true,
	}
	for name, k := range kinds {
		if !states[k] {
			t.Errorf("%s = %q, which is not one of the states", name, k)
		}
	}
}

// Kind unwraps, because the CLI adds what was asked in front of every failure
// on its way out and the exit code is read after that has happened.
func TestKindSeesThroughWrapping(t *testing.T) {
	inner := apiError(4511001, "音频未找到或已下架")
	wrapped := fmt.Errorf("audio au1: %w", inner)
	if got := Kind(wrapped); got != ErrNotFound {
		t.Errorf("Kind(wrapped) = %q, want %q", got, ErrNotFound)
	}
	if got := Kind(fmt.Errorf("plain")); got != ErrGeneric {
		t.Errorf("Kind(plain) = %q, want %q", got, ErrGeneric)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	e := apiError(-352, "风控校验失败")
	if e.Error() == "" {
		t.Fatal("empty error string")
	}
	// The hint should be present for a mapped code.
	if e.Hint == "" {
		t.Fatal("expected a hint for -352")
	}
}

// 4511001 is not the track id echoed back. It is the constant this endpoint
// uses for every missing track, and it has to read as one.
func TestAudioNotFoundReadsAsNotFound(t *testing.T) {
	e := apiError(4511001, "音频未找到或已下架")
	e.Endpoint = "https://www.bilibili.com/audio/music-service-c/web/song/info"
	if e.Status != StatusNotFound {
		t.Errorf("status = %q, want %q", e.Status, StatusNotFound)
	}
	msg := e.Error()
	for _, want := range []string{"audio not found or removed", "4511001", "audio/music-service-c"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q does not mention %q", msg, want)
		}
	}
}
