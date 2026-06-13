---
title: "Finding things"
description: "Search across the site, autosuggest, hot-search terms, the popular feeds, and the leaderboard."
weight: 60
---

When you do not already have an id, five commands help you find one: `search`,
`suggest`, `trending`, `popular`, and `rank`. Their output flows straight into
the resolving commands.

## Search

```bash
bili search lofi
```

`search` queries across types and emits one record per hit. The record adapts to
what it found: a video hit looks like a video, a user hit like a user, and so on,
so a mixed result still renders cleanly and pipes by URL:

```bash
bili search lofi -o url            # the URL of every hit
bili search lofi --type video      # restrict to one type
bili search lofi --type user
```

Restrict with `--type` to `video`, `user`, `bangumi`, `live`, or `article`. Page
through results with `--page`, and cap them with `-n`.

The natural next step is to resolve every hit to its full record:

```bash
bili search lofi -o url | bili video -
```

Search is one of the more aggressively rate-limited endpoints, so a fast loop may
occasionally see a transient risk-control error; bili retries, and `--rate`
keeps a polite gap between calls.

## Autosuggest and hot search

```bash
bili suggest lof       # the terms bilibili would autocomplete
bili trending          # the current hot-search terms
```

These are the same lists the search box shows. `trending` is a quick read on
what the site is talking about right now.

## Popular feeds

```bash
bili popular           # the popular (综合热门) feed
bili popular -n 20
```

`popular` reads the general popular feed. With the right flag it also reads a
specific issue of the weekly selection (每周必看), bilibili's curated weekly list.

## The leaderboard

```bash
bili rank              # the overall leaderboard
bili rank --partition <id>   # one category's leaderboard
```

`rank` reads the排行榜 leaderboard, optionally scoped to a single partition
(category). Each row is a video, so it pipes into `video` and `crawl` like any
other list.
