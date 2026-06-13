---
title: "Bangumi, audio, and articles"
description: "Read anime and film seasons, audio tracks, and column articles."
weight: 40
---

Beyond user-uploaded videos, bilibili carries professionally licensed seasons
(bangumi), an audio library, and a long-form column. Each has its own command and
its own id scheme.

## Bangumi

A season is addressed several ways, and `bangumi` accepts all of them:

```bash
bili bangumi ss33802     # by season id
bili bangumi ep331204    # by a single episode id
bili bangumi md28229233  # by the media (detail page) id
```

The record carries the season's metadata and the full episode list, each episode
with its title, id, and the `cid` you would use to fetch its danmaku. Note that
the bangumi endpoints are the ones that return their payload in `result` rather
than `data`; bili handles that transparently.

```bash
bili bangumi ss33802 -o json     # season plus every episode, lossless
```

## Audio

```bash
bili audio au1
```

`audio` resolves an audio track to its metadata and statistics: title, uploader,
play and coin counts, and duration.

## Articles

bilibili's column (专栏) is long-form writing. `article` resolves a `cv` id or a
`read/cv` URL to the article's metadata, and pulls the body text when asked:

```bash
bili article cv7018872
bili article cv7018872 --text     # include the article body
```

Articles can be commented on, so `bili comments cv7018872` reads the discussion
under one (see [comments and danmaku](/guides/comments-and-danmaku/)).
