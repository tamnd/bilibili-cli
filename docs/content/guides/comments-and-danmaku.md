---
title: "Comments and danmaku"
description: "Pull the full comment tree on any object, and decode bullet-chat into rows."
weight: 20
---

bilibili has two kinds of conversation: threaded **comments** under an object,
and **danmaku**, the bullet-chat that scrolls across a video. bili reads both as
clean structured records.

## Comments

```bash
bili comments BV17x411w7KC
```

`comments` walks the full comment tree: top-level comments and the replies
nested under each one. It works on anything that can be commented on, not just
videos, and figures out the object type from the id you give it:

```bash
bili comments BV17x411w7KC      # a video
bili comments cv7018872         # a column article
bili comments au1               # an audio track
bili comments <dynamic-id>      # a dynamic post
```

Each record carries the author, the text, the like count, the timestamp, and the
reply relationship, so you can reconstruct threads downstream:

```bash
bili comments BVID -o jsonl > comments.jsonl
```

Use `-n` to cap how many you pull on a busy video, and `--order` where the
endpoint supports sorting by time or by likes.

## Danmaku

```bash
bili danmaku BV17x411w7KC
```

Danmaku is delivered as protobuf segments, one per six minutes of video. bili
fetches the segments for a part and decodes them into rows with `progress` (the
millisecond offset into the video), `mode`, `color`, `fontsize`, `content`, and
the sender's hashed id.

Because it is plain data, it answers questions a player cannot:

```bash
# the busiest moments, by comment count per 10s bucket
bili danmaku BVID -o jsonl \
  | jq -r '(.progress/10000|floor) ' | sort -n | uniq -c | sort -rn | head

# every comment in the first minute
bili danmaku BVID -o jsonl | jq -r 'select(.progress < 60000) | .content'
```

For a multi-part video, `--page` selects the part (see
[videos](/guides/videos/)).

## Pulling both at scale

When you want the whole conversation around a set of videos, let
[crawl](/guides/crawling/) do it:

```bash
bili crawl BVID --out ./data --comments --danmaku
# writes comments.jsonl and danmaku.jsonl alongside videos.jsonl
```
