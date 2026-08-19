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
another with `-p`, which is 1-based and counts parts the way the site labels
them, P1, P2 and so on:

```bash
bili video BVID -o json        # look at the "pages" array for the parts
bili danmaku BVID -p 2         # bullet-chat for the second part
bili streams BVID -p 2         # the streams for that same part
```

`--page` is the global pagination flag and has nothing to do with parts, so it
is `-p` you want here.

## Related videos

```bash
bili related BV17x411w7KC
bili related BV17x411w7KC -o url     # just the watch URLs
```

This is the same list bilibili shows alongside a video, which makes it the
natural edge to follow when [crawling](/guides/crawling/).

## Streams

```bash
bili streams BV17x411w7KC
```

`streams` lists the playable media URLs the API exposes for a part, with their
quality, codec, and format. Use `--quality` to ask for a specific level. These
are the URLs the player would use; `streams` itself downloads nothing and
decrypts nothing, it reports what is offered.

## Download audio

```bash
bili download BV17x411w7KC
bili download BV17x411w7KC --parts 1-3 --format mp3
bili download BV17x411w7KC --format flac --output-dir ~/Music
```

`download` wraps [BBDown](https://github.com/nilaoda/BBDown) rather than
reimplementing it. BBDown already handles the DASH manifest, the stream
selection, the multi part handling and the merge, and a second Go
implementation of all that would be a second thing to maintain that does the
same job worse. What bili adds is the part BBDown does not: whatever you paste
is resolved to a bvid first, so a URL, a bare av number and a bvid all work
here exactly as they do in every other command, and an id for a bangumi or an
audio track is refused by name rather than becoming a BBDown error about a page
that does not exist.

`m4a` is what BBDown emits and involves no second process. `mp3`, `flac` and
`wav` are transcoded with ffmpeg once the download finishes. `--quality` sets
the mp3 bitrate preset, from `best` to `worst`, and is ignored for `flac` and
`wav`, where a lossless format leaves it nothing to mean.

`--parts` takes BBDown's own selection syntax, `1,3-5,LAST` and the like, and
is passed straight through, as are `--file-pattern` and `--multi-file-pattern`.
BBDown writes into a staging directory and finished files are moved into place
from there, so a run that fails halfway leaves nothing behind in the directory
you pointed at.

### The two binaries

Neither BBDown nor ffmpeg is bundled, vendored or installed for you. Both are
looked up on PATH, and `--bbdown-bin` and `--ffmpeg-bin` point at a copy
somewhere else, as do `BILI_BBDOWN_BIN` and `BILI_FFMPEG_BIN`. A missing one is
reported before the transfer starts rather than after it, with the name of the
binary and where to get it, because discovering that ffmpeg is absent at the
end of a forty minute audiobook is a bad way to find out.

`--dry-run` prints the command that would run, quoted so it can be pasted into
a shell, and runs nothing.

BBDown's progress output goes to the terminal as BBDown wrote it. bili does not
parse it into a progress display of its own.

This command downloads audio. It does not download video, and it does not
decrypt anything.

## Bullet-chat

```bash
bili danmaku BV17x411w7KC
```

Each row is one comment that scrolls across the video: its `progress_ms` (the
millisecond offset where it appears), `mode`, `color`, `fontsize`, and `content`.
The data arrives as protobuf and bili decodes it into plain records, so it sorts,
filters, and pipes like anything else:

```bash
bili danmaku BVID -o jsonl | jq -r 'select(.progress_ms < 10000) | .content'
```

See [comments and danmaku](/guides/comments-and-danmaku/) for the conversation
data in depth.
