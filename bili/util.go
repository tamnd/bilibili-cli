package bili

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (c *Client) fetchedAt() string {
	return c.now().UTC().Format(time.RFC3339)
}

// flexInt64 unmarshals from either a JSON number or a JSON string. Several
// bilibili fields (e.g. dynamic pub_ts) are inconsistently typed across items.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil // tolerate non-numeric strings
	}
	*f = flexInt64(n)
	return nil
}

func fmtUnix(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

func vals(pairs ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v.Set(pairs[i], pairs[i+1])
	}
	return v
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

var reTag = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return reTag.ReplaceAllString(s, "")
}

func splitComma(s string) []string {
	return strings.Split(s, ",")
}

var (
	reScript  = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reArticle = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*article-holder[^"]*"[^>]*>(.*?)</div>\s*</div>`)
	reWS      = regexp.MustCompile(`[ \t]+`)
	reNL      = regexp.MustCompile(`\n{3,}`)
)

// extractArticleText pulls a readable plain-text body out of a column article page.
func extractArticleText(body []byte) string {
	s := string(body)
	s = reScript.ReplaceAllString(s, "")
	if m := reArticle.FindStringSubmatch(s); len(m) > 1 {
		s = m[1]
	}
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = stripTags(s)
	s = htmlUnescape(s)
	s = reWS.ReplaceAllString(s, " ")
	s = reNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func htmlUnescape(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'", "&nbsp;", " ")
	return r.Replace(s)
}

func parseSuggest(body []byte) ([]string, error) {
	var r struct {
		Result struct {
			Tag []struct {
				Value string `json:"value"`
				Name  string `json:"name"`
			} `json:"tag"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	var out []string
	for _, t := range r.Result.Tag {
		v := t.Value
		if v == "" {
			v = t.Name
		}
		if v != "" {
			out = append(out, v)
		}
	}
	return out, nil
}
