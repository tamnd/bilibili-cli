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
  "cache_dir": "~/Library/Caches/bili",
  "config_dir": "~/Library/Application Support/bili",
  "data_dir": "~/.local/share/bili",
  "cache_ttl": "1h0m0s",
  "rate": "350ms",
  "retries": 4,
  "timeout": "30s",
  "cookie_set": false
}
```

The only thing bili keeps on disk is the response cache, under `cache_dir`. Clear
it with `bili cache clear`; inspect it with `bili cache info`.

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

bili never prints your cookie back: `config show` reports only `cookie_set:
true/false`, and the cookie is never logged. Treat the cookie like a password; it
is your live session.

## Environment variables

| Variable | Used for |
|---|---|
| `BILI_COOKIE` | Cookie header for logged-in endpoints |
| `BILI_CACHE_DIR` | Override the cache directory |
| `BILI_DATA_DIR` | Override the data directory |
| `HTTP_PROXY` / `HTTPS_PROXY` | Standard Go proxy variables, honored by the client |

## Global flags

| Flag | Default | Meaning |
|---|---|---|
| `-o, --output` | auto | `table`, `json`, `jsonl`, `csv`, `tsv`, `yaml`, `url`, `raw` |
| `-n, --limit` | `0` | Maximum records; `0` is unlimited |
| `--fields` | all | Comma-separated columns to keep and order |
| `--template` | none | Go `text/template` applied per record |
| `--no-header` | off | Omit the header row in table/csv output |
| `--page`, `--page-size` | endpoint | Pagination, where the endpoint supports it |
| `--order` | endpoint | Sort order, where supported |
| `--cookie`, `--cookie-file` | none | Logged-in session |
| `--lang` | `zh-CN` | Locale for localized fields |
| `--rate` | `350ms` | Minimum delay between requests, to stay polite |
| `--retries` | `4` | Retry attempts on rate-limit/risk-control/5xx |
| `--cache` / `--no-cache` | on | Use or bypass the on-disk cache |
| `--cache-ttl` | `1h` | Cache freshness window |
| `--proxy` | none | HTTP or SOCKS proxy URL |
| `--dry-run` | off | Print the requests that would be made, without calling |
| `--color` | auto | `auto`, `always`, or `never` |
| `-q, --quiet` | off | Suppress progress output on stderr |

## Caching and politeness

bili caches API responses on disk for `--cache-ttl` (one hour by default) so
repeated commands and overlapping crawls do not re-fetch the same data. `--rate`
keeps a minimum gap between requests so a busy session stays a good citizen
against the public API, and `--retries` backs off and retries the rate-limit and
risk-control responses bilibili returns when you go too fast.

## Output auto-detection

The default output format adapts to where it is going: an aligned table when the
output is a terminal, JSONL when it is piped. That keeps interactive use readable
and scripted use parseable without you setting `-o` either way. See
[output formats](/reference/output/) for the full set.
