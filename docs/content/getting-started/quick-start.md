---
title: "Quick start"
description: "From an empty terminal to a real video, its comments, and its bullet-chat, in a handful of commands."
weight: 30
---

This walks the core loop: resolve an id, read a video's full metadata, and pull
its comments and danmaku. Every command here hits live data and finishes in a
second or two. No login required.

## 1. Resolve an id

`bili id` tells you what a string is before you do anything with it:

```bash
bili id BV17x411w7KC
```

```
kind   video
bvid   BV17x411w7KC
aid    170001
url    https://www.bilibili.com/video/BV17x411w7KC
```

It accepts any form: an `av` number, a `BV` string, a `space.bilibili.com` link,
an `ss`/`ep` season or episode, and more. The other commands accept all the same
forms, so you rarely call `id` directly; it is just the quickest way to see how
something will be classified.

## 2. Read a video

```bash
bili video BV17x411w7KC --fields bvid,aid,title,owner_name,view_count,pubdate_text
```

```
bvid          aid     title                     owner_name  view_count  pubdate_text
BV17x411w7KC  170001  【MV】保加利亚妖王AZIS视频合辑    冰封.虾子      45703352    2011-11-09 21:55:33
```

The table is a projection of a much richer record. Ask for JSON to see
everything the API returns:

```bash
bili video BV17x411w7KC -o json
```

## 3. Pull the conversation

Every comment and reply on the video, one JSON object per line:

```bash
bili comments BV17x411w7KC -o jsonl | head
```

The bullet-chat (danmaku) for the video, decoded from protobuf into rows:

```bash
bili danmaku BV17x411w7KC -n 5
```

```
progress  mode  color     content
1200      1     16777215  first!
3400      1     16777215  classic
...
```

## 4. Find things

Search, and feed the results into another command:

```bash
bili search lofi -o url | head
bili popular -n 5             # the popular feed
bili trending                # current hot-search terms
```

## 5. Compose

Output that pipes is the point. Resolve every search hit to full metadata:

```bash
bili search lofi -o url | bili video -
```

Crawl a seed video plus its comments and related videos into JSONL files:

```bash
bili crawl BV17x411w7KC --out ./data --comments
ls ./data
# comments.jsonl  users.jsonl  videos.jsonl
```

## Where to next

You have the core loop. From here:

- [Videos](/guides/videos/) covers metadata, related, streams, and danmaku.
- [Comments and danmaku](/guides/comments-and-danmaku/) goes deep on the
  conversation data.
- [Users and feeds](/guides/users-and-feeds/) covers creators, favorites, and
  dynamics.
- [Crawling](/guides/crawling/) builds a dataset from seed ids.
- The [CLI reference](/reference/cli/) lists every command and flag.
