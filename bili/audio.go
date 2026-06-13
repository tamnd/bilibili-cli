package bili

import (
	"context"
	"fmt"
)

// Audio fetches a music track's metadata and stat.
func (c *Client) Audio(ctx context.Context, idOrURL string) (*Audio, error) {
	id, err := c.Resolve(ctx, idOrURL)
	if err != nil {
		return nil, err
	}
	if id.Kind != KindAudio {
		return nil, fmt.Errorf("not an audio id: %s", idOrURL)
	}
	sid := itoa(id.SID)
	var info struct {
		ID        int64  `json:"id"`
		Title     string `json:"title"`
		Author    string `json:"author"`
		UID       int64  `json:"uid"`
		Uname     string `json:"uname"`
		Cover     string `json:"cover"`
		Intro     string `json:"intro"`
		Duration  int    `json:"duration"`
		Ctime     int64  `json:"ctime"`
		Statistic struct {
			Play    int64 `json:"play"`
			Comment int64 `json:"comment"`
			Collect int64 `json:"collect"`
			Share   int64 `json:"share"`
		} `json:"statistic"`
	}
	if err := c.getJSON(ctx, "https://www.bilibili.com/audio/music-service-c/web/song/info", vals("sid", sid), &info); err != nil {
		return nil, err
	}
	return &Audio{
		SID: info.ID, Title: info.Title, Author: info.Author, Uname: info.Uname, UID: info.UID,
		CoverURL: info.Cover, Intro: info.Intro, Duration: info.Duration, PlayCount: info.Statistic.Play,
		ReplyCount: info.Statistic.Comment, FavoriteCount: info.Statistic.Collect,
		ShareCount: info.Statistic.Share, Ctime: info.Ctime, FetchedAt: c.fetchedAt(),
	}, nil
}
