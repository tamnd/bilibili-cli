---
title: "Videos"
description: "Resolve videos to full metadata, find related videos, list playable streams, and read bullet-chat."
weight: 10
---

A video is the center of bilibili, and four commands cover it: `video` for
metadata, `related` for what sits next to it, `streams` for playable URLs, and
`danmaku` for bullet-chat. All four accept any id form (`BV…`, `av…`, or a full
watch URL).

## Metadata

```bash
bili video BV17x411w7KC
```

The table view shows the headline fields; `-o json` is the full lossless record,
including the owner, statistics, dimensions, and every part (`page`) of a
multi-part video.

`video` takes more than one id at once, and reads ids from stdin with `-`, so it
composes with any command that emits URLs:

```bash
bili video BV17x411w7KC BV1xx411c7XW       # several at once
bili search lofi -o url | bili video -      # everything a search found
```

## Multi-part videos

Many videos have several parts, each with its own `cid`. The `streams` and
`danmaku` commands work on one part at a time and default to the first. Pick
another with `--page` (1-based) or by passing the `cid` directly with `--cid`:

```bash
bili video BVID -o json        # look at the "pages" array for the parts
bili danmaku BVID --page 2     # bullet-chat for the second part
```

## Related videos

```bash
bili related BV17x411w7KC
bili related BV17x411w7KC -o url     # just the watch URLs
```

This is the same list bilibili shows alongside a video, which makes it the
natural edge to follow when [crawling](/crawling/).

## Streams

```bash
bili streams BV17x411w7KC
```

`streams` lists the playable media URLs the API exposes for a part, with their
quality, codec, and format. Use `--quality` to ask for a specific level. These
are the URLs the player would use; bili does not download or decrypt anything,
it just reports what is offered.

## Bullet-chat

```bash
bili danmaku BV17x411w7KC
```

Each row is one comment that scrolls across the video: its `progress` (the
millisecond offset where it appears), `mode`, `color`, `fontsize`, and `content`.
The data arrives as protobuf and bili decodes it into plain records, so it sorts,
filters, and pipes like anything else:

```bash
bili danmaku BVID -o jsonl | jq -r 'select(.progress < 10000) | .content'
```

See [comments and danmaku](/guides/comments-and-danmaku/) for the conversation
data in depth.
