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

## Checking whether the site changed

What an endpoint needs in order to answer is a fact about bilibili's servers,
not about this tool, and it moves. `bili verify --live` requests every endpoint
in the recorded requirement matrix and prints what each one answered:

```console
$ bili verify --live --rate 1s
x/web-interface/view                         ok
x/v2/reply/wbi/main                          forbidden / signed ok
x/space/upstat                               refused_silent
x/polymer/web-dynamic/v1/feed/space          risk / signed ok
```

Two columns means the endpoint is recorded as needing a WBI signature, so it was
asked both ways: refused without one, answering with one. A single column means
it needs no signature. `refused_silent` is an endpoint that returned the success
code carrying nothing, which is a refusal and not an empty result.

`--strict` exits non-zero when a row no longer matches what is recorded. A row
that refuses is asked again three minutes later before it is reported, because a
burst of requests from one address is the shape risk control exists to slow
down, and a refusal caused by your own pace looks exactly like the site
tightening. Pass `--rate 1s` or slower for a full run.

A row the site rate limited on both passes is reported as not measured rather
than as drift, and does not fail `--strict`. A `-509` says you are asking too
often, which is a fact about your run and not about what the endpoint requires.
`x/article/view` is the row that does this: its penalty window is per address
and cumulative, so a machine that has verified a few times recently will not get
an answer out of it at any pace, and the run should say it did not find out
rather than claim the endpoint changed.

This is the same command the weekly `drift` workflow runs. If it reports drift
and this documentation disagrees with what you are seeing, the drift report is
the one that was measured today.
