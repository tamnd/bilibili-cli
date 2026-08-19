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

## "rate limited" (-509)

You are going too fast. bili backs off and retries this one, and keeps a
`--rate` gap between calls, but a tight loop on an aggressively throttled
endpoint (search and a creator's video list are the touchy ones) can still
outrun it. Raise `--rate`, lower concurrency, or let the cache absorb repeats. A
single transient hit usually clears on the built-in retry, and one that outlives
the retries exits 6, which is the code that clears by waiting.

## "intercepted" (-412)

Despite the number, this is risk control and not a rate limit. It is the
envelope form of the HTTP 412 interstitial, it exits 2, and it is never
retried: retrying an address that has just been turned away is the worst
available response to it. A logged-in cookie is what usually clears it. Note
that `--user-agent` makes this more likely rather than less, because risk
control reads the platform token out of the User-Agent and most overrides fail
it.

## "login required" (-101)

The endpoint needs a session even though it is not anti-bot gated. Supply a
cookie as above. Note that `bili nav` deliberately works while anonymous: it
reports `is_login: false` and still returns the live WBI keys, which is normal.

## "not found" (-404)

The object does not exist, or was removed or made private. Double-check the id
with `bili id <thing>`, which shows how bili classified it. A video that was taken
down returns -404 even though the BV id is well-formed.

## "refused" on an endpoint that returned success

This is bilibili answering `code: 0` with the message `OK` and no payload at
all, which is a refusal wearing a success code. bili names it rather than
handing you an empty list:

```console
$ bili favorites 946974
favorites 946974: refused: this endpoint always carries a payload when it
answers, so this is a refusal and not an empty result. It refuses anonymous
callers, and only a logged-in cookie via --cookie or BILI_COOKIE changes that.
Reading the folder directly by its id used to be the way around this and is not
any more, so this needs a logged-in cookie rather than a different command
(code 0 with no payload)
[api.bilibili.com/x/v3/fav/folder/created/list-all]
```

Two endpoints do this today. `x/v3/fav/folder/created/list-all`, behind `bili
favorites <mid>`, and `x/space/upstat`, which carries a creator's total views
and likes. Both answer normally with a logged-in cookie. The second one does not
produce an error of its own, because it is one part of a record the rest of
which arrived: see "A count that came back as nothing" below.

An older version of this page said an empty favorites list was the owner's
privacy setting. That was wrong. The endpoint answers the same way for every
mid measured, public folders included.

That paragraph also said reading one folder directly by its `ml` id still works
while anonymous, and on 2026-08-19 that stopped being true. The folder read
answers with the folder's title, owner and `media_count`, and with `medias:
null` and `has_more: true` beside them, which is the contents withheld rather
than a folder with nothing in it. bili reads the count it was given and says so:

```console
$ bili favorite ml1952291424 >/dev/null; echo $?
4
```

The count is in the message, because a reader told that the folder holds 145
items and sent none of them does not have to take the tool's word for the
classification. Both reads answer normally with a logged-in cookie.

The distinction bili is drawing here is not visible in the response. A folder
with nothing in it and an endpoint refusing to tell you both have no records in
them. It is drawn from a table of which endpoints carry a payload when they
answer, which is measured rather than assumed, and which `bili verify --live`
re-measures weekly.

## A count that came back as nothing

`bili user` is four requests behind one row. `acc/info` carries the identity,
and the follower, upload, view and like counts come from three other endpoints
with three other gates. Any of them can refuse while the rest answer, and one of
them refuses anonymous callers every time: `x/space/upstat`, which is where a
creator's total views and likes live and nowhere else.

Those counts used to print as `0`, which is the same lie as an empty list that
means refused, one field down. They are now left out of the record and named in
the envelope with what stopped them:

```console
$ bili user 946974 -o jsonl | jq '{video_count, total_view, missed: .envelope.missed}'
{
  "video_count": 924,
  "total_view": null,
  "missed": {
    "total_like": "x/space/upstat refused_silent: code 0 with no payload",
    "total_view": "x/space/upstat refused_silent: code 0 with no payload"
  }
}
```

In a table or a csv the cell is empty rather than `0`, which is the difference
that matters when something downstream is going to sum the column. A creator
whose totals really are zero still gets a zero.

`video_count` is worth watching in that output. It comes from the listing
endpoint's own pagination total, and `x/space/wbi/arc/search` is risk-gated
intermittently for anonymous callers, so the same command a minute later can
show it in `missed` instead. That is a real property of the site, and it is only
visible because the two outcomes no longer look the same.

The whole command still exits 0. A record that arrived with a field missing is
not a failed read, and the envelope is where you go to find out what it cost.

## A feed that returns nothing

`bili dynamics <mid>` reads a paginated feed, and a paginated feed has two ways
to produce no items: it ran out, or the page was refused. They look identical
from the outside, so bili reads what the server said rather than counting what
it sent.

A page carrying no items ends the feed only when the server also said
`has_more: false` and only after an earlier page produced something. A page that
promised more and then sent none is a refusal, and so is a first page that
carried nothing at all, because this endpoint refuses anonymous callers far more
often than a creator posts nothing:

```console
$ bili dynamics 946974 >/dev/null; echo $?
4
```

A creator who has genuinely never posted looks the same from here, which is why
the message says so and asks for a cookie rather than asserting that the feed is
empty. With a logged-in cookie the same read returns the feed.

## A walk that came back short

`bili discover` meets refusals as a matter of course: it follows edges across
many endpoints and some of them are gated for some addresses. A gated edge is a
note on stderr and the walk carries on, and the notes are counted by kind when
it finishes:

```console
$ bili discover BV1gtgE6AEmZ --depth 2 -o jsonl > graph.jsonl
bili: note: risk control: ... [api.bilibili.com/x/space/wbi/arc/search]
bili: reached 214 nodes past the seeds, 3 risk
```

That run exits 0, because it reached the graph it was asked for and said what it
could not see. Only a walk where every edge was refused, so nothing past the
seeds was reached, exits with the refusal's code. A walk refused in more than
one way exits 1, since no single state describes it.

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

## Telling the results apart in a script

Every failure on this page has its own exit code, so a loop does not have to
read stderr to know what happened:

```bash
for id in $(cat ids.txt); do
  bili video "$id" -o jsonl >> out.jsonl
  case $? in
    0) ;;
    3) echo "$id: nothing there" ;;
    6) sleep 300 ;;          # rate limited, and this one clears by waiting
    2) echo "risk control, get a cookie"; break ;;
    *) echo "$id: failed" ;;
  esac
done
```

The full table is in the [CLI reference](/reference/cli/#exit-codes). The three
worth branching on are 2, which says stop and get a cookie, 6, which says sleep
and try again, and 3, which says this one is genuinely empty and the next one is
still worth asking for.

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
