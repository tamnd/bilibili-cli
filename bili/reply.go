package bili

import (
	"context"
	"fmt"
	"iter"
)

// commentTarget resolves an id/url to the (oid, type) pair the reply API needs.
func (c *Client) commentTarget(ctx context.Context, idOrURL string) (oid int64, typ int, err error) {
	id, err := c.Resolve(ctx, idOrURL)
	if err != nil {
		return 0, 0, err
	}
	switch id.Kind {
	case KindVideo:
		return id.AID, 1, nil
	case KindArticle:
		return id.CVID, 12, nil
	case KindAudio:
		return id.SID, 14, nil
	case KindDynamic:
		var n int64
		fmt.Sscan(id.Dynamic, &n)
		return n, 17, nil
	}
	return 0, 0, fmt.Errorf("no comment area for %s", idOrURL)
}

type rawReplyMember struct {
	Mid       string `json:"mid"`
	Uname     string `json:"uname"`
	Avatar    string `json:"avatar"`
	LevelInfo struct {
		Current int `json:"current_level"`
	} `json:"level_info"`
}

type rawReply struct {
	RpID    int64          `json:"rpid"`
	OID     int64          `json:"oid"`
	Type    int            `json:"type"`
	Mid     int64          `json:"mid"`
	Root    int64          `json:"root"`
	Parent  int64          `json:"parent"`
	Count   int            `json:"count"`
	Rcount  int            `json:"rcount"`
	Like    int            `json:"like"`
	Ctime   int64          `json:"ctime"`
	Member  rawReplyMember `json:"member"`
	Content struct {
		Message string `json:"message"`
	} `json:"content"`
	ReplyControl struct {
		Location string `json:"location"`
	} `json:"reply_control"`
	Replies []rawReply `json:"replies"`
}

type rawReplyMain struct {
	Cursor struct {
		IsEnd bool `json:"is_end"`
		Next  int  `json:"next"`
	} `json:"cursor"`
	Replies []rawReply `json:"replies"`
	Top     struct {
		Upper *rawReply `json:"upper"`
	} `json:"top"`
}

func (c *Client) replyToRecord(r rawReply, top bool) Comment {
	cm := Comment{
		RpID: r.RpID, OID: r.OID, Type: r.Type, Parent: r.Parent, Root: r.Root,
		Mid: r.Mid, Uname: r.Member.Uname, AvatarURL: r.Member.Avatar,
		Level: r.Member.LevelInfo.Current, Content: r.Content.Message, LikeCount: r.Like,
		ReplyCount: r.Rcount, Ctime: r.Ctime, CtimeText: fmtUnix(r.Ctime),
		Location: r.ReplyControl.Location, IsTop: top, FetchedAt: c.fetchedAt(),
	}
	for _, sub := range r.Replies {
		cm.Replies = append(cm.Replies, c.replyToRecord(sub, false))
	}
	return cm
}

// Comments streams top-level comments for an object.
func (c *Client) Comments(ctx context.Context, idOrURL string, opt CommentOptions) iter.Seq2[Comment, error] {
	return func(yield func(Comment, error) bool) {
		oid, typ, err := c.commentTarget(ctx, idOrURL)
		if err != nil {
			yield(Comment{}, err)
			return
		}
		mode := 3 // hot
		if opt.Order == "time" {
			mode = 2
		}
		next := 0
		emitted := 0
		for {
			p := vals("oid", itoa(oid), "type", fmt.Sprint(typ), "mode", fmt.Sprint(mode), "next", fmt.Sprint(next))
			var r rawReplyMain
			if err := c.getJSONSigned(ctx, "https://api.bilibili.com/x/v2/reply/wbi/main", p, &r); err != nil {
				yield(Comment{}, err)
				return
			}
			if next == 0 && r.Top.Upper != nil && r.Top.Upper.RpID != 0 {
				rec := c.replyToRecord(*r.Top.Upper, true)
				if opt.Replies && rec.ReplyCount > len(rec.Replies) {
					rec.Replies = c.allReplies(ctx, oid, typ, rec.RpID)
				}
				if !yield(rec, nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			if len(r.Replies) == 0 {
				return
			}
			for _, rr := range r.Replies {
				rec := c.replyToRecord(rr, false)
				if opt.Replies && rec.ReplyCount > len(rec.Replies) {
					rec.Replies = c.allReplies(ctx, oid, typ, rec.RpID)
				}
				if !yield(rec, nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			if r.Cursor.IsEnd {
				return
			}
			next = r.Cursor.Next
		}
	}
}

// allReplies pages through every reply under a root comment.
func (c *Client) allReplies(ctx context.Context, oid int64, typ int, root int64) []Comment {
	var out []Comment
	pn := 1
	for {
		p := vals("oid", itoa(oid), "type", fmt.Sprint(typ), "root", itoa(root), "pn", fmt.Sprint(pn), "ps", "20")
		var r struct {
			Replies []rawReply `json:"replies"`
			Page    struct {
				Count int `json:"count"`
				Num   int `json:"num"`
				Size  int `json:"size"`
			} `json:"page"`
		}
		if err := c.getJSON(ctx, "https://api.bilibili.com/x/v2/reply/reply", p, &r); err != nil {
			return out
		}
		for _, rr := range r.Replies {
			out = append(out, c.replyToRecord(rr, false))
		}
		if len(r.Replies) == 0 || pn*r.Page.Size >= r.Page.Count {
			return out
		}
		pn++
	}
}
