# Stored captures

One response per state, so that the classifier is tested against what bilibili
actually sends rather than against what this repository believes it sends.

| File | State | Where it came from |
| --- | --- | --- |
| `ok_relation_stat.json` | `ok` | `x/relation/stat?vmid=946974` |
| `ok_fav_page_past_end.json` | `ok` | `x/v3/fav/resource/list` at `pn=999`, a page past the end of a real folder |
| `empty_tags.json` | `empty` | `x/tag/archive/tags?bvid=BV1xx411c7mD`, a video carrying no tags |
| `refused_silent_upstat.json` | `refused_silent` | `x/space/upstat?mid=946974` |
| `refused_silent_fav_list_all.json` | `refused_silent` | `x/v3/fav/folder/created/list-all?up_mid=946974` |
| `risk_352_ranking.json` | `risk` | `x/web-interface/ranking/v2` without the buvid cookies |
| `risk_412.html` | `risk` | `x/polymer/web-dynamic/v1/feed/space` without the buvid cookies |
| `forbidden_reply.json` | `forbidden` | `x/v2/reply/wbi/main` without a signature |
| `not_found_audio.json` | `not_found` | `audio/music-service-c/web/song/info?sid=1` |
| `rate_509.json` | `rate` | reconstructed, see below |

Every file above was captured live on 2026-08-19 except `rate_509.json`.

`-509` is the one state that cannot be captured on demand, because provoking it
means hammering an endpoint until risk control objects, and the whole point of
the rate state is that it clears on its own. That file is the documented shape
of the envelope rather than a recording of one. The classifier only reads
`code`, so nothing in the test depends on the message text.

Two of these are worth reading rather than skimming.

`risk_412.html` is the interstitial risk control serves instead of an envelope:
3400 bytes of a complete HTML page, `content-type: text/html`, `server:
openresty`, with an `x-sec-request-id` header and `出错啦! - bilibili.com` in
the title. A JSON decoder meeting this reports a parse error, which is a true
statement about the bytes and a useless statement about what happened.

`ok_fav_page_past_end.json` is the boundary of what this classification can see.
It is a request for page 999 of a folder with one page, and the answer carries
the folder metadata with `medias: null` inside it. The payload is present, so
the response is `ok`, and it is: the server answered the question. That the list
inside it is empty is a fact about the records, not about the response, and it
belongs to the record envelope rather than to this step.
