package bili

import (
	"context"
	"fmt"
	"iter"
)

type rawRoomInfo struct {
	RoomID         int64  `json:"room_id"`
	ShortID        int64  `json:"short_id"`
	UID            int64  `json:"uid"`
	Title          string `json:"title"`
	UserCover      string `json:"user_cover"`
	Keyframe       string `json:"keyframe"`
	LiveStatus     int    `json:"live_status"`
	Online         int64  `json:"online"`
	Attention      int64  `json:"attention"`
	AreaName       string `json:"area_name"`
	ParentAreaName string `json:"parent_area_name"`
	Tags           string `json:"tags"`
	LiveTime       string `json:"live_time"`
}

// Live fetches a live room by room id, or by uid when byUID is set.
func (c *Client) Live(ctx context.Context, roomOrUID string, byUID bool) (*LiveRoom, error) {
	roomID := roomOrUID
	if byUID {
		var status struct {
			RoomID int64 `json:"room_id"`
		}
		if err := c.getJSON(ctx, "https://api.live.bilibili.com/room/v2/Room/room_id_by_uid", vals("uid", roomOrUID), &status); err != nil {
			return nil, err
		}
		if status.RoomID != 0 {
			roomID = itoa(status.RoomID)
		}
	}
	var ri rawRoomInfo
	if err := c.getJSON(ctx, "https://api.live.bilibili.com/room/v1/Room/get_info", vals("room_id", roomID), &ri); err != nil {
		return nil, err
	}
	room := &LiveRoom{
		RoomID: ri.RoomID, ShortID: ri.ShortID, UID: ri.UID,
		Title: ri.Title, AreaName: ri.AreaName, ParentAreaName: ri.ParentAreaName,
		LiveStatus: ri.LiveStatus, Online: ri.Online, Attention: ri.Attention, Tags: ri.Tags,
		CoverURL: ri.UserCover, KeyframeURL: ri.Keyframe, LiveStartText: ri.LiveTime,
		FetchedAt: c.fetchedAt(),
	}
	// the room endpoint omits the anchor name; fetch it best-effort.
	var master struct {
		Info struct {
			Uname string `json:"uname"`
		} `json:"info"`
	}
	if err := c.getJSON(ctx, "https://api.live.bilibili.com/live_user/v1/Master/info", vals("uid", itoa(ri.UID)), &master); err == nil {
		room.Uname = master.Info.Uname
	}
	return room, nil
}

// BrowseLive streams live rooms in an area.
func (c *Client) BrowseLive(ctx context.Context, areaID int, opt ListOptions) iter.Seq2[LiveRoom, error] {
	return func(yield func(LiveRoom, error) bool) {
		page := opt.Page
		if page < 1 {
			page = 1
		}
		// getRoomList requires a parent category. Treat --area as the parent area id
		// (1=网游 2=手游 3=单机 5=电台 9=虚拟 etc.); default to 1 when unset. area_id
		// stays 0 to list every sub-area under that parent.
		parent := "1"
		if areaID > 0 {
			parent = fmt.Sprint(areaID)
		}
		area := "0"
		emitted := 0
		for {
			p := vals("platform", "web", "parent_area_id", parent, "area_id", area,
				"sort_type", "online", "page", fmt.Sprint(page), "page_size", "30")
			var r struct {
				List []struct {
					RoomID         int64  `json:"roomid"`
					UID            int64  `json:"uid"`
					Title          string `json:"title"`
					Uname          string `json:"uname"`
					Online         int64  `json:"online"`
					Cover          string `json:"cover"`
					Keyframe       string `json:"keyframe"`
					AreaName       string `json:"area_name"`
					ParentAreaName string `json:"parent_name"`
					Tags           string `json:"tags"`
				} `json:"list"`
			}
			if err := c.getJSON(ctx, "https://api.live.bilibili.com/room/v3/area/getRoomList", p, &r); err != nil {
				yield(LiveRoom{}, err)
				return
			}
			if len(r.List) == 0 {
				return
			}
			for _, it := range r.List {
				rec := LiveRoom{
					RoomID: it.RoomID, UID: it.UID, Title: it.Title, Uname: it.Uname,
					Online: it.Online, CoverURL: it.Cover, KeyframeURL: it.Keyframe,
					AreaName: it.AreaName, ParentAreaName: it.ParentAreaName, Tags: it.Tags,
					LiveStatus: 1, FetchedAt: c.fetchedAt(),
				}
				if !yield(rec, nil) {
					return
				}
				emitted++
				if opt.Limit > 0 && emitted >= opt.Limit {
					return
				}
			}
			page++
		}
	}
}
