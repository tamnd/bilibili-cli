package bili

// Nav is the bootstrap response: login state plus the current WBI keys.
type Nav struct {
	IsLogin bool   `json:"isLogin"`
	Uname   string `json:"uname"`
	Mid     int64  `json:"mid"`
	WbiImg  struct {
		ImgURL string `json:"img_url"`
		SubURL string `json:"sub_url"`
	} `json:"wbi_img"`
}

// Video is the central record.
type Video struct {
	BVID          string   `json:"bvid"`
	AID           int64    `json:"aid"`
	CID           int64    `json:"cid"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	OwnerMid      int64    `json:"owner_mid"`
	OwnerName     string   `json:"owner_name"`
	TypeID        int      `json:"type_id"`
	TypeName      string   `json:"type_name"`
	Duration      int      `json:"duration_seconds"`
	ViewCount     int64    `json:"view_count"`
	DanmakuCount  int64    `json:"danmaku_count"`
	ReplyCount    int64    `json:"reply_count"`
	FavoriteCount int64    `json:"favorite_count"`
	CoinCount     int64    `json:"coin_count"`
	ShareCount    int64    `json:"share_count"`
	LikeCount     int64    `json:"like_count"`
	NowRank       int      `json:"now_rank"`
	HisRank       int      `json:"his_rank"`
	Pubdate       int64    `json:"pubdate"`
	PubdateText   string   `json:"pubdate_text"`
	Ctime         int64    `json:"ctime"`
	Parts         int      `json:"parts"`
	Width         int      `json:"dimension_width"`
	Height        int      `json:"dimension_height"`
	Copyright     int      `json:"copyright"`
	CoverURL      string   `json:"cover_url"`
	ShortLink     string   `json:"short_link"`
	URL           string   `json:"url"`
	Tags          []string `json:"tags,omitempty"`
	Honor         []string `json:"honor,omitempty"`
	Pages         []Page   `json:"pages,omitempty"`
	State         int      `json:"state"`
	FetchedAt     string   `json:"fetched_at"`
}

// Page is one part of a multi-part video.
type Page struct {
	CID        int64  `json:"cid"`
	Page       int    `json:"page"`
	PartTitle  string `json:"part_title"`
	Duration   int    `json:"duration_seconds"`
	Width      int    `json:"dimension_width"`
	Height     int    `json:"dimension_height"`
	FirstFrame string `json:"first_frame_url"`
}

// User is a creator's profile and stat.
type User struct {
	Mid            int64  `json:"mid"`
	Name           string `json:"name"`
	Sex            string `json:"sex"`
	FaceURL        string `json:"face_url"`
	Sign           string `json:"sign"`
	Level          int    `json:"level"`
	TopPhotoURL    string `json:"top_photo_url"`
	OfficialRole   int    `json:"official_role"`
	OfficialTitle  string `json:"official_title"`
	VipType        int    `json:"vip_type"`
	VipStatus      int    `json:"vip_status"`
	Birthday       string `json:"birthday"`
	School         string `json:"school"`
	FollowerCount  int64  `json:"follower_count"`
	FollowingCount int64  `json:"following_count"`
	VideoCount     int64  `json:"video_count"`
	TotalView      int64  `json:"total_view"`
	TotalLike      int64  `json:"total_like"`
	FetchedAt      string `json:"fetched_at"`
}

// Comment is one comment, with nested replies when expanded.
type Comment struct {
	RpID       int64     `json:"rpid"`
	OID        int64     `json:"oid"`
	Type       int       `json:"type"`
	Parent     int64     `json:"parent"`
	Root       int64     `json:"root"`
	Mid        int64     `json:"mid"`
	Uname      string    `json:"uname"`
	AvatarURL  string    `json:"avatar_url"`
	Level      int       `json:"level"`
	Content    string    `json:"content"`
	LikeCount  int       `json:"like_count"`
	ReplyCount int       `json:"reply_count"`
	Ctime      int64     `json:"ctime"`
	CtimeText  string    `json:"ctime_text"`
	Location   string    `json:"location"`
	IsTop      bool      `json:"is_top"`
	Replies    []Comment `json:"replies,omitempty"`
	FetchedAt  string    `json:"fetched_at"`
}

// Danmaku is one bullet-chat line.
type Danmaku struct {
	DmID       int64  `json:"dmid"`
	ProgressMs int32  `json:"progress_ms"`
	Mode       int32  `json:"mode"`
	Fontsize   int32  `json:"fontsize"`
	Color      uint32 `json:"color"`
	Ctime      int64  `json:"ctime"`
	Pool       int32  `json:"pool"`
	SenderHash string `json:"sender_hash"`
	Content    string `json:"content"`
}

// Dynamic is one post in a user's feed.
type Dynamic struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	AuthorMid   int64    `json:"author_mid"`
	AuthorName  string   `json:"author_name"`
	PubTs       int64    `json:"pub_ts"`
	PubText     string   `json:"pub_text"`
	Text        string   `json:"text"`
	Pics        []string `json:"pics,omitempty"`
	OrigID      string   `json:"orig_id,omitempty"`
	VideoBVID   string   `json:"video_bvid,omitempty"`
	StatLike    int64    `json:"stat_like"`
	StatReply   int64    `json:"stat_reply"`
	StatForward int64    `json:"stat_forward"`
	FetchedAt   string   `json:"fetched_at"`
}

// LiveRoom is a live streaming room.
type LiveRoom struct {
	RoomID         int64  `json:"room_id"`
	ShortID        int64  `json:"short_id"`
	UID            int64  `json:"uid"`
	Uname          string `json:"uname"`
	Title          string `json:"title"`
	AreaName       string `json:"area_name"`
	ParentAreaName string `json:"parent_area_name"`
	LiveStatus     int    `json:"live_status"`
	Online         int64  `json:"online"`
	Attention      int64  `json:"attention"`
	Tags           string `json:"tags"`
	CoverURL       string `json:"cover_url"`
	KeyframeURL    string `json:"keyframe_url"`
	LiveStartText  string `json:"live_start_text"`
	FetchedAt      string `json:"fetched_at"`
}

// Bangumi is an anime/film season with its episodes.
type Bangumi struct {
	SeasonID      int64     `json:"season_id"`
	MediaID       int64     `json:"media_id"`
	SeasonTitle   string    `json:"season_title"`
	Title         string    `json:"title"`
	TypeName      string    `json:"type_name"`
	TotalEp       int       `json:"total_ep"`
	Status        int       `json:"status"`
	CoverURL      string    `json:"cover_url"`
	Evaluate      string    `json:"evaluate"`
	RatingScore   float64   `json:"rating_score"`
	RatingCount   int64     `json:"rating_count"`
	Area          string    `json:"area"`
	Styles        []string  `json:"styles,omitempty"`
	PublishText   string    `json:"publish_text"`
	StatViews     int64     `json:"stat_views"`
	StatFavorites int64     `json:"stat_favorites"`
	StatDanmakus  int64     `json:"stat_danmakus"`
	Episodes      []Episode `json:"episodes,omitempty"`
	FetchedAt     string    `json:"fetched_at"`
}

// Episode is one episode of a bangumi season.
type Episode struct {
	EpID      int64  `json:"ep_id"`
	AID       int64  `json:"aid"`
	BVID      string `json:"bvid"`
	CID       int64  `json:"cid"`
	Title     string `json:"title"`
	LongTitle string `json:"long_title"`
	CoverURL  string `json:"cover_url"`
	Duration  int    `json:"duration_seconds"`
	PubText   string `json:"pub_text"`
	Badge     string `json:"badge"`
}

// Audio is a music track.
type Audio struct {
	SID           int64  `json:"sid"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	Uname         string `json:"uname"`
	UID           int64  `json:"uid"`
	CoverURL      string `json:"cover_url"`
	Intro         string `json:"intro"`
	Duration      int    `json:"duration_seconds"`
	PlayCount     int64  `json:"play_count"`
	ReplyCount    int64  `json:"reply_count"`
	FavoriteCount int64  `json:"favorite_count"`
	ShareCount    int64  `json:"share_count"`
	Ctime         int64  `json:"ctime"`
	FetchedAt     string `json:"fetched_at"`
}

// Article is a column post.
type Article struct {
	CVID          int64  `json:"cvid"`
	Title         string `json:"title"`
	AuthorMid     int64  `json:"author_mid"`
	AuthorName    string `json:"author_name"`
	Summary       string `json:"summary"`
	BannerURL     string `json:"banner_url"`
	CategoryName  string `json:"category_name"`
	Words         int    `json:"words"`
	ViewCount     int64  `json:"view_count"`
	LikeCount     int64  `json:"like_count"`
	ReplyCount    int64  `json:"reply_count"`
	FavoriteCount int64  `json:"favorite_count"`
	CoinCount     int64  `json:"coin_count"`
	PublishTime   int64  `json:"publish_time"`
	ContentText   string `json:"content_text,omitempty"`
	FetchedAt     string `json:"fetched_at"`
}

// Favorite is a favorite folder.
type Favorite struct {
	MediaID    int64  `json:"media_id"`
	FID        int64  `json:"fid"`
	Title      string `json:"title"`
	CoverURL   string `json:"cover_url"`
	Intro      string `json:"intro"`
	MediaCount int    `json:"media_count"`
	OwnerMid   int64  `json:"owner_mid"`
	OwnerName  string `json:"owner_name"`
	Ctime      int64  `json:"ctime"`
	FetchedAt  string `json:"fetched_at"`
}

// SearchResult is a discriminated search hit.
type SearchResult struct {
	ResultType string    `json:"result_type"`
	Video      *Video    `json:"video,omitempty"`
	User       *User     `json:"user,omitempty"`
	Bangumi    *Bangumi  `json:"bangumi,omitempty"`
	LiveRoom   *LiveRoom `json:"live_room,omitempty"`
	Article    *Article  `json:"article,omitempty"`
}

// Payload returns the concrete record the result wraps (a *Video, *User, etc.),
// so output formats can render real columns instead of the wrapper. It falls
// back to the wrapper itself if nothing is set.
func (r SearchResult) Payload() any {
	switch {
	case r.Video != nil:
		return r.Video
	case r.User != nil:
		return r.User
	case r.Bangumi != nil:
		return r.Bangumi
	case r.LiveRoom != nil:
		return r.LiveRoom
	case r.Article != nil:
		return r.Article
	default:
		return r
	}
}

// Stream is one playable quality of a video part.
type Stream struct {
	Quality     int      `json:"quality"`
	QualityText string   `json:"quality_text"`
	Codecs      string   `json:"codecs"`
	MIME        string   `json:"mime"`
	Bandwidth   int64    `json:"bandwidth"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	FrameRate   string   `json:"frame_rate"`
	URL         string   `json:"url"`
	BackupURLs  []string `json:"backup_urls,omitempty"`
	DurationMs  int64    `json:"duration_ms"`
}

// Suggestion is one autosuggest term.
type Suggestion struct {
	Term string `json:"term"`
}

// ListOptions controls paged list endpoints.
type ListOptions struct {
	Page     int
	PageSize int
	Order    string
	Keyword  string
	Limit    int
}

// SearchOptions controls search.
type SearchOptions struct {
	Type     string
	Order    string
	Duration int
	Tid      int
	Page     int
	PageSize int
	Limit    int
}

// VideoOptions controls the video command.
type VideoOptions struct {
	Detail  bool
	Related bool
	Tags    bool
}

// CommentOptions controls the comments command.
type CommentOptions struct {
	Order   string // hot | time
	Replies bool
	Limit   int
}
