package bili

import (
	"context"
	"fmt"
	"iter"
)

// Favorites lists a user's created favorite folders.
func (c *Client) Favorites(ctx context.Context, mid string) ([]Favorite, error) {
	var r struct {
		Count int `json:"count"`
		List  []struct {
			ID         int64  `json:"id"`
			FID        int64  `json:"fid"`
			Title      string `json:"title"`
			MediaCount int    `json:"media_count"`
		} `json:"list"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/v3/fav/folder/created/list-all", vals("up_mid", mid), &r); err != nil {
		return nil, err
	}
	out := make([]Favorite, 0, len(r.List))
	for _, f := range r.List {
		out = append(out, Favorite{
			MediaID: f.ID, FID: f.FID, Title: f.Title, MediaCount: f.MediaCount,
			FetchedAt: c.fetchedAt(),
		})
	}
	return out, nil
}

// FavoriteItems streams the videos in a favorite folder.
func (c *Client) FavoriteItems(ctx context.Context, idOrURL string, opt ListOptions) iter.Seq2[Video, error] {
	return func(yield func(Video, error) bool) {
		id, err := c.Resolve(ctx, idOrURL)
		if err != nil {
			yield(Video{}, err)
			return
		}
		if id.Kind != KindFavorite {
			yield(Video{}, fmt.Errorf("not a favorite id: %s", idOrURL))
			return
		}
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
			p := vals("media_id", itoa(id.FavID), "pn", fmt.Sprint(page), "ps", fmt.Sprint(ps))
			var r struct {
				Medias []struct {
					BVID     string `json:"bvid"`
					ID       int64  `json:"id"`
					Title    string `json:"title"`
					Cover    string `json:"cover"`
					Intro    string `json:"intro"`
					Page     int    `json:"page"`
					Duration int    `json:"duration"`
					Upper    struct {
						Mid  int64  `json:"mid"`
						Name string `json:"name"`
					} `json:"upper"`
					CntInfo struct {
						Play    int64 `json:"play"`
						Danmaku int64 `json:"danmaku"`
						Collect int64 `json:"collect"`
					} `json:"cnt_info"`
					Pubtime int64 `json:"pubtime"`
				} `json:"medias"`
				HasMore bool `json:"has_more"`
			}
			if err := c.getJSON(ctx, "https://api.bilibili.com/x/v3/fav/resource/list", p, &r); err != nil {
				yield(Video{}, err)
				return
			}
			if len(r.Medias) == 0 {
				return
			}
			for _, m := range r.Medias {
				rec := Video{
					BVID: m.BVID, AID: m.ID, Title: m.Title, Description: m.Intro,
					OwnerMid: m.Upper.Mid, OwnerName: m.Upper.Name, ViewCount: m.CntInfo.Play,
					DanmakuCount: m.CntInfo.Danmaku, FavoriteCount: m.CntInfo.Collect,
					Duration: m.Duration, Parts: m.Page, Pubdate: m.Pubtime,
					PubdateText: fmtUnix(m.Pubtime), CoverURL: m.Cover,
					URL:       "https://www.bilibili.com/video/" + m.BVID,
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
			if !r.HasMore {
				return
			}
			page++
		}
	}
}
