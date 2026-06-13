---
title: "Introduction"
description: "What bilibili's web API looks like, the parts that make it awkward, and the model bili uses to make it feel small."
weight: 10
---

[bilibili](https://www.bilibili.com) is one of the largest video and community
sites in the world: videos, creators, comments, bullet-chat, dynamics, anime
seasons, audio, columns, and live streams, all behind a web frontend that talks
to a JSON API. That API is public in the sense that your browser uses it every
time you load a page, but it is not built to be called by hand.

bili closes that gap. It is a single binary that treats bilibili the way `curl`
treats a web server: you ask for something by its id or URL, it fetches exactly
that, and it hands you a clean structured record.

## What the API looks like

Almost every endpoint returns the same envelope:

```json
{ "code": 0, "message": "0", "ttl": 1, "data": { ... } }
```

A `code` of `0` means success and `data` carries the payload. A non-zero `code`
is an error, and bili maps the common ones to clear messages (not found, rate
limited, login required, risk control). The bangumi endpoints are the one
exception: they put the payload in `result` instead of `data`, and bili handles
that for you.

## The parts that make it awkward

Calling the API by hand runs into four things bili does for you:

- **The anonymous session.** bilibili expects a set of fingerprint cookies
  (`buvid3`, `buvid4`, and friends) before it will answer most read endpoints.
  bili mints them on first use, so the common case needs no login.
- **WBI signing.** Many endpoints require a signed `w_rid` query parameter
  derived from a pair of rotating keys. bili fetches the keys and signs every
  request that needs it.
- **id forms.** The same thing has several spellings: a video is both an `av`
  number and a `BV` string; a season is `ss`, an episode `ep`, a media page
  `md`. bili accepts any of them, plus full URLs, and normalizes before calling.
- **danmaku.** Bullet-chat is delivered as protobuf, not JSON. bili decodes it
  into plain rows with timing, mode, color, and text.

## What needs a login

A handful of endpoints are gated by bilibili's anti-bot system for anonymous
callers (single-dynamic detail is the main one). When you hit one, bili says so
and tells you to supply a logged-in cookie with `--cookie` or `BILI_COOKIE`.
Everything in the quick start works anonymously.

## What bili is not

bili is a read-only client. It does not log in for you, post, vote, or download
DRM-protected streams; `streams` returns the playable URLs the API exposes, no
more. It reads the public data and shapes it. That narrow scope is what keeps it
a single small binary with no database, no daemon, and no setup.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
