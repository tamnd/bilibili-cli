---
title: "Live rooms"
description: "Look up a live room, and browse rooms by area."
weight: 50
---

bilibili Live is a separate surface with its own ids. The `live` command both
looks up a single room and browses the directory.

## A single room

```bash
bili live 5440
bili live https://live.bilibili.com/5440
```

The record carries the room's title, the streamer, the area and parent area, the
online count, the live status (whether it is broadcasting right now), and the
cover. Ask for JSON to get everything:

```bash
bili live 5440 -o json
```

## Browsing by area

With `--browse` instead of a room id, `live` reads the directory. `--area` picks
the parent area, and the rows are the rooms that are live right now:

```bash
bili live --browse --area 1        # rooms in a parent area
bili live --browse --area 1 -n 20  # cap the count
```

`--browse` is required. Called with neither a room id nor `--browse` nor
`--uid`, `live` says so rather than guessing which one you meant.

Each row is a room you can then open directly with `bili live <room>`.
