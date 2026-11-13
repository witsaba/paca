# The image of the memory of PACA: the store of Engram.
#
# The payload is the official release of Engram, and it is verified: the
# archive of the release is checked against the sha256 that is pinned for its
# architecture, and the build fails when the bytes are not the ones that were
# pinned. The binary is static (CGO_ENABLED=0), so it runs on any base.
#
# The version and the checksums live in the compose file of the stack, so the
# pin of the payload is one place and not two.

ARG BASE_IMAGE=alpine:3.22@sha256:2e0b81d60597f449baa56dc25d355367869d2dca4cd57e866f1e2b52af3eb55f
ARG ENGRAM_VERSION=3.7.1
ARG ENGRAM_SHA256_AMD64=bc1acef625f16923b2ba548e0c911996b30d65c7c7cf15e949ecbc58136b127b
ARG ENGRAM_SHA256_ARM64=d5023e786af7cefe07a5d46f8064851d5f89be73bdb10af4a148358ed0605583

FROM ${BASE_IMAGE}

ARG ENGRAM_VERSION
ARG ENGRAM_SHA256_AMD64
ARG ENGRAM_SHA256_ARM64
ARG TARGETARCH

# curl is the client of the smoke test and of the healthcheck of the
# container; tar unpacks the release; su-exec drops the privileges of the
# process before the store runs.
#
# The packages are pinned by version, like every other dependency of the
# repository. The base image is pinned by digest, and the packages are the
# ones of its release; when the release gets a security update, the pin is
# bumped with it, on purpose, and the build fails until it is.
RUN apk add --no-cache \
      ca-certificates=20261020-r0 \
      curl=8.14.1-r4 \
      su-exec=0.2-r3 \
      tar=1.35-r3

# The release of the architecture, checked against the pinned checksum.
RUN set -eu; \
    case "${TARGETARCH}" in \
      amd64) expected="${ENGRAM_SHA256_AMD64}" ;; \
      arm64) expected="${ENGRAM_SHA256_ARM64}" ;; \
      *) echo "engram: unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/Gentleman-Programming/engram/releases/download/v${ENGRAM_VERSION}/engram_${ENGRAM_VERSION}_linux_${TARGETARCH}.tar.gz"; \
    curl -fsSL --retry 3 -o /tmp/engram.tar.gz "${url}"; \
    echo "${expected}  /tmp/engram.tar.gz" | sha256sum -c -; \
    mkdir -p /opt/engram; \
    tar -xzf /tmp/engram.tar.gz -C /opt/engram engram; \
    chmod 0755 /opt/engram/engram; \
    rm -f /tmp/engram.tar.gz

# The entrypoint of the image owns the user of the process and the ownership
# of the state, because the socket of the API is created by the store itself
# and it belongs to that user: the consumer of the socket is the container of
# the agent, and it runs as the same user.
COPY engram/entrypoint.sh /opt/engram/entrypoint.sh
RUN chmod 0755 /opt/engram/entrypoint.sh

# The state of the store lives in the volume of the stack, and every path of
# the process points at it.
ENV ENGRAM_BIN=/opt/engram/engram
ENV HOME=/data
WORKDIR /data

ENTRYPOINT ["/opt/engram/entrypoint.sh"]
CMD ["serve"]
