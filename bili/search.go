package bili

import (
	"context"
	"fmt"
	"iter"
)

type rawSearchType struct {
	NumResults int              `json:"numResults"`
	NumPages   int              `json:"numPages"`
	Page       int              `json:"page"`
	Result     []rawSearchEntry `json:"result"`
}

type rawSearchEntry struct {
	Type string `json:"type"`
	// video
	BVID      string `json:"bvid"`
	AID       int64  `json:"aid"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Mid       int64  `json:"mid"`
	Play      int64  `json:"play"`
	VideoRev  int64  `json:"video_review"`
	Favorites int64  `json:"favorites"`
	Review    int64  `json:"review"`
	Pubdate   int64  `json:"pubdate"`
	Duration  string `json:"duration"`
	Pic       string `json:"pic"`
	Typeid    string `json:"typeid"`
	Tag       string `json:"tag"`
	Desc      string `json:"description"`
	// user
	Uname  string `json:"uname"`
	Usign  string `json:"usign"`
	Fans   int64  `json:"fans"`
	Videos int64  `json:"videos"`
	Upic   string `json:"upic"`
	Level  int    `json:"level"`
	// bangumi/media
	SeasonID    int64  `json:"season_id"`
	MediaID     int64  `json:"media_id"`
	SeasonTitle string `json:"season_type_name"`
	Cover       string `json:"cover"`
	// live room (shares uname/title with the fields above)
	RoomID int64 `json:"roomid"`
	UID    int64 `json:"uid"`
	// article (id is the cv id)
	ID       int64 `json:"id"`
	View     int64 `json:"view"`
	Like     int64 `json:"like"`
	Reply    int64 `json:"reply"`
	Favorite int64 `json:"favorite"`
}

var searchTypeMap = map[string]string{
	"video":     "video",
	"user":      "bili_user",
	"bangumi":   "media_bangumi",
	"film":      "media_ft",
	"live_room": "live_room",
	"live":      "live_room",
	"article":   "article",
}

// Search streams search results of the requested type (default video).
func (c *Client) Search(ctx context.Context, q string, opt SearchOptions) iter.Seq2[SearchResult, error] {
	return func(yield func(SearchResult, error) bool) {
		st := opt.Type
		if st == "" || st == "all" {
			st = "video"
		}
		apiType, ok := searchTypeMap[st]
		if !ok {
			yield(SearchResult{}, fmt.Errorf("unknown search type %q", st))
			return
		}
		page := opt.Page
		if page < 1 {
			page = 1
		}
		emitted := 0
		for {
			p := vals("keyword", q, "search_type", apiType, "page", fmt.Sprint(page))
			if opt.Order != "" {
				p.Set("order", opt.Order)
			}
			if opt.Duration > 0 {
				p.Set("duration", fmt.Sprint(opt.Duration))
			}
			if opt.Tid > 0 {
				p.Set("tids", fmt.Sprint(opt.Tid))
			}
			var r rawSearchType
			if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/web-interface/wbi/search/type", p, &r); err != nil {
				yield(SearchResult{}, err)
				return
			}
			if len(r.Result) == 0 {
				return
			}
			for _, e := range r.Result {
				res := c.entryToResult(st, e)
				if !yield(res, nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			if r.NumPages > 0 && page >= r.NumPages {
				return
			}
			page++
		}
	}
}

func (c *Client) entryToResult(st string, e rawSearchEntry) SearchResult {
	switch st {
	case "user":
		return SearchResult{ResultType: "user", User: &User{
			Mid: e.Mid, Name: e.Uname, Sign: e.Usign, FaceURL: e.Upic, Level: e.Level,
			FollowerCount: e.Fans, VideoCount: e.Videos, FetchedAt: c.fetchedAt(),
		}}
	case "bangumi", "film":
		return SearchResult{ResultType: "bangumi", Bangumi: &Bangumi{
			SeasonID: e.SeasonID, MediaID: e.MediaID, Title: stripTags(e.Title),
			TypeName: e.SeasonTitle, CoverURL: e.Cover, FetchedAt: c.fetchedAt(),
		}}
	case "live_room", "live":
		return SearchResult{ResultType: "live_room", LiveRoom: &LiveRoom{
			RoomID: e.RoomID, UID: e.UID, Uname: e.Uname, Title: stripTags(e.Title),
			CoverURL: e.Cover, FetchedAt: c.fetchedAt(),
		}}
	case "article":
		return SearchResult{ResultType: "article", Article: &Article{
			CVID: e.ID, Title: stripTags(e.Title), AuthorMid: e.Mid, AuthorName: e.Author,
			ViewCount: e.View, LikeCount: e.Like, ReplyCount: e.Reply, FavoriteCount: e.Favorite,
			FetchedAt: c.fetchedAt(),
		}}
	default:
		return SearchResult{ResultType: "video", Video: &Video{
			BVID: e.BVID, AID: e.AID, Title: stripTags(e.Title), OwnerMid: e.Mid,
			OwnerName: e.Author, ViewCount: e.Play, DanmakuCount: e.VideoRev,
			FavoriteCount: e.Favorites, ReplyCount: e.Review, Pubdate: e.Pubdate,
			PubdateText: fmtUnix(e.Pubdate), CoverURL: fixProto(e.Pic), Description: e.Desc,
			URL:  "https://www.bilibili.com/video/" + e.BVID,
			Tags: splitTag(e.Tag), FetchedAt: c.fetchedAt(),
		}}
	}
}

func fixProto(s string) string {
	if len(s) > 2 && s[0] == '/' && s[1] == '/' {
		return "https:" + s
	}
	return s
}

func splitTag(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, t := range splitComma(s) {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Suggest returns autosuggest terms for a query.
func (c *Client) Suggest(ctx context.Context, term string) ([]string, error) {
	body, err := c.rawGet(ctx, buildURL("https://s.search.bilibili.com/main/suggest", vals("term", term, "main_ver", "v1")), nil)
	if err != nil {
		return nil, err
	}
	return parseSuggest(body)
}
