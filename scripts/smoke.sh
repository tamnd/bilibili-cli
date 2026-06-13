#!/usr/bin/env bash
# Exercise every bili command against the live bilibili.com API and report a
# pass/fail line per command. Read-only: it only fetches public data, never
# logs in, downloads media, or writes anything outside a temp directory.
#
# Usage:
#   ./scripts/smoke.sh              # uses the bili on $PATH
#   BILI=./bin/bili ./scripts/smoke.sh
#
# Some endpoints (single dynamic detail, column articles) are gated by
# bilibili's anti-bot for anonymous traffic and may return code -352 from a
# fresh IP. Those are reported as SKIP, not FAIL, since they need a logged-in
# cookie rather than a code fix.

set -u

BILI="${BILI:-bili}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0 fail=0 skip=0

# run NAME -- args...   : succeeds when the command exits 0 and prints something.
# Retries once after a pause so a single transient rate-limit does not fail the
# suite; bilibili throttles bursts of anonymous requests.
run() {
  local name="$1"; shift
  [ "$1" = "--" ] && shift
  local out rc
  out="$("$BILI" "$@" 2>"$TMP/err")"; rc=$?
  if [ $rc -ne 0 ] || [ -z "$out" ]; then
    sleep 3
    out="$("$BILI" "$@" 2>"$TMP/err")"; rc=$?
  fi
  if [ $rc -eq 0 ] && [ -n "$out" ]; then
    printf 'PASS  %-26s %s\n' "$name" "$*"
    pass=$((pass + 1))
    printf '%s\n' "$out" | head -1 | cut -c1-100
  elif grep -q '\-352\|\-509\|risk control\|rate limit' "$TMP/err"; then
    printf 'SKIP  %-26s %s  (anti-bot / rate limit)\n' "$name" "$*"
    skip=$((skip + 1))
  else
    printf 'FAIL  %-26s %s\n' "$name" "$*"
    sed 's/^/      /' "$TMP/err" | head -3
    fail=$((fail + 1))
  fi
  sleep 0.6
}

# run_optional NAME -- args... : passes on a clean exit even with no output.
# Used for endpoints whose result is legitimately empty for many accounts
# (favorites are private by default site-wide).
run_optional() {
  local name="$1"; shift
  [ "$1" = "--" ] && shift
  if "$BILI" "$@" >/dev/null 2>"$TMP/err"; then
    printf 'PASS  %-26s %s  (may be empty)\n' "$name" "$*"
    pass=$((pass + 1))
  elif grep -q '\-352\|\-509\|risk control\|rate limit' "$TMP/err"; then
    printf 'SKIP  %-26s %s  (anti-bot / rate limit)\n' "$name" "$*"
    skip=$((skip + 1))
  else
    printf 'FAIL  %-26s %s\n' "$name" "$*"
    sed 's/^/      /' "$TMP/err" | head -3
    fail=$((fail + 1))
  fi
  sleep 0.6
}

echo "bili smoke test against the live API"
echo "binary: $BILI"
"$BILI" version
echo

# Discover a live video, user, and search term so the suite never depends on a
# specific id that might be removed later.
BV="$("$BILI" popular -n 1 -o jsonl --fields bvid 2>/dev/null | sed -n 's/.*"bvid":"\([^"]*\)".*/\1/p')"
BV="${BV:-BV17x411w7KC}"
echo "discovered video: $BV"
echo

# --- id / resolution ---
run id              -- id "$BV"
run id-url          -- id "https://www.bilibili.com/video/$BV"
run id-space        -- id "https://space.bilibili.com/2"

# --- video ---
run video           -- video "$BV" -o jsonl -n 1
run related         -- related "$BV" -n 3 -o jsonl
run streams         -- streams "$BV" -o jsonl -n 1
run comments        -- comments "$BV" -n 3 -o jsonl
run danmaku         -- danmaku "$BV" -n 5 -o jsonl

# --- discovery feeds ---
run popular         -- popular -n 3 -o jsonl
run rank            -- rank -n 3 -o jsonl
run trending        -- trending -n 5 -o jsonl
run suggest         -- suggest 原神
run search-video    -- search 原神 --type video -n 3 -o jsonl
run search-user     -- search 影视飓风 --type user -n 2 -o jsonl
run search-bangumi  -- search 凡人修仙传 --type bangumi -n 2 -o jsonl
run search-live     -- search 英雄联盟 --type live_room -n 2 -o jsonl

# --- creators ---
run user            -- user 2 -o jsonl
run user-stat       -- user 2 --stat -o jsonl
run user-videos     -- user 2 --videos -n 3 -o jsonl
run user-dynamics   -- user 2 --dynamics -n 2 -o jsonl
run dynamics        -- dynamics 2 -n 2 -o jsonl
run_optional favorites -- favorites 2 -o jsonl

# --- live / bangumi ---
run live-browse     -- live --browse --area 1 -n 3 -o jsonl
run live-room       -- live 1 -o jsonl
run bangumi         -- bangumi ss33802 -n 3 -o jsonl

# --- anti-bot gated (SKIP on -352 from a fresh IP) ---
DYN="$("$BILI" dynamics 2 -n 1 -o jsonl --fields id 2>/dev/null | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')"
[ -n "$DYN" ] && run dynamic -- dynamic "$DYN" -o jsonl
CV="$("$BILI" search bilibili --type article -n 1 -o jsonl --fields cvid 2>/dev/null | sed -n 's/.*"cvid":\([0-9]*\).*/\1/p')"
[ -n "$CV" ] && run article -- article "cv$CV" -o jsonl

# --- meta / housekeeping ---
run nav             -- nav -o jsonl
run config-show     -- config show
run cache-stat      -- cache stat
run version-json    -- version -o jsonl

# --- crawl ---
echo
echo "crawl $BV --comments --danmaku --out $TMP/crawl"
if "$BILI" crawl "$BV" --comments --danmaku --out "$TMP/crawl" >/dev/null 2>"$TMP/err"; then
  lines=$(wc -l < "$TMP/crawl/videos.jsonl" 2>/dev/null || echo 0)
  if [ "$lines" -gt 0 ]; then
    printf 'PASS  %-26s wrote %s video records\n' "crawl" "$lines"
    pass=$((pass + 1))
  else
    printf 'FAIL  %-26s no records written\n' "crawl"
    fail=$((fail + 1))
  fi
else
  printf 'FAIL  %-26s\n' "crawl"; sed 's/^/      /' "$TMP/err" | head -3
  fail=$((fail + 1))
fi

echo
echo "----------------------------------------"
echo "pass=$pass  skip=$skip  fail=$fail"
[ $fail -eq 0 ]
