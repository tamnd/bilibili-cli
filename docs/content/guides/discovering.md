---
title: "Discovering"
description: "Walk the graph of related videos and creators, breadth first, streaming one record per node."
weight: 75
---

Every other command answers one question about one object: a video's related
list, a creator's catalogue, the comments on a post. `discover` chains them. From
a seed video or creator it follows the object's edges, and from each neighbor it
follows theirs, hop by hop, streaming one record per node as the node is reached.

```bash
bili discover BV17x411w7KC
```

A seed is anything `bili` can resolve to a video or a creator: a `BV`/`av` id, a
video URL, a `b23.tv` short link, or a `space.bilibili.com` creator URL.

## The graph

There are two kinds of node, videos and creators, and three edges between them:

| Edge | From → to | What it follows |
|---|---|---|
| `related` | video → video | the videos bilibili recommends alongside one |
| `uploader` | video → creator | the creator who posted a video |
| `uploads` | creator → video | the videos in a creator's catalogue |

You rarely name edges one at a time. `--follow` takes a **preset**:

| Preset | Expands to | Walk shape |
|---|---|---|
| `content` *(default)* | `related` + `uploads` | stays among videos: a video's recommendations, a creator's uploads |
| `creators` | `uploader` + `uploads` | bounces between videos and the people who make them |
| `all` | every edge | the whole neighborhood |

```bash
bili discover BV17x411w7KC                       # content (the default)
bili discover BV17x411w7KC --follow creators     # video → uploader → their uploads
bili discover BV17x411w7KC --follow all --depth 2
```

`--follow` also takes a single edge name, or a comma-separated mix of presets and
edges, so you can be exact:

```bash
bili discover BV17x411w7KC --follow uploader        # just hop to the creator
bili discover BV17x411w7KC --follow related,uploader
```

## Bounding the walk

Three independent limits keep a walk finite:

- `--depth` is how many hops to follow (default `1`; `0` emits only the seeds).
- `--fanout` caps neighbors per edge (default `25`).
- `-n` caps the total nodes streamed (default `500`), so an unbounded
  `discover BV…` always terminates instead of spidering forever.

```bash
bili discover BV17x411w7KC --depth 3 --fanout 10 -n 200
```

## Reading the output

Each row is a node tagged with how it was reached: how deep, by which edge, the
object and its owner. The full typed record rides along for `-o json`/`-o jsonl`,
and `-o url` prints one link per node:

```bash
bili discover BV17x411w7KC                 # the readable table
bili discover BV17x411w7KC -o jsonl        # one lossless object per line
bili discover BV17x411w7KC -o url          # one URL per node, to pipe onward
```

Seeds can come from stdin via `-`, so any command that emits URLs feeds a walk:

```bash
bili search lofi -o url | bili discover - --depth 1
```

## When an edge is gated

Most read endpoints answer anonymous callers, but a few (creator catalogues in
particular) are gated by bilibili's anti-bot for some IPs. The walk treats the
two cases differently:

- A **seed** that cannot be fetched fails the walk, like any bad id.
- An edge that fails **deeper** in the walk becomes a one-line note on stderr and
  the walk carries on with the other edges.

Supplying a logged-in `--cookie` or `BILI_COOKIE` widens what a walk can reach;
`-q` silences the notes.

## discover or crawl?

Both walk the graph from seeds, but they are built for different jobs:

- **`discover`** streams one record per node to stdout. It is for exploring,
  piping, and rendering in any output format. To keep a walk, redirect it:
  `bili discover BV… --depth 2 -o jsonl > graph.jsonl`.
- **`crawl`** writes one JSONL file per record type into a directory, and pulls
  attached data like comments and danmaku. It is for building a split dataset on
  disk. See [crawling](/guides/crawling/).

Reach for `discover` to look around; reach for `crawl` to bulk-dump.
