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
| `crawl <id>...` | seeds, or `-` | Walk the graph from seeds into per-type JSONL files |

## Utility

| Command | What it does |
|---|---|
| `nav` | Login state and current WBI keys (debug) |
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
| `--retries` | `4` | Retry attempts on rate-limit/risk-control/5xx |
| `--cache` / `--no-cache` | on | Use or bypass the on-disk response cache |
| `--cache-ttl` | `1h` | Cache freshness window |
| `--proxy` | none | HTTP or SOCKS proxy URL |
| `--dry-run` | off | Print the requests that would be made, without calling |
| `--color` | auto | `auto`, `always`, or `never` |
| `-q, --quiet` | off | Suppress progress output on stderr |
