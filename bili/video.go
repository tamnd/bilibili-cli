package bili

import (
	"context"
	"fmt"
)

// raw structures for /x/web-interface/view/detail
type rawView struct {
	BVID      string `json:"bvid"`
	AID       int64  `json:"aid"`
	Videos    int    `json:"videos"`
	TID       int    `json:"tid"`
	Tname     string `json:"tname"`
	Pic       string `json:"pic"`
	Title     string `json:"title"`
	Pubdate   int64  `json:"pubdate"`
	Ctime     int64  `json:"ctime"`
	Desc      string `json:"desc"`
	Duration  int    `json:"duration"`
	Copyright int    `json:"copyright"`
	State     int    `json:"state"`
	CID       int64  `json:"cid"`
	ShortLink string `json:"short_link_v2"`
	Dimension struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimension"`
	Owner struct {
		Mid  int64  `json:"mid"`
		Name string `json:"name"`
		Face string `json:"face"`
	} `json:"owner"`
	Stat struct {
		View     int64 `json:"view"`
		Danmaku  int64 `json:"danmaku"`
		Reply    int64 `json:"reply"`
		Favorite int64 `json:"favorite"`
		Coin     int64 `json:"coin"`
		Share    int64 `json:"share"`
		Like     int64 `json:"like"`
		NowRank  int   `json:"now_rank"`
		HisRank  int   `json:"his_rank"`
	} `json:"stat"`
	Pages []struct {
		CID       int64  `json:"cid"`
		Page      int    `json:"page"`
		Part      string `json:"part"`
		Duration  int    `json:"duration"`
		Dimension struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"dimension"`
		FirstFrame string `json:"first_frame"`
	} `json:"pages"`
}

func (c *Client) videoToRecord(v rawView) Video {
	out := Video{
		BVID: v.BVID, AID: v.AID, CID: v.CID, Title: v.Title, Description: v.Desc,
		OwnerMid: v.Owner.Mid, OwnerName: v.Owner.Name, TypeID: v.TID, TypeName: v.Tname,
		Duration: v.Duration, ViewCount: v.Stat.View, DanmakuCount: v.Stat.Danmaku,
		ReplyCount: v.Stat.Reply, FavoriteCount: v.Stat.Favorite, CoinCount: v.Stat.Coin,
		ShareCount: v.Stat.Share, LikeCount: v.Stat.Like, NowRank: v.Stat.NowRank,
		HisRank: v.Stat.HisRank, Pubdate: v.Pubdate, PubdateText: fmtUnix(v.Pubdate),
		Ctime: v.Ctime, Parts: v.Videos, Width: v.Dimension.Width, Height: v.Dimension.Height,
		Copyright: v.Copyright, CoverURL: v.Pic, ShortLink: v.ShortLink, State: v.State,
		URL:       "https://www.bilibili.com/video/" + v.BVID,
		FetchedAt: c.fetchedAt(),
	}
	for _, p := range v.Pages {
		out.Pages = append(out.Pages, Page{
			CID: p.CID, Page: p.Page, PartTitle: p.Part, Duration: p.Duration,
			Width: p.Dimension.Width, Height: p.Dimension.Height, FirstFrame: p.FirstFrame,
		})
	}
	return out
}

// Video resolves an id/url to a full video record.
//
// It uses the plain /x/web-interface/view endpoint for the core record (the
// combined .../view/detail endpoint is risk-controlled for anonymous callers and
// returns HTTP 412), then enriches with tags and, when asked, related videos via
// their own endpoints.
func (c *Client) Video(ctx context.Context, idOrURL string, opt VideoOptions) (*VideoResult, error) {
	id, err := c.Resolve(ctx, idOrURL)
	if err != nil {
		return nil, err
	}
	if id.Kind != KindVideo {
		return nil, fmt.Errorf("not a video: %s", idOrURL)
	}
	p := vals("bvid", id.BVID)
	var v rawView
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/web-interface/view", p, &v); err != nil {
		return nil, err
	}
	rec := c.videoToRecord(v)
	rec.Tags = c.videoTags(ctx, id.BVID)

	res := &VideoResult{Video: rec}
	if opt.Related {
		if rel, err := c.Related(ctx, id.BVID); err == nil {
			res.Related = rel
		}
	}
	return res, nil
}

// videoTags fetches the tag names for a video. Tag failures are non-fatal: the
// core record is still returned without tags.
func (c *Client) videoTags(ctx context.Context, bvid string) []string {
	var tags []struct {
		TagName string `json:"tag_name"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/tag/archive/tags", vals("bvid", bvid), &tags); err != nil {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t.TagName != "" {
			out = append(out, t.TagName)
		}
	}
	return out
}

// VideoResult bundles a video with optional related videos.
type VideoResult struct {
	Video   Video
	Related []Video
}

// Pages returns the part list of a video.
func (c *Client) Pages(ctx context.Context, idOrURL string) ([]Page, error) {
	res, err := c.Video(ctx, idOrURL, VideoOptions{})
	if err != nil {
		return nil, err
	}
	return res.Video.Pages, nil
}

// Related returns related videos for a video.
func (c *Client) Related(ctx context.Context, idOrURL string) ([]Video, error) {
	id, err := c.Resolve(ctx, idOrURL)
	if err != nil {
		return nil, err
	}
	var rel []rawView
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/web-interface/archive/related", vals("bvid", id.BVID), &rel); err != nil {
		return nil, err
	}
	out := make([]Video, 0, len(rel))
	for _, r := range rel {
		out = append(out, c.videoToRecord(r))
	}
	return out, nil
}
