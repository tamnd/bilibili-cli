package bili

import (
	"context"
	"fmt"
	"iter"
)

type rawPopItem struct {
	BVID     string `json:"bvid"`
	AID      int64  `json:"aid"`
	CID      int64  `json:"cid"`
	Title    string `json:"title"`
	Desc     string `json:"desc"`
	Pic      string `json:"pic"`
	Tname    string `json:"tname"`
	Tid      int    `json:"tid"`
	Pubdate  int64  `json:"pubdate"`
	Duration int    `json:"duration"`
	Owner    struct {
		Mid  int64  `json:"mid"`
		Name string `json:"name"`
	} `json:"owner"`
	Stat struct {
		View     int64 `json:"view"`
		Danmaku  int64 `json:"danmaku"`
		Reply    int64 `json:"reply"`
		Favorite int64 `json:"favorite"`
		Coin     int64 `json:"coin"`
		Share    int64 `json:"share"`
		Like     int64 `json:"like"`
	} `json:"stat"`
}

func (c *Client) popToRecord(p rawPopItem) Video {
	return Video{
		BVID: p.BVID, AID: p.AID, CID: p.CID, Title: p.Title, Description: p.Desc,
		OwnerMid: p.Owner.Mid, OwnerName: p.Owner.Name, TypeID: p.Tid, TypeName: p.Tname,
		Duration: p.Duration, ViewCount: p.Stat.View, DanmakuCount: p.Stat.Danmaku,
		ReplyCount: p.Stat.Reply, FavoriteCount: p.Stat.Favorite, CoinCount: p.Stat.Coin,
		ShareCount: p.Stat.Share, LikeCount: p.Stat.Like, Pubdate: p.Pubdate,
		PubdateText: fmtUnix(p.Pubdate), CoverURL: p.Pic,
		URL:       "https://www.bilibili.com/video/" + p.BVID,
		FetchedAt: c.fetchedAt(),
	}
}

// Popular streams the "popular" feed.
func (c *Client) Popular(ctx context.Context, opt ListOptions) iter.Seq2[Video, error] {
	return func(yield func(Video, error) bool) {
		page := opt.Page
		if page < 1 {
			page = 1
		}
		ps := opt.PageSize
		if ps <= 0 {
			ps = 20
		}
		emitted := 0
		for {
			var r struct {
				List   []rawPopItem `json:"list"`
				NoMore bool         `json:"no_more"`
			}
			if err := c.getJSON(ctx, "https://api.bilibili.com/x/web-interface/popular", vals("pn", fmt.Sprint(page), "ps", fmt.Sprint(ps)), &r); err != nil {
				yield(Video{}, err)
				return
			}
			if len(r.List) == 0 {
				return
			}
			for _, it := range r.List {
				if !yield(c.popToRecord(it), nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			if r.NoMore {
				return
			}
			page++
		}
	}
}

// Weekly returns one issue of the "每周必看" weekly selection.
func (c *Client) Weekly(ctx context.Context, number int) ([]Video, error) {
	var r struct {
		List []rawPopItem `json:"list"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/web-interface/popular/series/one", vals("number", fmt.Sprint(number)), &r); err != nil {
		return nil, err
	}
	out := make([]Video, 0, len(r.List))
	for _, it := range r.List {
		out = append(out, c.popToRecord(it))
	}
	return out, nil
}

// Ranking returns the leaderboard, optionally for one partition (tid/rid).
func (c *Client) Ranking(ctx context.Context, tid int) ([]Video, error) {
	p := vals("type", "all")
	if tid > 0 {
		p.Set("rid", fmt.Sprint(tid))
	}
	var r struct {
		List []rawPopItem `json:"list"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/web-interface/ranking/v2", p, &r); err != nil {
		return nil, err
	}
	out := make([]Video, 0, len(r.List))
	for _, it := range r.List {
		out = append(out, c.popToRecord(it))
	}
	return out, nil
}

// Trending returns the current hot search terms.
func (c *Client) Trending(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	var r struct {
		Trending struct {
			List []struct {
				Keyword  string `json:"keyword"`
				ShowName string `json:"show_name"`
			} `json:"list"`
		} `json:"trending"`
	}
	if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/web-interface/wbi/search/square", vals("limit", fmt.Sprint(limit)), &r); err != nil {
		// fall back to the unsigned square endpoint
		if err2 := c.getJSON(ctx, "https://api.bilibili.com/x/web-interface/search/square", vals("limit", fmt.Sprint(limit)), &r); err2 != nil {
			return nil, err
		}
	}
	var out []string
	for _, t := range r.Trending.List {
		s := t.ShowName
		if s == "" {
			s = t.Keyword
		}
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}
