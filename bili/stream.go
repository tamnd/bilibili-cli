package bili

import (
	"context"
	"fmt"
)

var qualityName = map[int]string{
	6: "240P", 16: "360P", 32: "480P", 64: "720P", 74: "720P60", 80: "1080P",
	112: "1080P+", 116: "1080P60", 120: "4K", 125: "HDR", 126: "Dolby", 127: "8K",
}

// Streams returns the available playable qualities for a video part.
// fnval 4048 requests the DASH set (video+audio streams). Anonymous callers get
// up to 480/720p; SESSDATA raises the ceiling to the account entitlement.
func (c *Client) Streams(ctx context.Context, idOrURL string, part, qn int) ([]Stream, error) {
	res, err := c.Video(ctx, idOrURL, VideoOptions{})
	if err != nil {
		return nil, err
	}
	v := res.Video
	cid := v.CID
	if part >= 1 && part <= len(v.Pages) {
		cid = v.Pages[part-1].CID
	}
	if qn == 0 {
		qn = 80
	}
	p := vals("bvid", v.BVID, "cid", itoa(cid), "qn", fmt.Sprint(qn), "fnval", "4048", "fourk", "1")
	var r struct {
		Quality int   `json:"quality"`
		Timelen int64 `json:"timelength"`
		Dash    struct {
			Video []rawDashStream `json:"video"`
			Audio []rawDashStream `json:"audio"`
		} `json:"dash"`
		Durl []struct {
			URL    string   `json:"url"`
			Backup []string `json:"backup_url"`
			Length int64    `json:"length"`
		} `json:"durl"`
	}
	if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/player/wbi/playurl", p, &r); err != nil {
		return nil, err
	}
	var out []Stream
	for _, d := range r.Dash.Video {
		out = append(out, dashToStream(d, "video", r.Timelen))
	}
	for _, d := range r.Dash.Audio {
		out = append(out, dashToStream(d, "audio", r.Timelen))
	}
	if len(out) == 0 {
		for _, d := range r.Durl {
			out = append(out, Stream{
				Quality: r.Quality, QualityText: qualityName[r.Quality], MIME: "video/flv",
				URL: d.URL, BackupURLs: d.Backup, DurationMs: r.Timelen,
			})
		}
	}
	return out, nil
}

type rawDashStream struct {
	ID        int      `json:"id"`
	BaseURL   string   `json:"baseUrl"`
	Backup    []string `json:"backupUrl"`
	Bandwidth int64    `json:"bandwidth"`
	MimeType  string   `json:"mimeType"`
	Codecs    string   `json:"codecs"`
	Width     int      `json:"width"`
	Height    int      `json:"height"`
	FrameRate string   `json:"frameRate"`
}

func dashToStream(d rawDashStream, kind string, dur int64) Stream {
	return Stream{
		Quality: d.ID, QualityText: qualityName[d.ID], Codecs: d.Codecs, MIME: d.MimeType,
		Bandwidth: d.Bandwidth, Width: d.Width, Height: d.Height, FrameRate: d.FrameRate,
		URL: d.BaseURL, BackupURLs: d.Backup, DurationMs: dur,
	}
}
