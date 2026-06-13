package dmproto

import (
	"encoding/binary"
	"testing"
)

// tag builds a protobuf field key for the given field number and wire type,
// varint-encoded (field numbers >= 16 need more than one byte).
func tag(field, wire int) []byte {
	return binary.AppendUvarint(nil, uint64(field<<3|wire))
}

func varintField(field int, v uint64) []byte {
	return binary.AppendUvarint(tag(field, 0), v)
}

func strField(field int, s string) []byte {
	b := binary.AppendUvarint(tag(field, 2), uint64(len(s)))
	return append(b, s...)
}

func TestDecodeOneElem(t *testing.T) {
	// Build one DanmakuElem: id=123, progress=5000, mode=1, content="你好".
	var elem []byte
	elem = append(elem, varintField(fID, 123)...)
	elem = append(elem, varintField(fProgress, 5000)...)
	elem = append(elem, varintField(fMode, 1)...)
	elem = append(elem, strField(fContent, "你好")...)
	elem = append(elem, strField(fIDStr, "123")...)

	// Wrap it as repeated field 1 of DmSegMobileReply.
	var msg []byte
	msg = append(msg, tag(1, 2)...)
	msg = binary.AppendUvarint(msg, uint64(len(elem)))
	msg = append(msg, elem...)

	out, err := Decode(msg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d elems, want 1", len(out))
	}
	e := out[0]
	if e.ID != 123 || e.Progress != 5000 || e.Mode != 1 || e.Content != "你好" || e.IDStr != "123" {
		t.Fatalf("decoded elem wrong: %+v", e)
	}
}

func TestDecodeMultiple(t *testing.T) {
	var msg []byte
	for i := 1; i <= 3; i++ {
		elem := append(varintField(fID, uint64(i)), strField(fContent, "x")...)
		msg = append(msg, tag(1, 2)...)
		msg = binary.AppendUvarint(msg, uint64(len(elem)))
		msg = append(msg, elem...)
	}
	out, err := Decode(msg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("got %d elems, want 3", len(out))
	}
}

func TestDecodeEmpty(t *testing.T) {
	out, err := Decode(nil)
	if err != nil {
		t.Fatalf("Decode(nil): %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("got %d elems, want 0", len(out))
	}
}
