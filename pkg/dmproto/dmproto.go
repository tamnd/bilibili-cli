// Package dmproto decodes Bilibili's danmaku (bullet-chat) protobuf segments
// without protoc or any generated code. It reads exactly the one message shape
// the web danmaku endpoint returns: a DmSegMobileReply whose field 1 is a
// repeated DanmakuElem.
package dmproto

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Elem is one decoded danmaku line.
type Elem struct {
	ID       int64  // dmid
	Progress int32  // milliseconds into the video
	Mode     int32  // 1-3 scroll, 4 bottom, 5 top, 6 reverse, 7 advanced, 8 code, 9 BAS
	Fontsize int32  // font size
	Color    uint32 // decimal RGB
	MidHash  string // sender id hash
	Content  string // the text
	Ctime    int64  // unix send time
	Weight   int32  // shielding weight
	Pool     int32  // 0 normal, 1 subtitle, 2 special
	IDStr    string // dmid as string
}

// field numbers inside DanmakuElem
const (
	fID       = 1
	fProgress = 2
	fMode     = 3
	fFontsize = 4
	fColor    = 5
	fMidHash  = 6
	fContent  = 7
	fCtime    = 8
	fWeight   = 9
	fPool     = 11
	fIDStr    = 22
)

// Decode parses a DmSegMobileReply body and returns its danmaku elements.
func Decode(b []byte) ([]Elem, error) {
	var out []Elem
	i := 0
	for i < len(b) {
		key, n := uvarint(b[i:])
		if n <= 0 {
			return out, errors.New("dmproto: bad field key")
		}
		i += n
		field := key >> 3
		wire := key & 7
		switch {
		case field == 1 && wire == 2: // repeated DanmakuElem
			length, m := uvarint(b[i:])
			if m <= 0 || i+m+int(length) > len(b) {
				return out, errors.New("dmproto: bad elem length")
			}
			i += m
			el, err := decodeElem(b[i : i+int(length)])
			if err != nil {
				return out, err
			}
			out = append(out, el)
			i += int(length)
		default:
			ni, err := skip(b, i, wire)
			if err != nil {
				return out, err
			}
			i = ni
		}
	}
	return out, nil
}

func decodeElem(b []byte) (Elem, error) {
	var e Elem
	i := 0
	for i < len(b) {
		key, n := uvarint(b[i:])
		if n <= 0 {
			return e, errors.New("dmproto: bad elem key")
		}
		i += n
		field := key >> 3
		wire := key & 7
		switch wire {
		case 0: // varint
			v, m := uvarint(b[i:])
			if m <= 0 {
				return e, errors.New("dmproto: bad varint")
			}
			i += m
			switch field {
			case fID:
				e.ID = int64(v)
			case fProgress:
				e.Progress = int32(v)
			case fMode:
				e.Mode = int32(v)
			case fFontsize:
				e.Fontsize = int32(v)
			case fColor:
				e.Color = uint32(v)
			case fCtime:
				e.Ctime = int64(v)
			case fWeight:
				e.Weight = int32(v)
			case fPool:
				e.Pool = int32(v)
			}
		case 2: // length-delimited
			length, m := uvarint(b[i:])
			if m <= 0 || i+m+int(length) > len(b) {
				return e, errors.New("dmproto: bad string length")
			}
			i += m
			s := string(b[i : i+int(length)])
			i += int(length)
			switch field {
			case fMidHash:
				e.MidHash = s
			case fContent:
				e.Content = s
			case fIDStr:
				e.IDStr = s
			}
		case 1: // 64-bit
			i += 8
		case 5: // 32-bit
			i += 4
		default:
			return e, fmt.Errorf("dmproto: unsupported wire type %d", wire)
		}
	}
	return e, nil
}

func skip(b []byte, i int, wire uint64) (int, error) {
	switch wire {
	case 0:
		_, n := uvarint(b[i:])
		if n <= 0 {
			return i, errors.New("dmproto: bad skip varint")
		}
		return i + n, nil
	case 1:
		return i + 8, nil
	case 5:
		return i + 4, nil
	case 2:
		length, n := uvarint(b[i:])
		if n <= 0 {
			return i, errors.New("dmproto: bad skip length")
		}
		return i + n + int(length), nil
	}
	return i, fmt.Errorf("dmproto: cannot skip wire type %d", wire)
}

func uvarint(b []byte) (uint64, int) {
	v, n := binary.Uvarint(b)
	return v, n
}
