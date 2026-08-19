---
title: "Installation"
description: "Install bili from a release, with go install, or from source. No dependencies."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/bilibili-cli/releases) carries archives
for Linux, macOS, and Windows on amd64 and arm64, plus deb, rpm, and apk
packages for Linux. Download, unpack, put `bili` on your `PATH`, done. Each
archive ships with a CycloneDX SBOM, and the `checksums.txt` is signed with
keyless [cosign](https://docs.sigstore.dev/) if you want to verify before
running:

```bash
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp 'https://github.com/tamnd/bilibili-cli' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

## With Go

```bash
go install github.com/tamnd/bilibili-cli/cmd/bili@latest
```

That puts `bili` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless you moved
it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/bilibili-cli
cd bilibili-cli
make build        # produces ./bin/bili
./bin/bili version
```

## Container

The release also publishes a multi-arch image on GHCR:

```bash
docker run --rm ghcr.io/tamnd/bili:latest video BV17x411w7KC
```

## Requirements

- **Go 1.26 or later** to build. The released binary has no Go requirement.

That is the whole list for reading. No config file, no database to provision, no
daemon, and no API key.

One command wants something bili does not ship. `bili download` runs
[BBDown](https://github.com/nilaoda/BBDown) to fetch the audio, and
[ffmpeg](https://ffmpeg.org/download.html) to transcode it when you ask for
`mp3`, `flac` or `wav`. Both are looked up on PATH and neither is installed for
you. Everything else here works without them, and a missing one is reported by
name before anything starts.

## Checking the install

```bash
bili version
```

prints the version and exits. Then confirm it can reach bilibili:

```bash
bili nav
```

prints the current login state (anonymous by default) and the live WBI keys. If
that comes back without an error, you are ready for the
[quick start](/getting-started/quick-start/).
