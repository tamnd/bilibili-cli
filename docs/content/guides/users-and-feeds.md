---
title: "Users and feeds"
description: "Browse a creator's profile and catalogue, their favorite folders, and their dynamics feed."
weight: 30
---

A creator (`mid`) is the other big anchor on bilibili. Four commands cover the
account and what hangs off it: `user` for the profile and catalogue, `favorites`
and `favorite` for collections, and `dynamics`/`dynamic` for the feed.

All accept a bare `mid`, a `space.bilibili.com/<mid>` link, or (for the feed) a
`t.bilibili.com` dynamic link.

## A creator

```bash
bili user 2
```

The default view is the profile: name, sign, level, and the follower and video
statistics. The same command pivots to other parts of the account with a flag:

```bash
bili user 2 --videos      # the creator's uploaded videos
bili user 2 --videos -o url
bili user 2 --dynamics    # their dynamics feed (shorthand for `dynamics`)
```

`--videos` paginates; combine it with `-n` to cap the count and `--order` to
sort by newest or by plays where the endpoint allows.

## Favorites

bilibili organizes saved videos into folders. List a creator's public folders,
then open one:

```bash
bili favorites 2            # the folders (each has an `ml` id)
bili favorite ml123         # the videos inside one folder
```

Most folders are private by default, so a `favorites` call on an account that
keeps them private returns an empty list rather than an error. That is the site's
privacy setting, not a problem with the command.

## Dynamics

The dynamics feed is a creator's stream of posts: videos, forwards, text, and
images.

```bash
bili dynamics 2            # the whole feed for a creator
bili dynamic <dynamic-id>  # one post in full
```

`dynamics` works anonymously. A single `dynamic` detail is one of the few
endpoints bilibili gates behind its anti-bot system for anonymous callers; when
you hit that, bili tells you to pass a logged-in cookie:

```bash
export BILI_COOKIE='SESSDATA=...; bili_jct=...; DedeUserID=...'
bili dynamic <dynamic-id>
```

See [configuration](/reference/configuration/) for how cookies are supplied and
why bili never prints them back.
