package bili

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/bilibili-cli/pkg/bvconv"
)

// IDKind classifies an identifier.
type IDKind string

const (
	KindVideo    IDKind = "video"
	KindUser     IDKind = "user"
	KindBangumi  IDKind = "bangumi"
	KindEpisode  IDKind = "episode"
	KindMedia    IDKind = "media"
	KindLive     IDKind = "live"
	KindAudio    IDKind = "audio"
	KindArticle  IDKind = "article"
	KindFavorite IDKind = "favorite"
	KindDynamic  IDKind = "dynamic"
	KindUnknown  IDKind = "unknown"
)

// Identity is the normalized form of any pasted identifier.
type Identity struct {
	Kind     IDKind `json:"kind"`
	BVID     string `json:"bvid,omitempty"`
	AID      int64  `json:"aid,omitempty"`
	Mid      int64  `json:"mid,omitempty"`
	SeasonID int64  `json:"season_id,omitempty"`
	EpID     int64  `json:"ep_id,omitempty"`
	MediaID  int64  `json:"media_id,omitempty"`
	RoomID   int64  `json:"room_id,omitempty"`
	SID      int64  `json:"sid,omitempty"`
	CVID     int64  `json:"cvid,omitempty"`
	FavID    int64  `json:"fav_id,omitempty"`
	Dynamic  string `json:"dynamic,omitempty"`
	Input    string `json:"input"`
}

var (
	reBV     = regexp.MustCompile(`(?i)(BV[0-9A-Za-z]{10})`)
	reAV     = regexp.MustCompile(`(?i)av(\d+)`)
	reSS     = regexp.MustCompile(`(?i)ss(\d+)`)
	reEP     = regexp.MustCompile(`(?i)ep(\d+)`)
	reMD     = regexp.MustCompile(`(?i)md(\d+)`)
	reAU     = regexp.MustCompile(`(?i)au(\d+)`)
	reCV     = regexp.MustCompile(`(?i)cv(\d+)`)
	reML     = regexp.MustCompile(`(?i)ml(\d+)`)
	reSpace  = regexp.MustCompile(`space\.bilibili\.com/(\d+)`)
	reLive   = regexp.MustCompile(`live\.bilibili\.com/(\d+)`)
	reTBili  = regexp.MustCompile(`t\.bilibili\.com/(\d+)`)
	reDigits = regexp.MustCompile(`^\d+$`)
)

// Resolve classifies and normalizes an arbitrary id or URL. It performs at most
// one network call (to follow b23.tv short links).
func (c *Client) Resolve(ctx context.Context, s string) (*Identity, error) {
	in := strings.TrimSpace(s)
	id := &Identity{Input: in, Kind: KindUnknown}
	work := in

	// follow short links once
	if strings.Contains(work, "b23.tv") {
		if resolved, err := c.followRedirect(ctx, ensureScheme(work)); err == nil {
			work = resolved
		}
	}

	switch {
	case reSpace.MatchString(work):
		m := reSpace.FindStringSubmatch(work)
		id.Mid, _ = strconv.ParseInt(m[1], 10, 64)
		id.Kind = KindUser
	case reLive.MatchString(work):
		m := reLive.FindStringSubmatch(work)
		id.RoomID, _ = strconv.ParseInt(m[1], 10, 64)
		id.Kind = KindLive
	case reTBili.MatchString(work):
		m := reTBili.FindStringSubmatch(work)
		id.Dynamic = m[1]
		id.Kind = KindDynamic
	case reBV.MatchString(work):
		id.BVID = normalizeBV(reBV.FindStringSubmatch(work)[1])
		if aid, err := bvconv.ToAV(id.BVID); err == nil {
			id.AID = aid
		}
		id.Kind = KindVideo
	case reEP.MatchString(work):
		id.EpID, _ = strconv.ParseInt(reEP.FindStringSubmatch(work)[1], 10, 64)
		id.Kind = KindEpisode
	case reSS.MatchString(work):
		id.SeasonID, _ = strconv.ParseInt(reSS.FindStringSubmatch(work)[1], 10, 64)
		id.Kind = KindBangumi
	case reMD.MatchString(work):
		id.MediaID, _ = strconv.ParseInt(reMD.FindStringSubmatch(work)[1], 10, 64)
		id.Kind = KindMedia
	case reAU.MatchString(work) && !reAV.MatchString(work):
		id.SID, _ = strconv.ParseInt(reAU.FindStringSubmatch(work)[1], 10, 64)
		id.Kind = KindAudio
	case reCV.MatchString(work):
		id.CVID, _ = strconv.ParseInt(reCV.FindStringSubmatch(work)[1], 10, 64)
		id.Kind = KindArticle
	case reML.MatchString(work):
		id.FavID, _ = strconv.ParseInt(reML.FindStringSubmatch(work)[1], 10, 64)
		id.Kind = KindFavorite
	case reAV.MatchString(work):
		id.AID, _ = strconv.ParseInt(reAV.FindStringSubmatch(work)[1], 10, 64)
		id.BVID = bvconv.ToBV(id.AID)
		id.Kind = KindVideo
	case reDigits.MatchString(work):
		// bare number: treat as aid for video by default; callers that want a
		// mid pass it through Resolve with a hint.
		id.AID, _ = strconv.ParseInt(work, 10, 64)
		id.BVID = bvconv.ToBV(id.AID)
		id.Kind = KindVideo
	case len(work) >= 17 && reDigits.MatchString(strings.TrimSpace(work)):
		id.Dynamic = work
		id.Kind = KindDynamic
	}
	if id.Kind == KindUnknown {
		return id, fmt.Errorf("could not classify %q", in)
	}
	return id, nil
}

// followRedirect returns the final URL after following redirects.
func (c *Client) followRedirect(ctx context.Context, u string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return resp.Request.URL.String(), nil
}

func ensureScheme(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	return "https://" + s
}

func normalizeBV(s string) string {
	if len(s) == 12 {
		return "BV" + s[2:]
	}
	return s
}
