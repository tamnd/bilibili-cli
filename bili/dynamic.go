package bili

import (
	"context"
	"fmt"
	"iter"
)

// rawDynItem is a subset of a web-dynamic feed item.
type rawDynItem struct {
	IDStr   string `json:"id_str"`
	Type    string `json:"type"`
	Modules struct {
		Author struct {
			Mid     int64     `json:"mid"`
			Name    string    `json:"name"`
			PubTs   flexInt64 `json:"pub_ts"`
			PubText string    `json:"pub_time"`
		} `json:"module_author"`
		Dynamic struct {
			Desc *struct {
				Text string `json:"text"`
			} `json:"desc"`
			Major *struct {
				Type    string `json:"type"`
				Archive *struct {
					BVID  string `json:"bvid"`
					Title string `json:"title"`
					Desc  string `json:"desc"`
				} `json:"archive"`
				Draw *struct {
					Items []struct {
						Src string `json:"src"`
					} `json:"items"`
				} `json:"draw"`
			} `json:"major"`
		} `json:"module_dynamic"`
		Stat struct {
			Like struct {
				Count int64 `json:"count"`
			} `json:"like"`
			Comment struct {
				Count int64 `json:"count"`
			} `json:"comment"`
			Forward struct {
				Count int64 `json:"count"`
			} `json:"forward"`
		} `json:"module_stat"`
	} `json:"modules"`
	Orig *rawDynItem `json:"orig"`
}

func (c *Client) dynToRecord(it rawDynItem) Dynamic {
	d := Dynamic{
		ID: it.IDStr, Type: it.Type, AuthorMid: it.Modules.Author.Mid,
		AuthorName: it.Modules.Author.Name, PubTs: int64(it.Modules.Author.PubTs),
		PubText: it.Modules.Author.PubText, StatLike: it.Modules.Stat.Like.Count,
		StatReply: it.Modules.Stat.Comment.Count, StatForward: it.Modules.Stat.Forward.Count,
		FetchedAt: c.fetchedAt(),
	}
	if it.Modules.Dynamic.Desc != nil {
		d.Text = it.Modules.Dynamic.Desc.Text
	}
	if m := it.Modules.Dynamic.Major; m != nil {
		if m.Archive != nil {
			d.VideoBVID = m.Archive.BVID
			if d.Text == "" {
				d.Text = m.Archive.Title
			}
		}
		if m.Draw != nil {
			for _, im := range m.Draw.Items {
				d.Pics = append(d.Pics, im.Src)
			}
		}
	}
	if it.Orig != nil {
		d.OrigID = it.Orig.IDStr
	}
	return d
}

// Dynamic fetches one dynamic post.
func (c *Client) Dynamic(ctx context.Context, id string) (*Dynamic, error) {
	var r struct {
		Item rawDynItem `json:"item"`
	}
	p := vals("id", id, "features", "itemOpusStyle", "timezone_offset", "-480",
		"platform", "web", "gaia_source", "main_web", "web_location", "333.1330")
	if err := c.getJSON(ctx, "https://api.bilibili.com/x/polymer/web-dynamic/v1/detail", p, &r); err != nil {
		return nil, err
	}
	d := c.dynToRecord(r.Item)
	return &d, nil
}

// Dynamics streams a user's dynamics feed.
func (c *Client) Dynamics(ctx context.Context, mid string, opt ListOptions) iter.Seq2[Dynamic, error] {
	return func(yield func(Dynamic, error) bool) {
		offset := ""
		emitted := 0
		for {
			p := vals("host_mid", mid, "features", "itemOpusStyle", "platform", "web", "web_location", "333.999")
			if offset != "" {
				p.Set("offset", offset)
			}
			p = addDeviceParams(p)
			var r struct {
				HasMore bool         `json:"has_more"`
				Offset  string       `json:"offset"`
				Items   []rawDynItem `json:"items"`
			}
			if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space", p, &r); err != nil {
				yield(Dynamic{}, err)
				return
			}
			if len(r.Items) == 0 {
				return
			}
			for _, it := range r.Items {
				if !yield(c.dynToRecord(it), nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			if !r.HasMore || r.Offset == "" {
				return
			}
			offset = r.Offset
		}
	}
}

var _ = fmt.Sprint
