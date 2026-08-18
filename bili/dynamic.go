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

// dynFeedBase is the space feed endpoint, named once because the pagination
// rule below builds errors that have to point at the same URL the request went
// to.
const dynFeedBase = "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space"

// endOfFeed reports whether a page carrying no items is the feed ending.
//
// The item count is not the signal. This endpoint says has_more, and reading
// the count instead means a refused page and an exhausted feed are the same
// event: both arrive as zero items, and stopping quietly on either one turns a
// refusal into a short list that looks complete. So the count only ends the
// loop when the server also said there is nothing after it, and only on a page
// that already produced items. A first page carrying nothing has not paginated
// anywhere, and this feed refuses anonymous callers far more often than a
// creator posts nothing at all.
func endOfFeed(page int, hasMore bool) bool { return page > 0 && !hasMore }

// emptyFeedError is the refusal a page of no items amounts to.
func emptyFeedError(hasMore bool) *APIError {
	e := &APIError{
		Message:  "the feed returned no items",
		Status:   StatusRefusedSilent,
		Endpoint: dynFeedBase,
	}
	if hasMore {
		e.Hint = "the feed said there was more and then sent none, so this is a refusal rather than the end of the list. Supply a logged-in cookie via --cookie or BILI_COOKIE and read it again"
	} else {
		e.Hint = "the first page of this feed carried nothing. A creator who has never posted would look the same, but this endpoint refuses anonymous callers far more often than that, so supply a logged-in cookie via --cookie or BILI_COOKIE before believing the feed is empty"
	}
	return e
}

// Dynamics streams a user's dynamics feed.
func (c *Client) Dynamics(ctx context.Context, mid string, opt ListOptions) iter.Seq2[Dynamic, error] {
	return func(yield func(Dynamic, error) bool) {
		offset := ""
		emitted := 0
		for page := 0; ; page++ {
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
			if err := c.getJSONSigned(ctx, dynFeedBase, p, &r); err != nil {
				yield(Dynamic{}, err)
				return
			}
			if len(r.Items) == 0 {
				if !endOfFeed(page, r.HasMore) {
					yield(Dynamic{}, emptyFeedError(r.HasMore))
				}
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
