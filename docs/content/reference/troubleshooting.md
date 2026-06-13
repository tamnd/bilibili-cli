---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to bilibili's anti-bot and rate-limit behavior, not a
bug. bili reports the API's error code and a hint for the common ones.

## "risk control" (-352)

A few endpoints are gated by bilibili's anti-bot system for anonymous callers,
single-dynamic detail (`bili dynamic <id>`) being the main one. The fix is to
supply a logged-in session:

```bash
export BILI_COOKIE='SESSDATA=...; bili_jct=...; DedeUserID=...'
bili dynamic <id>
```

See [configuration](/reference/configuration/) for how cookies are supplied. This
is an account/IP gate, not something a different request shape gets around.

## "rate limited" (-509) or "intercepted" (-412)

You are going too fast. bili already backs off and retries these, and keeps a
`--rate` gap between calls, but a tight loop on an aggressively throttled endpoint
(search and a creator's video list are the touchy ones) can still hit them. Raise
`--rate`, lower concurrency, or let the cache absorb repeats. A single transient
hit usually clears on the built-in retry.

## "login required" (-101)

The endpoint needs a session even though it is not anti-bot gated. Supply a
cookie as above. Note that `bili nav` deliberately works while anonymous: it
reports `is_login: false` and still returns the live WBI keys, which is normal.

## "not found" (-404)

The object does not exist, or was removed or made private. Double-check the id
with `bili id <thing>`, which shows how bili classified it. A video that was taken
down returns -404 even though the BV id is well-formed.

## A favorites list comes back empty

Favorite folders are private by default site-wide. `bili favorites <mid>` on an
account that keeps them private returns an empty list, not an error. That is the
owner's privacy setting.

## Localized fields are in Chinese

Many fields (titles, area names, descriptions) are authored in Chinese and bili
passes them through verbatim; it does not translate. `--lang` sets the locale
sent to endpoints that localize, but most content is whatever the uploader wrote.

## Checking what bili resolved

When something behaves unexpectedly, `bili id <thing>` shows how an id was
classified, `bili nav` shows the session and WBI key state, and `bili config
show` prints the resolved paths and settings. `--dry-run` prints the exact
requests a command would make without sending them, which is the quickest way to
see what bili is about to do.
