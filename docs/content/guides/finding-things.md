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

Restrict with `--type` to `video`, `user`, `bangumi`, `film`, `live_room` (which
also answers to `live`), or `article`. Page through results with `--page`, and
cap them with `-n`.

The natural next step is to resolve every hit to its full record:

```bash
bili search lofi -o url | bili video -
```

Search is one of the more aggressively rate-limited endpoints, so a fast loop
can see a rate limit or a risk control refusal. bili retries the rate limit and
never retries the refusal, because asking again four times is the worst possible
answer to being turned away. Raise `--rate` and let the cache absorb repeats.

Worth knowing before you write a loop around it: this endpoint reports 1000
results and 50 pages for every query measured, including nonsense ones, so those
counts are a constant and not an answer, and a search that genuinely matched
nothing is not something the API can express.

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

`popular` reads the general popular feed. `--weekly <n>` reads a specific issue
of the weekly selection (每周必看) instead, bilibili's curated weekly list.

## The leaderboard

```bash
bili rank              # the overall leaderboard
bili rank --tid 129    # one category's leaderboard, 129 being dance
```

`rank` reads the 排行榜 leaderboard, optionally scoped to a single partition
(category) by its numeric id. Each row is a video, so it pipes into `video` and
`crawl` like any other list.
