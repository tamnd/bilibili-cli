package bili

import (
	"context"
	"fmt"
	"iter"
)

type rawAccInfo struct {
	Mid      int64  `json:"mid"`
	Name     string `json:"name"`
	Sex      string `json:"sex"`
	Face     string `json:"face"`
	Sign     string `json:"sign"`
	Level    int    `json:"level"`
	Birthday string `json:"birthday"`
	TopPhoto string `json:"top_photo"`
	Official struct {
		Role  int    `json:"role"`
		Title string `json:"title"`
	} `json:"official"`
	Vip struct {
		Type   int `json:"type"`
		Status int `json:"status"`
	} `json:"vip"`
	School struct {
		Name string `json:"name"`
	} `json:"school"`
}

// User fetches a creator's profile and stat.
func (c *Client) User(ctx context.Context, mid string) (*User, error) {
	var info rawAccInfo
	if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/space/wbi/acc/info", addDeviceParams(vals("mid", mid)), &info); err != nil {
		return nil, err
	}
	u := &User{
		Mid: info.Mid, Name: info.Name, Sex: info.Sex, FaceURL: info.Face, Sign: info.Sign,
		Level: info.Level, TopPhotoURL: info.TopPhoto, OfficialRole: info.Official.Role,
		OfficialTitle: info.Official.Title, VipType: info.Vip.Type, VipStatus: info.Vip.Status,
		Birthday: info.Birthday, School: info.School.Name, FetchedAt: c.fetchedAt(),
	}
	// relation + upstat are best-effort; do not fail the whole call if missing
	var rel struct {
		Follower  int64 `json:"follower"`
		Following int64 `json:"following"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/relation/stat", vals("vmid", mid), &rel); err == nil {
		u.FollowerCount, u.FollowingCount = rel.Follower, rel.Following
	}
	var up struct {
		Archive struct {
			View int64 `json:"view"`
		} `json:"archive"`
		Likes int64 `json:"likes"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/space/upstat", vals("mid", mid), &up); err == nil {
		u.TotalView, u.TotalLike = up.Archive.View, up.Likes
	}
	return u, nil
}

type rawArcSearch struct {
	List struct {
		Vlist []struct {
			BVID     string `json:"bvid"`
			AID      int64  `json:"aid"`
			Title    string `json:"title"`
			Author   string `json:"author"`
			Mid      int64  `json:"mid"`
			Created  int64  `json:"created"`
			Length   string `json:"length"`
			Play     int64  `json:"play"`
			Comment  int64  `json:"comment"`
			VideoRev int64  `json:"video_review"`
			Pic      string `json:"pic"`
			TypeID   int    `json:"typeid"`
			Desc     string `json:"description"`
		} `json:"vlist"`
	} `json:"list"`
	Page struct {
		Count int `json:"count"`
		PN    int `json:"pn"`
		PS    int `json:"ps"`
	} `json:"page"`
}

// UserVideos streams a creator's uploaded videos.
func (c *Client) UserVideos(ctx context.Context, mid string, opt ListOptions) iter.Seq2[Video, error] {
	return func(yield func(Video, error) bool) {
		page := opt.Page
		if page < 1 {
			page = 1
		}
		ps := opt.PageSize
		if ps <= 0 {
			ps = 30
		}
		order := opt.Order
		if order == "" {
			order = "pubdate"
		}
		emitted := 0
		for {
			p := vals("mid", mid, "pn", fmt.Sprint(page), "ps", fmt.Sprint(ps), "order", order)
			if opt.Keyword != "" {
				p.Set("keyword", opt.Keyword)
			}
			p = addDeviceParams(p)
			var r rawArcSearch
			if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/space/wbi/arc/search", p, &r); err != nil {
				yield(Video{}, err)
				return
			}
			if len(r.List.Vlist) == 0 {
				return
			}
			for _, v := range r.List.Vlist {
				rec := Video{
					BVID: v.BVID, AID: v.AID, Title: stripTags(v.Title), OwnerMid: v.Mid,
					OwnerName: v.Author, ViewCount: v.Play, ReplyCount: v.Comment,
					DanmakuCount: v.VideoRev, Pubdate: v.Created, PubdateText: fmtUnix(v.Created),
					CoverURL: v.Pic, TypeID: v.TypeID, Description: v.Desc,
					URL:       "https://www.bilibili.com/video/" + v.BVID,
					FetchedAt: c.fetchedAt(),
				}
				if !yield(rec, nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			if page*ps >= r.Page.Count {
				return
			}
			page++
		}
	}
}
