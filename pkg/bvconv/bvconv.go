// Package bvconv converts between Bilibili's BV and AV video identifiers.
//
// BVID is the canonical encoded form (for example BV1GJ411x7h7); AVID is the
// legacy numeric form (for example av170001). The two are losslessly
// interconvertible with two positional swaps, a base58 alphabet, an XOR mask,
// and a 51-bit modulus. This mirrors the current published algorithm.
package bvconv

import (
	"errors"
	"strings"
)

const (
	xorCode  = 23442827791579
	maskCode = (1 << 51) - 1
	maxAid   = 1 << 51
	base     = 58
	alpha    = "FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf"
)

// ToAV converts a BVID to its numeric AVID. The bvid must be the 12-character
// form, with or without the "BV" prefix casing.
func ToAV(bvid string) (int64, error) {
	if len(bvid) != 12 {
		return 0, errors.New("bvconv: bvid must be 12 characters")
	}
	b := []byte(bvid)
	b[3], b[9] = b[9], b[3]
	b[4], b[7] = b[7], b[4]
	var tmp int64
	for _, c := range b[3:] {
		i := strings.IndexByte(alpha, c)
		if i < 0 {
			return 0, errors.New("bvconv: invalid character in bvid")
		}
		tmp = tmp*base + int64(i)
	}
	return (tmp & maskCode) ^ xorCode, nil
}

// ToBV converts a numeric AVID to its 12-character BVID (with the BV1 prefix).
func ToBV(aid int64) string {
	b := []byte("BV1000000000")
	idx := len(b) - 1
	tmp := (maxAid | aid) ^ xorCode
	for tmp > 0 {
		b[idx] = alpha[tmp%base]
		tmp /= base
		idx--
	}
	b[3], b[9] = b[9], b[3]
	b[4], b[7] = b[7], b[4]
	return string(b)
}
