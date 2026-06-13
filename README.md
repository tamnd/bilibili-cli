# bili

A fast, friendly command line for [bilibili.com](https://www.bilibili.com). One
binary that resolves any video, user, comment, danmaku, dynamic, live room,
bangumi, audio track, column article, or favorite folder into clean structured
records you can pipe anywhere.

```
bili video BV17x411w7KC --fields bvid,title,owner_name,view_count,pubdate_text
```

```
bvid          title               owner_name  view_count  pubdate_text
BV17x411w7KC  【MV】保加利亚妖王AZIS视频合辑  冰封.虾子       45703435    2011-11-09 21:55:33
```

Full documentation: [bilibili-cli.tamnd.com](https://bilibili-cli.tamnd.com).

## Why

Working with bilibili usually means reverse engineering its web API: the WBI
request signing, the anonymous `buvid` session it expects, the `{code, message,
data}` envelope, av/BV id conversion, and protobuf-encoded danmaku. bili puts all
of it behind one tool with sensible defaults, real output formats, and pipelines
that compose. It talks to the public web endpoints over plain HTTPS, signs every
request that needs it, and bootstraps an anonymous session on its own, so there
are no credentials to set up for public data.

A cookie is optional. Provide one with `--cookie` or `--cookie-file` and bili
reaches the same data your logged-in browser sees; without one it stays anonymous.
bili is read-only: it never logs in for you, posts, or downloads protected media.

## Install

```sh
go install github.com/tamnd/bilibili-cli/cmd/bili@latest
```

Or grab a prebuilt binary from the [releases page](https://github.com/tamnd/bilibili-cli/releases).
The binary is pure Go with no runtime dependencies.

Build from source:

```sh
git clone https://github.com/tamnd/bilibili-cli
cd bilibili-cli
make build      # produces ./bili
```

## Quick start

```sh
bili video BV17x411w7KC               # full metadata for a video
bili comments BV17x411w7KC -n 50      # the top comments, with their replies
bili danmaku BV17x411w7KC             # bullet-chat for the first part
bili search lofi                      # search videos
bili user 122541                      # a creator's profile and stat
bili search lofi -o url | bili crawl - --out ./data --comments
```

## How it works

bilibili exposes a large web API behind a few shared conventions. bili handles
them so you do not have to:

- **WBI signing.** Many endpoints reject unsigned requests. bili fetches the
  current WBI key pair from the nav endpoint, derives the mixin key, and signs
  each call with the right `w_rid` and `wts`.
- **Anonymous session.** bili activates a fresh `buvid3`/`buvid4` pair on first
  use so endpoints that expect a browser session answer normally.
- **The envelope.** Responses arrive as `{code, message, ttl, data}` (or `result`
  for the bangumi/pgc endpoints). bili unwraps it, maps bilibili's risk-control
  codes to readable errors, and gives you just the record.
- **ids.** Videos carry both an `avNNN` number and a `BV` string; bili converts
  between them, follows `b23.tv` short links, and classifies any id or URL you
  paste with `bili id`.
- **danmaku.** Bullet chat ships as a protobuf segment stream. bili decodes it
  into one record per comment with its timestamp, mode, color, and text.

## Commands

| Command | What it does |
| --- | --- |
| `video` | Resolve one or more videos to full metadata |
| `related` | Related videos for a video |
| `streams` | Playable stream URLs for a video part |
| `danmaku` | Bullet-chat (danmaku) for a video part |
| `comments` | Every comment and reply on a video, article, audio, or dynamic |
| `user` | A creator's profile, catalogue, stat, or dynamics |
| `dynamic` / `dynamics` | One feed post, or a user's whole dynamics feed |
| `favorite` / `favorites` | A favorite folder's videos, or a user's folders |
| `bangumi` | An anime/film season with every episode |
| `audio` | An audio track's metadata and stat |
| `article` | A column article's metadata and text |
| `live` | Live room info, or browse rooms by area |
| `search` | Search videos, users, bangumi, live rooms, or articles |
| `suggest` / `trending` | Search autosuggest terms, and current hot searches |
| `popular` / `rank` | The popular feed or a weekly issue, and the leaderboards |
| `id` | Classify and normalize any id or URL |
| `crawl` | Crawl connected records from seed ids into JSONL files |
| `nav` | Login state and current WBI keys (debug) |
| `config` | Show resolved configuration and important paths |
| `cache` | Inspect and clear the on-disk response cache |

Run `bili <command> --help` for the full flag list on any command.

## Recipes

Pull a video and all of its top comments with replies as JSONL:

```sh
bili comments BV17x411w7KC --replies -o jsonl > comments.jsonl
```

Collect the danmaku timeline for a multi-part video's third part:

```sh
bili danmaku BV17x411w7KC --page 3 -o jsonl | wc -l
```

Turn a search into a small dataset, comments and uploader profiles included:

```sh
bili search 'lofi' -n 20 -o url \
  | bili crawl - --out ./data --comments --user
```

Walk a creator's whole catalogue:

```sh
bili user 122541 --videos -o jsonl
```

See what an id or URL actually is before fetching it:

```sh
bili id https://b23.tv/abc123        # follows the short link and classifies it
```

Inspect the exact requests a command would make, without sending them:

```sh
bili video BV17x411w7KC --dry-run
```

## Output formats

Every command renders through the same formatter. Pick a format with `-o`, or let
bili choose: a table when writing to a terminal, JSONL when piped.

```sh
bili search lofi -o table   # aligned columns for reading
bili search lofi -o jsonl   # one JSON object per line, for piping
bili search lofi -o json    # a single JSON array
bili search lofi -o csv     # spreadsheet friendly
bili search lofi -o url     # just the canonical URL column
```

Narrow the columns with `--fields`, or template each row:

```sh
bili video BV17x411w7KC --fields bvid,title,view_count,like_count
bili search lofi --template '{{.BVID}} {{.Title}}'
```

## Configuration

bili keeps its cache and state under your platform's standard cache and config
directories. See the resolved paths and effective settings any time (secrets are
redacted):

```sh
bili config show
```

Useful global flags (all have sensible defaults):

| Flag | Meaning |
| --- | --- |
| `--cookie`, `--cookie-file` | Send a cookie to reach logged-in data |
| `-o, --output` | Output format (default auto) |
| `-n, --limit` | Maximum records emitted (`0` means unlimited) |
| `--fields` | Comma-separated columns to keep and order |
| `--rate` | Minimum delay between requests, to stay polite |
| `--lang` | Locale for localized fields (default `zh-CN`) |
| `--proxy` | HTTP or SOCKS proxy URL |
| `--no-cache` | Bypass the on-disk cache |
| `--dry-run` | Print the requests that would be made |

A cookie is read only and never printed back: `bili config show` redacts it, and
nothing logs it.

## Development

```sh
make test    # run the test suite
make vet     # go vet
make build   # build ./bili
make smoke   # build, then run the live smoke script
```

The code is layered. `cli/` is the command tree built on Cobra. `bili/` is the
library it sits on: the HTTP client, WBI signing, the anonymous session, id
resolution, and one method per data model. Two small, self-contained packages
live under `pkg/`, each importable on its own:

| Package | What it does |
| --- | --- |
| `pkg/bvconv` | Convert between `avNNN` numbers and `BV` strings |
| `pkg/dmproto` | Decode the protobuf danmaku segment stream |

Neither depends on `bili/` or the CLI, so you can pull just the piece you need
into your own program.

## License

[Apache 2.0](LICENSE).

This project is an independent client and is not affiliated with bilibili. Use it
to access public data responsibly and within bilibili's terms of service.
