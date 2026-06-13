---
title: "bili"
description: "A fast, friendly command line for bilibili.com. Resolve any video, user, comment, danmaku, dynamic, live room, bangumi, audio, article, or favorite into clean structured records, all from one binary."
heroTitle: "bilibili, from the command line"
heroLead: "bili is a single pure-Go binary that puts bilibili.com behind a tool that feels like curl. Resolve any id or URL into rich structured records, pull comments and bullet-chat, search and browse the popular feeds, fetch playable stream URLs, and crawl the whole graph into JSONL, with no API key and nothing to pay for."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

Working with bilibili usually means reverse-engineering the web API by hand: the
`{code, message, data}` envelope, WBI request signing, anti-bot fingerprint
cookies, the av/BV id conversion, and protobuf-encoded danmaku. bili puts all of
it behind one tool with sensible defaults, real output formats, and pipelines
that compose.

```bash
bili video BV17x411w7KC              # full metadata for a video
bili comments BV17x411w7KC -o jsonl  # every comment and reply, one per line
bili search lofi -o url              # the watch URLs of matching videos
bili crawl BV17x411w7KC --out ./data # the connected graph, into JSONL files
```

It runs anonymously against `api.bilibili.com` over plain HTTPS, so there is
nothing to sign up for. The binary is pure Go with no runtime dependencies.

## What you can do with it

- **Resolve anything.** Hand `bili` a `BV…`, `av…`, `ss…`/`ep…`, `md…`, `au…`,
  `cv…`, `ml…`, a space link, a live link, or a dynamic link, and it normalizes
  it and fetches the record.
- **Get rich records.** Every command emits the fields the API returns rather
  than flattening them, so `-o json` is lossless and the table view is a
  readable projection of it.
- **Pull conversations.** `bili comments` walks the full comment tree and
  `bili danmaku` decodes the protobuf bullet-chat into plain rows.
- **Find things.** Search videos, users, bangumi, live rooms, and articles, read
  the popular feeds, the leaderboard, and the hot-search terms.
- **Crawl the graph.** `bili crawl` walks outward from seed ids into per-type
  JSONL files, following related videos and optionally comments and danmaku.

## Where to go next

- New here? Start with the [introduction](/getting-started/introduction/) for
  the mental model, then the [quick start](/getting-started/quick-start/).
- Want to install it? See [installation](/getting-started/installation/).
- Looking for a specific task? The [guides](/guides/) cover videos, comments and
  danmaku, users and feeds, bangumi and audio, live rooms, finding things, and
  crawling.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
