# bili

[![CI](https://github.com/tamnd/bilibili-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/tamnd/bilibili-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tamnd/bilibili-cli)](https://github.com/tamnd/bilibili-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/tamnd/bilibili-cli.svg)](https://pkg.go.dev/github.com/tamnd/bilibili-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/tamnd/bilibili-cli)](https://goreportcard.com/report/github.com/tamnd/bilibili-cli)
[![License](https://img.shields.io/github/license/tamnd/bilibili-cli)](./LICENSE)

A command line for [bilibili.com](https://www.bilibili.com). `bili` resolves
any video, user, comment, danmaku, dynamic, live room, bangumi, audio, article,
or favorite folder into clean structured records. One pure-Go binary, no API
key, no login.

[Install](#install) • [Commands](#commands) • [Usage](#usage) • [How it works](#how-it-works)

![bili searching bilibili and reading a video record from the command line](docs/static/demo.gif)

It talks to the public bilibili web endpoints over plain HTTPS: WBI signing,
anonymous `buvid` session bootstrap, `{code, message, data}` envelope
unwrapping, av/BV id conversion, and protobuf danmaku decoding are all handled
for you. A cookie is optional — pass `--cookie` and `bili` reaches the same data
your logged-in browser sees.

`bili` is an independent tool. It is not affiliated with bilibili.

## Install

```bash
go install github.com/tamnd/bilibili-cli/cmd/bili@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/bilibili-cli/releases),
or run the container image:

```bash
docker run --rm ghcr.io/tamnd/bili:latest search 'lofi' -n 10
```

Shell completion is built in: `bili completion bash|zsh|fish|powershell`.

## Commands

| Command | Reads |
| --- | --- |
| `bili video <id\|url>...` | one or more videos; full metadata |
| `bili related <id\|url>` | related videos for a video |
| `bili streams <id\|url>` | playable stream URLs for a video part |
| `bili danmaku <id\|url>` | bullet-chat for a video part; `--page` |
| `bili comments <id\|url>` | every comment and reply on a video, article, audio, or dynamic |
| `bili user <mid\|url>` | a creator's profile, catalogue, stat, or dynamics; `--videos`, `--dynamics` |
| `bili dynamic <id\|url>` | one dynamic post in full |
| `bili dynamics <mid\|url>` | a user's whole dynamics feed |
| `bili favorite <ml\|url>` | the videos inside a favorite folder |
| `bili favorites <mid\|url>` | a user's favorite folders |
| `bili bangumi <ss\|ep\|md\|url>` | an anime or film season with every episode |
| `bili audio <au\|url>` | an audio track's metadata and stat |
| `bili article <cv\|url>` | a column article's metadata and text |
| `bili live <room\|url>` | live room info, or browse rooms by area |
| `bili search <query>` | search videos, users, bangumi, live rooms, or articles; `--type` |
| `bili suggest <term>` | search autosuggest terms |
| `bili trending` | current hot search terms |
| `bili popular` | the popular feed, or a weekly selection issue |
| `bili rank` | the leaderboard, optionally for one partition |
| `bili id <thing>` | classify and normalize any id or URL |
| `bili crawl <id\|url>...` | crawl connected records from seed ids into JSONL files |
| `bili nav` | login state and current WBI keys (debug) |
| `bili config show` | resolved configuration and paths |
| `bili cache path\|info\|clear` | inspect or clear the on-disk cache |
| `bili version` | print version, commit, and build date |

Full reference and guides live at [bilibili-cli.tamnd.com](https://bilibili-cli.tamnd.com).

## Usage

```bash
bili video BV17x411w7KC                    # full video metadata
bili comments BV17x411w7KC -n 50           # top comments with replies
bili danmaku BV17x411w7KC                  # bullet-chat for the first part
bili search 'lofi' -n 20                   # search videos
bili user 122541                           # a creator's profile
bili bangumi ss12548                       # an anime season
bili rank --partition dance                # dance leaderboard
```

Records come out as a table (the default on a terminal), JSON, JSONL, CSV, TSV,
url, or raw:

```bash
bili video BV17x411w7KC --fields bvid,title,view_count,like_count -o table
bili search 'lofi' -o jsonl | jq '{bvid, title, view_count}'
bili search 'lofi' -o url
bili user 122541 --videos -o jsonl
bili comments BV17x411w7KC --replies -o jsonl > comments.jsonl
```

Crawl a search result and pull comments and uploader profiles for each hit:

```bash
bili search 'lofi' -n 20 -o url \
  | bili crawl - --out ./data --comments --user
```

### Global flags

```
-o, --output       table|json|jsonl|csv|tsv|yaml|url|raw   (auto: table on a TTY, jsonl when piped)
    --fields       comma-separated columns to keep, in order
    --no-header    omit the header row
    --template     Go text/template applied per record
-n, --limit        max records (0 = unlimited)
    --cookie       cookie header (SESSDATA=...; ...)
    --cookie-file  path to a Netscape cookie file
    --lang         locale for localized fields (default zh-CN)
-q, --quiet        suppress progress output
    --color        auto|always|never
    --rate         min spacing between requests (default 350ms)
    --timeout      per-request timeout (default 30s)
    --retries      retry attempts on 429/-412/5xx (default 4)
-j, --workers      concurrency for fan-out commands (default 4)
    --no-cache     bypass the on-disk cache
    --cache-ttl    cache freshness window (default 1h)
    --dry-run      print the requests that would be made
    --proxy        HTTP or SOCKS proxy URL
```

## How it works

bilibili's public API is behind a few shared conventions. `bili` handles them so
you do not have to:

**WBI signing.** Many endpoints reject unsigned requests. `bili` fetches the
current WBI key pair from the nav endpoint, derives the mixin key, and signs
each call with `w_rid` and `wts`.

**Anonymous session.** `bili` activates a fresh `buvid3`/`buvid4` pair on first
use so endpoints that expect a browser session answer normally.

**The envelope.** Responses arrive as `{code, message, ttl, data}` (or `result`
for bangumi endpoints). `bili` unwraps it and maps bilibili's risk-control codes
to readable errors.

**IDs.** Videos carry both an `avNNN` number and a `BV` string. `bili` converts
between them, follows `b23.tv` short links, and classifies any id or URL you
paste with `bili id`.

**Danmaku.** Bullet chat ships as a protobuf segment stream. `bili` decodes it
into one record per comment with its timestamp, mode, color, and text.

Two importable packages ship alongside the CLI:

| Package | Does |
| --- | --- |
| `pkg/bvconv` | Convert between `avNNN` numbers and `BV` strings |
| `pkg/dmproto` | Decode the protobuf danmaku segment stream |

## Exit codes

```
0  success
1  error
2  usage error
3  no results
```

## Development

```
cmd/bili/    thin main entry point
cli/         cobra commands and output rendering
bili/        HTTP client, WBI signing, session bootstrap, id resolution, models
pkg/bvconv/  av/BV id conversion (no dependencies)
pkg/dmproto/ protobuf danmaku decoder (no dependencies)
docs/        documentation site (Hugo, tago-doks theme)
```

```bash
make build   # ./bili
make test    # go test ./...
make vet     # go vet ./...
make smoke   # build + live smoke script
```

Requires Go 1.23+.

## Releasing

Push a version tag and GitHub Actions runs GoReleaser:

```bash
git tag -a v0.2.0 -m "v0.2.0"
git push --tags
```

The image tag carries no `v` prefix (`ghcr.io/tamnd/bili:0.2.0`).

## License

Apache-2.0. See [LICENSE](LICENSE).

`bili` is an independent client. Use it to access public data responsibly and
within bilibili's terms of service.
