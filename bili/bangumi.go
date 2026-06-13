package bili

import (
	"context"
	"fmt"
)

type rawSeason struct {
	SeasonID int64  `json:"season_id"`
	MediaID  int64  `json:"media_id"`
	Title    string `json:"title"`
	Cover    string `json:"cover"`
	Evaluate string `json:"evaluate"`
	Total    int    `json:"total"`
	Status   int    `json:"status"`
	Type     int    `json:"type"`
	Subtitle string `json:"subtitle"`
	Areas    []struct {
		Name string `json:"name"`
	} `json:"areas"`
	Styles  []string `json:"styles"`
	Publish struct {
		PubTimeShow string `json:"pub_time_show"`
	} `json:"publish"`
	Rating struct {
		Score float64 `json:"score"`
		Count int64   `json:"count"`
	} `json:"rating"`
	Stat struct {
		Views     int64 `json:"views"`
		Favorites int64 `json:"favorites"`
		Danmakus  int64 `json:"danmakus"`
	} `json:"stat"`
	SeasonTitle string `json:"season_title"`
	Episodes    []struct {
		ID        int64  `json:"id"`
		AID       int64  `json:"aid"`
		BVID      string `json:"bvid"`
		CID       int64  `json:"cid"`
		Title     string `json:"title"`
		LongTitle string `json:"long_title"`
		Cover     string `json:"cover"`
		Duration  int    `json:"duration"`
		PubTime   int64  `json:"pub_time"`
		BadgeInfo struct {
			Text string `json:"text"`
		} `json:"badge_info"`
	} `json:"episodes"`
}

var seasonTypeName = map[int]string{1: "番剧", 2: "电影", 3: "纪录片", 4: "国创", 5: "电视剧", 7: "综艺"}

// Bangumi fetches a season (by ss/ep/md) with all episodes.
func (c *Client) Bangumi(ctx context.Context, idOrURL string) (*Bangumi, error) {
	id, err := c.Resolve(ctx, idOrURL)
	if err != nil {
		return nil, err
	}
	p := vals()
	switch id.Kind {
	case KindBangumi:
		p.Set("season_id", itoa(id.SeasonID))
	case KindEpisode:
		p.Set("ep_id", itoa(id.EpID))
	case KindMedia:
		// resolve media id to season id first
		var mr struct {
			Media struct {
				SeasonID int64 `json:"season_id"`
			} `json:"media"`
		}
		if err := c.getJSON(ctx, "https://api.bilibili.com/pgc/review/user", vals("media_id", itoa(id.MediaID)), &mr); err != nil {
			return nil, err
		}
		p.Set("season_id", itoa(mr.Media.SeasonID))
	default:
		return nil, fmt.Errorf("not a bangumi id: %s", idOrURL)
	}
	var s rawSeason
	if err := c.getJSON(ctx, "https://api.bilibili.com/pgc/view/web/season", p, &s); err != nil {
		return nil, err
	}
	b := &Bangumi{
		SeasonID: s.SeasonID, MediaID: s.MediaID, SeasonTitle: s.SeasonTitle, Title: s.Title,
		TypeName: seasonTypeName[s.Type], TotalEp: s.Total, Status: s.Status, CoverURL: s.Cover,
		Evaluate: s.Evaluate, RatingScore: s.Rating.Score, RatingCount: s.Rating.Count,
		Styles: s.Styles, PublishText: s.Publish.PubTimeShow, StatViews: s.Stat.Views,
		StatFavorites: s.Stat.Favorites, StatDanmakus: s.Stat.Danmakus, FetchedAt: c.fetchedAt(),
	}
	if len(s.Areas) > 0 {
		b.Area = s.Areas[0].Name
	}
	for _, e := range s.Episodes {
		b.Episodes = append(b.Episodes, Episode{
			EpID: e.ID, AID: e.AID, BVID: e.BVID, CID: e.CID, Title: e.Title,
			LongTitle: e.LongTitle, CoverURL: e.Cover, Duration: e.Duration / 1000,
			PubText: fmtUnix(e.PubTime), Badge: e.BadgeInfo.Text,
		})
	}
	if b.TotalEp == 0 {
		b.TotalEp = len(b.Episodes)
	}
	return b, nil
}
