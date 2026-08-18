---
title: "CLI"
description: "The full command tree and every flag, grouped by what each command does."
weight: 10
---

Run `bili <command> --help` for the live flag list on any command; this page is
the map. Every command accepts the [global flags](#global-flags) and renders
through the shared [output formatter](/reference/output/).

## Resolving

| Command | Argument | What it does |
|---|---|---|
| `id <thing>` | any id or URL | Classify and normalize an id or URL, and print its canonical forms |
| `video <id>...` | `BV`/`av`/URL, or `-` | Resolve one or more videos to full metadata |
| `related <id>` | a video | Related videos for a video |
| `streams <id>` | a video | Playable stream URLs for a video part |
| `danmaku <id>` | a video | Bullet-chat (danmaku) for a video part |

## Conversation

| Command | Argument | What it does |
|---|---|---|
| `comments <id>` | video, article, audio, or dynamic | Every comment and reply on an object |

## Creators

| Command | Argument | What it does |
|---|---|---|
| `user <mid>` | `mid` or space URL | A creator's profile; `--videos` or `--dynamics` to pivot |
| `favorites <mid>` | `mid` or space URL | A creator's favorite folders |
| `favorite <ml>` | `ml` id or URL | The videos inside one favorite folder |
| `dynamics <mid>` | `mid` or space URL | A creator's whole dynamics feed |
| `dynamic <id>` | a dynamic | One dynamic post in full (may need a cookie) |

## Catalogue

| Command | Argument | What it does |
|---|---|---|
| `bangumi <id>` | `ss`/`ep`/`md` or URL | An anime/film season with every episode |
| `audio <au>` | `au` id or URL | An audio track's metadata and stats |
| `article <cv>` | `cv` id or URL | A column article's metadata; `--text` for the body |
| `live <room>` | room id or URL | A live room, or browse rooms with `--area` |

## Discovery

| Command | Argument | What it does |
|---|---|---|
| `search <query>` | text | Search videos, users, bangumi, live rooms, or articles |
| `suggest <term>` | text | Search autosuggest terms |
| `trending` | — | Current hot-search terms |
| `popular` | — | The popular feed, or a weekly selection issue |
| `rank` | — | The leaderboard, optionally for one partition |

## Datasets

| Command | Argument | What it does |
|---|---|---|
| `discover <id>...` | seeds, or `-` | Breadth-first walk from a video or creator; `--follow content\|creators\|all` or an edge list, `--depth`, `--fanout`. Aliases: `walk`, `graph` |
| `crawl <id>...` | seeds, or `-` | Walk the graph from seeds into per-type JSONL files |

## Utility

| Command | What it does |
|---|---|
| `nav` | Login state and current WBI keys (debug) |
| `verify --live` | Re-measure the endpoint requirement matrix against the live API |
| `config show` | Print resolved configuration and important paths |
| `cache info` / `cache clear` | Inspect or clear the on-disk response cache |
| `version` | Print version, commit, and build date |
| `completion <shell>` | Generate a shell completion script |

## Global flags

Available on every command:

| Flag | Default | Meaning |
|---|---|---|
| `-o, --output` | auto | `table`, `json`, `jsonl`, `csv`, `tsv`, `yaml`, `url`, `raw` |
| `-n, --limit` | `0` | Maximum records; `0` is unlimited |
| `--fields` | all | Comma-separated columns to keep and order |
| `--template` | none | Go `text/template` applied per record |
| `--no-header` | off | Omit the header row in table/csv output |
| `--page` | endpoint | Start page where the endpoint paginates |
| `--page-size` | endpoint | Page size where the endpoint paginates |
| `--order` | endpoint | Sort order where supported |
| `--cookie` | none | Cookie header for logged-in endpoints |
| `--cookie-file` | none | Path to a file holding the cookie header |
| `--lang` | `zh-CN` | Locale for localized fields |
| `--rate` | `350ms` | Minimum delay between requests |
| `--retries` | `4` | Retry attempts on 429 and 5xx. A risk refusal is never retried |
| `--cache` / `--no-cache` | on | Use or bypass the on-disk response cache |
| `--cache-ttl` | `1h` | Cache freshness window |
| `--proxy` | none | HTTP or SOCKS proxy URL |
| `--dry-run` | off | Print the requests that would be made, without calling |
| `--color` | auto | `auto`, `always`, or `never` |
| `-q, --quiet` | off | Suppress progress output on stderr |

## Exit codes

A refusal and an empty answer are different results, so they get different
codes. A script can act on the difference without reading stderr.

| Code | Meaning |
|---|---|
| 0 | it did what it was asked |
| 1 | this tool could not do what was asked and cannot say why: a flag or argument it does not understand, a response it could not classify, or a run interrupted part way |
| 2 | risk control refused the request, as a `-352` or an HTTP 412. Retrying will not clear it, a logged-in cookie usually will |
| 3 | the request succeeded and there was genuinely nothing to return |
| 4 | the API returned a success code carrying no payload, on an endpoint that always carries one. This is a refusal wearing a success code, and it is described in [troubleshooting](/reference/troubleshooting/) |
| 5 | a network failure, a timeout, or a 5xx that outlived the retries |
| 6 | rate limited: a `-509` or a 429 that outlived the retries. This one clears by waiting |
| 7 | not found, either as a `-404` or as the private constant an endpoint uses instead |

```console
$ bili audio au1 >/dev/null; echo $?
7
$ bili favorites 946974 >/dev/null; echo $?
4
$ bili video BV1xx411c7mD --tags >/dev/null; echo $?
3
```

Three of those are worth branching on in a loop: 2 says stop and get a cookie, 6
says sleep and try again, and 3 says this one is genuinely empty and the next
one is worth asking for.

## Runs of many

`bili video` takes any number of identifiers, and so does `-` on stdin. The exit
codes describe the run rather than one target:

- a failure part way through does not stop the rest
- every failure is named on stderr as it happens, and the counts follow
- a status becomes the run's exit code only when it covers every target
- a run where every target failed differently exits 1, because no single status
  describes it

One refused folder listing in five hundred is not a refused run, so it exits 0
and says so on stderr.
