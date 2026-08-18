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

// The two danmaku surfaces. Neither is JSON, so both build their envelope by
// hand rather than getting one back from getJSONEnv.
const (
	dmSegBase = "https://api.bilibili.com/x/v2/dm/web/seg.so"
	dmXMLHost = "https://comment.bilibili.com"
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
			res, err := c.fetch(ctx, buildURL(dmSegBase, p), nil)
			if err != nil {
				yield(Danmaku{}, err)
				return
			}
			// This surface is protobuf, so it gets classified before decoding
			// too. The case that matters is an HTML body: risk control serves
			// the same interstitial here as anywhere else, and a protobuf
			// decoder meeting it reports a malformed varint, which sends the
			// reader looking for a bug in the decoder.
			//
			// The empty body rule applies to the first segment only. A video
			// has as many segments as it has six minute blocks, and the last
			// one is routinely short or absent, so an empty segment three is
			// the end of the stream rather than a refusal. An empty segment
			// one is a refusal.
			st, apiErr := classifyBinary(res)
			if apiErr != nil && (seg == 1 || st != StatusRefusedSilent) {
				yield(Danmaku{}, apiErr)
				return
			}
			env := &Envelope{Endpoint: endpointName(dmSegBase), Status: st, Fetched: c.fetchedAt(), Bytes: len(res.body)}
			elems, err := dmproto.Decode(res.body)
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
					Content: e.Content, Envelope: env,
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
	url := fmt.Sprintf("%s/%d.xml", dmXMLHost, cid)
	body, err := c.rawGet(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	// rawGet has already turned a non-2xx into an error, so anything that got
	// this far is an answer whatever the XML turns out to hold.
	env := &Envelope{Endpoint: endpointName(url), Status: StatusOK, Fetched: c.fetchedAt(), Bytes: len(body)}
	return parseDanmakuXML(body, env)
}

type xmlDoc struct {
	D []struct {
		P    string `xml:"p,attr"`
		Text string `xml:",chardata"`
	} `xml:"d"`
}

func parseDanmakuXML(body []byte, env *Envelope) ([]Danmaku, error) {
	var doc xmlDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	out := make([]Danmaku, 0, len(doc.D))
	for _, d := range doc.D {
		f := strings.Split(d.P, ",")
		var dm Danmaku
		dm.Content = d.Text
		dm.Envelope = env
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
