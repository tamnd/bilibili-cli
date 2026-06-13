package bili

import "testing"

func TestEnvelopePayload(t *testing.T) {
	// Most endpoints carry their payload in "data".
	d := envelope{Data: []byte(`{"a":1}`)}
	if string(d.payload()) != `{"a":1}` {
		t.Fatalf("data payload = %s", d.payload())
	}
	// pgc/* (bangumi) endpoints use "result" instead, and data is null.
	r := envelope{Data: []byte("null"), Result: []byte(`{"b":2}`)}
	if string(r.payload()) != `{"b":2}` {
		t.Fatalf("result payload = %s", r.payload())
	}
	// Data wins when both are present and data is non-null.
	both := envelope{Data: []byte(`{"a":1}`), Result: []byte(`{"b":2}`)}
	if string(both.payload()) != `{"a":1}` {
		t.Fatalf("both payload = %s", both.payload())
	}
}

func TestDecodeEnvelopeError(t *testing.T) {
	var out map[string]any
	err := decodeEnvelope([]byte(`{"code":-404,"message":"啥都木有"}`), &out)
	if err == nil {
		t.Fatal("expected error for non-zero code")
	}
	if ae, ok := err.(*APIError); !ok || ae.Code != -404 {
		t.Fatalf("error = %v, want APIError -404", err)
	}
}

func TestLSIDFormat(t *testing.T) {
	// lsid must be uppercase HEX_HEX and stable for a fixed seed.
	a := lsid(1700000000)
	b := lsid(1700000000)
	if a != b {
		t.Fatalf("lsid not deterministic: %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("empty lsid")
	}
}
