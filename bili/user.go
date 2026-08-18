package bili

import (
	"context"
	"fmt"
	"iter"
)

// The four endpoints a user record is assembled from. They have four different
// gates, and which of them answered is the difference between a count that is
// zero and a count that is unknown.
const (
	accInfoBase      = "https://api.bilibili.com/x/space/wbi/acc/info"
	arcSearchBase    = "https://api.bilibili.com/x/space/wbi/arc/search"
	relationStatBase = "https://api.bilibili.com/x/relation/stat"
	upstatBase       = "https://api.bilibili.com/x/space/upstat"
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
//
// It is four requests behind one record. Only the first is required: acc/info
// carries the identity, and without it there is no user to talk about. The
// other three carry counts, they have their own gates, and any of them can
// refuse while the other two answer. Refusing is not the same as counting zero,
// so a count from an endpoint that said no is left out of the record entirely
// and named in the envelope's missed map with the reason.
func (c *Client) User(ctx context.Context, mid string) (*User, error) {
	var info rawAccInfo
	env, err := c.getJSONSignedEnv(ctx, accInfoBase, addDeviceParams(vals("mid", mid)), &info)
	if err != nil {
		return nil, err
	}
	u := &User{
		Mid: info.Mid, Name: info.Name, Sex: info.Sex, FaceURL: info.Face, Sign: info.Sign,
		Level: info.Level, TopPhotoURL: info.TopPhoto, OfficialRole: info.Official.Role,
		OfficialTitle: info.Official.Title, VipType: info.Vip.Type, VipStatus: info.Vip.Status,
		Birthday: info.Birthday, School: info.School.Name, FetchedAt: c.fetchedAt(),
		Envelope: env,
	}
	var rel struct {
		Follower  int64 `json:"follower"`
		Following int64 `json:"following"`
	}
	if err := c.getJSON(ctx, relationStatBase, vals("vmid", mid), &rel); err != nil {
		env.miss("follower_count", refusalNote(err))
		env.miss("following_count", refusalNote(err))
	} else {
		u.FollowerCount, u.FollowingCount = &rel.Follower, &rel.Following
	}

	var up struct {
		Archive struct {
			View int64 `json:"view"`
		} `json:"archive"`
		Likes int64 `json:"likes"`
	}
	if err := c.getJSON(ctx, upstatBase, vals("mid", mid), &up); err != nil {
		env.miss("total_view", refusalNote(err))
		env.miss("total_like", refusalNote(err))
	} else {
		u.TotalView, u.TotalLike = &up.Archive.View, &up.Likes
	}

	// The upload count is on none of the three endpoints above. The listing
	// endpoint knows it, because it has to paginate by it, so the count comes
	// from asking for the shortest page it will serve and reading the total off
	// the response rather than off the items.
	if n, err := c.userVideoCount(ctx, mid); err != nil {
		env.miss("video_count", refusalNote(err))
	} else {
		u.VideoCount = &n
	}
	return u, nil
}

// userVideoCount reads a creator's total upload count from the listing
// endpoint's own pagination total.
func (c *Client) userVideoCount(ctx context.Context, mid string) (int64, error) {
	p := addDeviceParams(vals("mid", mid, "pn", "1", "ps", "1", "order", "pubdate"))
	var r rawArcSearch
	if err := c.getJSONSigned(ctx, arcSearchBase, p, &r); err != nil {
		return 0, err
	}
	return int64(r.Page.Count), nil
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
			env, err := c.getJSONSignedEnv(ctx, arcSearchBase, p, &r)
			if err != nil {
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
					Envelope:  env,
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
