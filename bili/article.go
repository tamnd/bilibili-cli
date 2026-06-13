package bili

import (
	"context"
	"fmt"
)

// Article fetches a column article. When withText is set the plain-text body is
// extracted and attached.
func (c *Client) Article(ctx context.Context, idOrURL string, withText bool) (*Article, error) {
	id, err := c.Resolve(ctx, idOrURL)
	if err != nil {
		return nil, err
	}
	if id.Kind != KindArticle {
		return nil, fmt.Errorf("not an article id: %s", idOrURL)
	}
	cvid := itoa(id.CVID)
	var v struct {
		Title  string `json:"title"`
		Author struct {
			Mid  int64  `json:"mid"`
			Name string `json:"name"`
		} `json:"author"`
		BannerURL   string `json:"banner_url"`
		Summary     string `json:"summary"`
		Words       int    `json:"words"`
		PublishTime int64  `json:"publish_time"`
		Category    struct {
			Name string `json:"name"`
		} `json:"category"`
		Stats struct {
			View     int64 `json:"view"`
			Favorite int64 `json:"favorite"`
			Like     int64 `json:"like"`
			Reply    int64 `json:"reply"`
			Coin     int64 `json:"coin"`
		} `json:"stats"`
	}
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/article/view", vals("id", cvid), &v); err != nil {
		return nil, err
	}
	a := &Article{
		CVID: id.CVID, Title: v.Title, AuthorMid: v.Author.Mid, AuthorName: v.Author.Name,
		Summary: v.Summary, BannerURL: v.BannerURL, Words: v.Words, ViewCount: v.Stats.View,
		LikeCount: v.Stats.Like, ReplyCount: v.Stats.Reply, FavoriteCount: v.Stats.Favorite,
		CoinCount: v.Stats.Coin, PublishTime: v.PublishTime, FetchedAt: c.fetchedAt(),
	}
	a.CategoryName = v.Category.Name
	if withText {
		body, err := c.rawGet(ctx, fmt.Sprintf("https://www.bilibili.com/read/cv%d", id.CVID), nil)
		if err == nil {
			a.ContentText = extractArticleText(body)
		}
	}
	return a, nil
}
