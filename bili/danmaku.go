package bili

import (
	"context"
	"encoding/xml"
	"fmt"
	"iter"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnd/bilibili-cli/pkg/dmproto"
)

// partCID returns the cid for a given 1-based part index of a video.
func (c *Client) partCID(ctx context.Context, idOrURL string, part int) (aid, cid int64, dur int, err error) {
	res, err := c.Video(ctx, idOrURL, VideoOptions{})
	if err != nil {
		return 0, 0, 0, err
	}
	v := res.Video
	if part < 1 {
		part = 1
	}
	if part > len(v.Pages) {
		if len(v.Pages) == 0 {
			return v.AID, v.CID, v.Duration, nil
		}
		return 0, 0, 0, fmt.Errorf("video has only %d part(s)", len(v.Pages))
	}
	p := v.Pages[part-1]
	return v.AID, p.CID, p.Duration, nil
}

// Danmaku streams every bullet-chat line for a video part (protobuf segments).
func (c *Client) Danmaku(ctx context.Context, idOrURL string, part int) iter.Seq2[Danmaku, error] {
	return func(yield func(Danmaku, error) bool) {
		aid, cid, dur, err := c.partCID(ctx, idOrURL, part)
		if err != nil {
			yield(Danmaku{}, err)
			return
		}
		segments := dur/360 + 1
		if segments < 1 {
			segments = 1
		}
		var all []Danmaku
		for seg := 1; seg <= segments; seg++ {
			p := vals("type", "1", "oid", itoa(cid), "pid", itoa(aid), "segment_index", strconv.Itoa(seg))
			body, err := c.rawGet(ctx, buildURL("https://api.bilibili.com/x/v2/dm/web/seg.so", p), nil)
			if err != nil {
				yield(Danmaku{}, err)
				return
			}
			elems, err := dmproto.Decode(body)
			if err != nil {
				// an empty or short segment is not fatal; stop walking
				break
			}
			if len(elems) == 0 {
				if seg > 1 {
					break
				}
				continue
			}
			for _, e := range elems {
				all = append(all, Danmaku{
					DmID: e.ID, ProgressMs: e.Progress, Mode: e.Mode, Fontsize: e.Fontsize,
					Color: e.Color, Ctime: e.Ctime, Pool: e.Pool, SenderHash: e.MidHash,
					Content: e.Content,
				})
			}
		}
		sort.Slice(all, func(i, j int) bool { return all[i].ProgressMs < all[j].ProgressMs })
		for _, d := range all {
			if !yield(d, nil) {
				return
			}
		}
	}
}

// DanmakuXML fetches the legacy XML danmaku snapshot for a video part.
func (c *Client) DanmakuXML(ctx context.Context, idOrURL string, part int) ([]Danmaku, error) {
	_, cid, _, err := c.partCID(ctx, idOrURL, part)
	if err != nil {
		return nil, err
	}
	body, err := c.rawGet(ctx, fmt.Sprintf("https://comment.bilibili.com/%d.xml", cid), nil)
	if err != nil {
		return nil, err
	}
	return parseDanmakuXML(body)
}

type xmlDoc struct {
	D []struct {
		P    string `xml:"p,attr"`
		Text string `xml:",chardata"`
	} `xml:"d"`
}

func parseDanmakuXML(body []byte) ([]Danmaku, error) {
	var doc xmlDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	out := make([]Danmaku, 0, len(doc.D))
	for _, d := range doc.D {
		f := strings.Split(d.P, ",")
		var dm Danmaku
		dm.Content = d.Text
		if len(f) >= 8 {
			if sec, err := strconv.ParseFloat(f[0], 64); err == nil {
				dm.ProgressMs = int32(sec * 1000)
			}
			dm.Mode = int32(atoi(f[1]))
			dm.Fontsize = int32(atoi(f[2]))
			dm.Color = uint32(atoi(f[3]))
			dm.Ctime = int64(atoi(f[4]))
			dm.Pool = int32(atoi(f[5]))
			dm.SenderHash = f[6]
			dm.DmID = int64(atoi(f[7]))
		}
		out = append(out, dm)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProgressMs < out[j].ProgressMs })
	return out, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
