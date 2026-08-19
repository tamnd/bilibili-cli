---
title: "Configuration"
description: "The data directories, cookies, environment variables, and global flags, with their defaults."
weight: 20
---

bili needs no configuration to run. There is no config file; every option is a
flag or an environment variable, and the defaults are chosen so the common case
needs neither. It runs anonymously against `api.bilibili.com` over HTTPS.

## Directories

bili follows the XDG base directory layout, so its cache, config, and data each
live in the standard place for your OS. See the resolved paths any time:

```bash
bili config show
```

```json
{
  "cache_dir": "/Users/you/Library/Caches/bili",
  "cache_ttl": "1h0m0s",
  "config_dir": "/Users/you/Library/Application Support/bili",
  "cookie": "",
  "cookie_set": false,
  "data_dir": "/Users/you/.local/share/bili",
  "proxy": "",
  "rate": "350ms",
  "retries": 4,
  "timeout": "30s",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ..."
}
```

`bili config path` prints just the three directories, which is the form to read
from a script. The only thing bili keeps on disk is the response cache, under
`cache_dir`. Empty it with `bili cache clear`, and `bili cache stat` reports
where it is, how many files it holds, and how large it is.

## Cookies

Most read endpoints work anonymously. A few are gated by bilibili's anti-bot
system for anonymous callers (single-dynamic detail is the main one) and need a
logged-in session. Supply it as a cookie header, the same string your browser
sends:

```bash
# inline
bili dynamic <id> --cookie 'SESSDATA=...; bili_jct=...; DedeUserID=...'

# from the environment (preferred, so it stays out of your shell history)
export BILI_COOKIE='SESSDATA=...; bili_jct=...; DedeUserID=...'
bili dynamic <id>

# from a file
bili dynamic <id> --cookie-file ~/.bili-cookie
```

bili never prints your cookie back in full. `config show` reports `cookie_set:
true` and a redacted `cookie` with three characters of each value kept, enough
to tell two sessions apart and not enough to use, and the cookie is never
logged. Treat it like a password, because it is your live session.

## Environment variables

| Variable | Used for |
|---|---|
| `BILI_COOKIE` | Cookie header for logged-in endpoints |
| `BILI_COOKIE_FILE` | Path to a cookie file, the environment form of `--cookie-file` |
| `BILI_CACHE_DIR` | Override the cache directory |
| `XDG_DATA_HOME` | Moves the data directory, which lives at `$XDG_DATA_HOME/bili` |
| `BILI_OUTPUT` | Default output format, used when `-o` is left at `auto` |
| `BILI_PROXY` | Proxy URL, the environment form of `--proxy` |
| `BILI_USER_AGENT` | Override the User-Agent, the environment form of `--user-agent` |
| `BILI_BBDOWN_BIN` | Path to BBDown, for `bili download` |
| `BILI_FFMPEG_BIN` | Path to ffmpeg, for `bili download --format mp3\|flac\|wav` |
| `HTTP_PROXY` / `HTTPS_PROXY` | Standard Go proxy variables, honored by the client |

A flag always wins over the matching variable. `BILI_CACHE_DIR` is read once at
startup, so changing it mid-session has no effect on a running command.

## Global flags

| Flag | Default | Meaning |
|---|---|---|
| `-o, --output` | auto | `list`, `table`, `markdown`, `json`, `jsonl`, `csv`, `tsv`, `url`, `raw` |
| `-n, --limit` | `0` | Maximum records; `0` is unlimited |
| `--fields` | all | Comma-separated columns to keep and order |
| `--template` | none | Go `text/template` applied per record |
| `--no-header` | off | Omit the header row in table/csv output |
| `--page`, `--page-size` | endpoint | Pagination, where the endpoint supports it |
| `--order` | endpoint | Sort order, where supported |
| `--cookie`, `--cookie-file` | none | Logged-in session |
| `--lang` | `zh-CN` | Locale for localized fields |
| `--rate` | `350ms` | Minimum delay between requests, to stay polite |
| `--retries` | `4` | Retry attempts on 429 and 5xx. A risk refusal is never retried |
| `--cache` / `--no-cache` | on | Use or bypass the on-disk cache |
| `--cache-ttl` | `1h` | Cache freshness window |
| `--timeout` | `30s` | Per-request timeout |
| `-j, --workers` | `4` | Concurrency for the commands that fan out |
| `--proxy` | none | HTTP or SOCKS proxy URL |
| `--user-agent` | a desktop Chrome string | Override the User-Agent |
| `--raw` | off | Print each record as pretty-printed JSON, whatever `-o` says |
| `--dry-run` | off | Print the requests that would be made, without calling |
| `--color` | auto | `auto`, `always`, or `never` |
| `-q, --quiet` | off | Suppress progress output on stderr |
| `-v, --verbose` | off | More detail on stderr; repeatable |
| `-y, --yes` | off | Assume yes to prompts |

## Caching and politeness

bili caches API responses on disk for `--cache-ttl` (one hour by default) so
repeated commands and overlapping crawls do not re-fetch the same data. `--rate`
keeps a minimum gap between requests so a busy session stays a good citizen
against the public API. `--retries` backs off and retries a 429 and a 5xx, which
are a server saying it is busy. It does not retry a risk control refusal: an
HTTP 412 and a `-352` are one address being turned away, and asking again four
times is the worst possible answer to that.

## Output auto-detection

The default output format adapts to where it is going: an aligned table when the
output is a terminal, JSONL when it is piped. That keeps interactive use readable
and scripted use parseable without you setting `-o` either way. See
[output formats](/reference/output/) for the full set.
