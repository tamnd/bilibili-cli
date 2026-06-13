# Consumed by GoReleaser: it copies the already cross-compiled binary out of the
# build context rather than compiling, so the image build is fast and uses the
# same static binary every other artifact ships.
#
# GoReleaser builds one multi-platform image with buildx and stages each
# platform's binary under a $TARGETPLATFORM directory (e.g. linux/amd64/) in the
# build context, so the COPY line selects the right one through the automatic
# TARGETPLATFORM build arg.
FROM alpine:3.21

ARG TARGETPLATFORM

# ca-certificates for HTTPS to api.bilibili.com; tzdata for sane timestamps.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 bili \
 && mkdir -p /data \
 && chown bili:bili /data

COPY $TARGETPLATFORM/bili /usr/bin/bili

USER bili
WORKDIR /data

# State (the response cache) lives under /data; mount a volume to keep it:
#
#   docker run ghcr.io/tamnd/bili video BV17x411w7KC
ENV BILI_CACHE_DIR=/data/cache
VOLUME ["/data"]

ENTRYPOINT ["/usr/bin/bili"]
