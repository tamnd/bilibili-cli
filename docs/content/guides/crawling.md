---
title: "Crawling"
description: "Walk the graph from seed ids into per-type JSONL files."
weight: 70
---

Every other command answers one question at a time. `crawl` is for building a
dataset: hand it seed ids and it walks outward, writing one JSONL file per record
type into a directory.

For exploring the graph interactively rather than dumping it to disk, see
[discovering](/guides/discovering/): `discover` does a breadth-first walk and
streams one record per node to stdout, in any output format. Reach for
`discover` to look around; reach for `crawl` to bulk-dump.

## The basics

```bash
bili crawl BV17x411w7KC --out ./data
```

```
./data/
  videos.jsonl    one object per video reached
  users.jsonl     the owners of those videos
```

By default `crawl` follows **related videos** from each seed, so a single seed
fans out into the neighborhood around it. Each record is the same rich, lossless
object the matching single-item command would emit.

## Following more edges

Turn on the extra record types you want:

```bash
bili crawl BVID --out ./data --comments --danmaku --related
```

```
./data/
  videos.jsonl
  users.jsonl
  comments.jsonl   with --comments
  danmaku.jsonl    with --danmaku
```

`--related` is on by default; pass `--related=false` to crawl only the seeds and
their attached data without expanding to neighbors.

## Seeds from anywhere

Seeds can be several ids on the command line, or a list on stdin via `-`, so any
command that emits URLs feeds the crawler:

```bash
bili crawl BV1xx BV1yy --out ./data           # several seeds
bili search lofi -o url | bili crawl - --out ./data
bili user 2 --videos -o url | bili crawl - --out ./data --comments
```

## Staying polite and resilient

A crawl makes a lot of requests, so the global politeness controls matter here:

- `--rate` keeps a minimum gap between requests (default 350ms).
- `--retries` retries on rate-limit and risk-control responses (default 4).
- The on-disk cache means re-running a crawl over overlapping seeds does not
  re-fetch what it already has; add `--no-cache` to force fresh reads.

Because the output is JSONL, the result loads into anything: `jq`, DuckDB,
pandas, or a database importer. Each file is independent, so you can process
videos without touching comments, or join them on the ids they share.
