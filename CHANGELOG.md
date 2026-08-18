# Changelog

Written for somebody deciding whether to upgrade. Behaviour changes come first
in every release, because those are the ones that can break a script that
already works.

## v0.3.0

The release is about one idea: a refusal must never look like a result. Most of
what changed follows from applying that idea everywhere it was not being
applied.

### Behaviour changes

**Exit codes.** v0.2.0 exited 0 or 1. v0.3.0 exits 0 through 7, and the code
says what happened: 2 for a risk control refusal, 3 for a genuinely empty
result, 4 for a refusal wearing a success code, 5 for the network, 6 for rate
limiting, 7 for not found, 1 for everything it could not name. Anything written
as `if bili ...; then` still works, because 0 still means it worked. Anything
testing `$? -eq 1` needs the table in the
[CLI reference](https://bilibili-cli.tamnd.com/reference/cli/#exit-codes).

The three worth branching on are 2, which says stop and get a cookie, 6, which
says sleep and try again, and 3, which says this one is empty and the next one
is still worth asking for.

**A refusal is no longer an empty result.** `bili favorites <mid>` used to print
`[]` and exit 0 on a creator whose folder listing the API had declined to send.
It now names the endpoint, says a logged-in cookie is what changes it, and exits
4. If you built on the old behaviour this is a break, and it is also the bug
this release exists to fix. The same correction applies to a dynamics feed that
returns no items and to a favorites folder that reports a count and sends
nothing.

**Absent counts are absent rather than zero.** `video_count`, `total_view` and
`total_like` no longer appear as `0` when the endpoint carrying them refused to
say. In JSON the key is gone, in a table or a csv the cell is empty, and a
consumer reading `.total_view` now gets null where it used to get `0`. A
creator whose totals really are zero still gets a zero, which is the entire
point of the distinction. Anything summing those columns was quietly wrong
before and is right now.

**A 412 is no longer retried.** An HTTP 412 and a `-352` are one refusal in two
forms, and the old code retried the first four times with backoff, which is the
worst possible response to risk control declining requests from an address.
Retries now apply to 429 and 5xx, which are a server saying it is busy, and to
nothing else.

### Added

**A provenance envelope on every record.** Every record carries an `envelope`
describing the reading rather than the thing read: which endpoint answered,
whether the request was signed, what state the response was sorted into, when,
and how large the body was. `envelope.missed` names the fields the record does
not carry and says what stopped each one. It is in the JSON, out of the table
and the csv, and reachable with `--fields envelope`.

**`bili download`.** Audio, by wrapping
[BBDown](https://github.com/nilaoda/BBDown), with ffmpeg doing the transcode for
`mp3`, `flac` and `wav`. Whatever you paste is resolved to a bvid first, so a
URL, a bare av number and a bvid all behave the same. Neither binary is bundled
and a missing one is reported before the transfer starts, with the name of the
binary and where to get it.

**`bili verify --live`.** Re-measures the recorded endpoint requirement matrix
against the real API and prints what each endpoint answered, signed and
unsigned. `--strict` exits non-zero when a row no longer matches what is
recorded.

**A weekly drift job.** The same measurement on a schedule, opening at most one
issue when the site moves rather than turning the build red.

**Typed errors.** `bili.Kind(err)` reports the state an error came from, so a
program embedding the library can branch on the same distinction the exit codes
expose.

### Changed

Go 1.26.6 and current dependencies. `x/space/upstat` refusing anonymous callers
is now recorded as a measured fact with a note rather than as an empty result.
The cache key for a signed request no longer includes `wts` and `w_rid`, which
are how a request was asked and not what was asked, so the cache is useful on
the gated half of the API for the first time.

### Fixed

`video_count` was never fetched at all. It was declared on the record and
written by nothing, which is why it printed `0` for a creator with 924 uploads.
It comes from the listing endpoint's own pagination total now.

`--dry-run` was being classified as a silent refusal, because the invented
response carries no payload by design and the payload rule is what catches a
refusal wearing a success code.

`bili audio au1` exited 1 with an untranslated `4511001`, and exits 7 with an
English message.

The cached WBI signing key was never being used by anything that pins the
clock. The six hour freshness window was measured against wall time while the
moment it was stored came from the injected clock, so the two never agreed and
every signed call refetched `nav`. This only affects a library consumer calling
`SetNow`, and it is the reason one test in this repository was reaching the
network.

## v0.2.0

`bili discover`, a breadth-first walk of the video and creator graph, with
`--follow`, `--depth` and `--fanout`. Colored output through lipgloss: rounded
table borders, a colored header on a true-color terminal, and syntax
highlighting for JSON and JSONL.

## v0.1.1

A demo GIF and a rewritten README.

## v0.1.0

First release. The client, the command tree, the output formatter, the offline
id resolver, WBI signing, the protobuf danmaku decoder, the documentation site,
and the release pipeline.
