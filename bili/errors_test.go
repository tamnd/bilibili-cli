package bili

import "testing"

func TestAPIErrorMapping(t *testing.T) {
	cases := []struct {
		code int
		kind ErrKind
	}{
		{-101, ErrAccess},
		{-352, ErrAccess},
		{-403, ErrAccess},
		{-404, ErrNotFound},
		{-412, ErrRate},
		{-509, ErrRate},
		{62002, ErrNotFound},
		{22001, ErrAccess},
		{99999, ErrGeneric},
	}
	for _, tc := range cases {
		e := apiError(tc.code, "msg")
		if e.Code != tc.code {
			t.Errorf("code = %d, want %d", e.Code, tc.code)
		}
		if e.Kind != tc.kind {
			t.Errorf("apiError(%d).Kind = %v, want %v", tc.code, e.Kind, tc.kind)
		}
		if Kind(e) != tc.kind {
			t.Errorf("Kind(apiError(%d)) = %v, want %v", tc.code, Kind(e), tc.kind)
		}
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
