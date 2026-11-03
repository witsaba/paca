# The image of the agent of PACA: Hermes Agent with the binary of the memory.
#
# The agent reads the memory of the monorepository over the stdio MCP
# transport, and the MCP server is a process that the agent spawns: the
# binary of the store has to be inside the image of the agent, because a
# container cannot reach the filesystem of its neighbour.
#
# The image is therefore the image of the agent, unchanged, plus the
# payload of the store: the official release of Engram, verified against
# the sha256 that is pinned for its architecture. The build fails when the
# bytes are not the ones that were pinned.
#
# The base image is pinned by digest and the payload is pinned by version
# and by checksum, exactly like the image of the memory itself. The two
# pins are the same pins: the gate of the repository asserts that they
# never drift apart.
#
# The entrypoint of the base image is left alone on purpose: it starts the
# supervision tree of the agent and it reaps the orphans that the tools
# leave behind.

ARG HERMES_IMAGE=nousresearch/hermes-agent:stable@sha256:9c5561579f3ffdf333401797094ac3a743738d6c25d40f4331c7bf77b694238d
ARG ENGRAM_VERSION=3.6.0
ARG ENGRAM_SHA256_AMD64=945fb471188826c251160947f8dda9fe148ba0dd095fa3453174fe7ab5c6479e
ARG ENGRAM_SHA256_ARM64=00a2d7645aa428e56c32c18b8de8bfa90d4c6e0f68c4d47bc82e3d6e3e77879d

FROM ${HERMES_IMAGE}

ARG ENGRAM_VERSION
ARG ENGRAM_SHA256_AMD64
ARG ENGRAM_SHA256_ARM64
ARG TARGETARCH

# The release of the architecture, checked against the pinned checksum. The
# binary is static (CGO_ENABLED=0), so it runs on the base of the agent
# without a library of its own.
RUN set -eu; \
    case "${TARGETARCH}" in \
      amd64) expected="${ENGRAM_SHA256_AMD64}" ;; \
      arm64) expected="${ENGRAM_SHA256_ARM64}" ;; \
      *) echo "hermes: unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/Gentleman-Programming/engram/releases/download/v${ENGRAM_VERSION}/engram_${ENGRAM_VERSION}_linux_${TARGETARCH}.tar.gz"; \
    curl -fsSL --retry 3 -o /tmp/engram.tar.gz "${url}"; \
    echo "${expected}  /tmp/engram.tar.gz" | sha256sum -c -; \
    mkdir -p /opt/engram; \
    tar -xzf /tmp/engram.tar.gz -C /opt/engram engram; \
    chmod 0755 /opt/engram/engram; \
    rm -f /tmp/engram.tar.gz

# The seeding of the configuration of the agent: the one-shot service of
# the stack runs it before the gateway comes up.
COPY hermes/init.sh /opt/hermes/init.sh
RUN chmod 0755 /opt/hermes/init.sh
