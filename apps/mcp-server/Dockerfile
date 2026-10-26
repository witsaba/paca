# syntax=docker/dockerfile:1.7

# The image of the PACA MCP server.
#
# Versions (checked 2026-10-27):
#   Go 1.27.3     -> the latest security release of the 1.27 line. The 1.26
#                    line got the same fix in 1.26.10 (2026-10-19): the
#                    crypto/x509 mitigation for a certificate authority.
#                    The toolchain of the repository is 1.27.3, and the
#                    image builds with the very same version.
#   distroless    -> static-debian12: the smallest runtime, without a shell
#                    and without a package manager, so nothing that runs
#                    inside the container can be changed.
#
# The base images are pinned by digest, so a future tag move cannot silently
# change what this image is built on top of.
#
# The build context is the root of the monorepository: the server imports
# the shared libraries of the repository, and they are modules of their own.

ARG GO_IMAGE=golang:1.27.3-alpine3.23@sha256:754d5200fc3edae76d3b6598a9a3fac7e74981ee6a5413ab955025f185753ef2
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

# --- Stage 1: build the binary -------------------------------------------
FROM ${GO_IMAGE} AS build

ARG VERSION=dev

WORKDIR /src

# The context of the build is the root of the monorepository, because the
# server imports the shared libraries of the repository, and a library is a
# module of its own next to it.
#
# The layout of the sources is the layout of the repository, so the local
# dependency of the server on the core resolves the same way here as it does
# on a machine.
#
# The dependencies are downloaded first, so the layer of the dependencies is
# reused every time the sources change.
COPY apps/mcp-server/go.mod apps/mcp-server/go.sum ./apps/mcp-server/
COPY packages/core/go.mod packages/core/go.sum ./packages/core/
RUN cd apps/mcp-server && go mod download

COPY apps/mcp-server/ ./apps/mcp-server/
COPY packages/core/ ./packages/core/

# A static binary, so the runtime image needs no C library at all.
RUN cd apps/mcp-server && CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w -X main.version=${VERSION}" \
        -o /out/paca-mcp-server \
        ./cmd/paca-mcp-server

# --- Stage 2: runtime ----------------------------------------------------
FROM ${RUNTIME_IMAGE}

ARG VERSION=dev

LABEL org.opencontainers.image.title="paca-mcp-server" \
      org.opencontainers.image.description="The PACA MCP server: the Model Context Protocol over the official Go SDK, with a health endpoint" \
      org.opencontainers.image.base.name="gcr.io/distroless/static-debian12:nonroot" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=build /out/paca-mcp-server /app/paca-mcp-server

# The port the MCP transport listens on. It is never published: the server
# is reachable from the other containers of the stack only.
EXPOSE 8081

# The user of the base image, not root.
USER nonroot:nonroot

ENV PACA_MCP_HOST=0.0.0.0 \
    PACA_MCP_PORT=8081 \
    PACA_LOG_LEVEL=info

# The health check asks the server, from inside the container, whether it is
# alive. It needs the binary itself and nothing else.
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/paca-mcp-server", "healthcheck"]

ENTRYPOINT ["/app/paca-mcp-server"]
