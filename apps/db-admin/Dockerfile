# syntax=docker/dockerfile:1.7

# The image of the PACA database admin service.
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
# The build context is the root of the monorepository: the service imports
# the shared libraries of the repository, and they are modules of their own.

ARG GO_IMAGE=golang:1.27.3-alpine3.23@sha256:754d5200fc3edae76d3b6598a9a3fac7e74981ee6a5413ab955025f185753ef2
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

# --- Stage 1: build the binary -------------------------------------------
FROM ${GO_IMAGE} AS build

ARG VERSION=dev

WORKDIR /src

# The context of the build is the root of the monorepository, because the
# service imports the shared libraries of the repository, and a library is a
# module of its own next to it.
#
# The layout of the sources is the layout of the repository, so the local
# dependency of the service on the database library resolves the same way
# it does on a machine.
#
# The dependencies are downloaded first, so the layer of the dependencies is
# reused every time the sources change.
COPY apps/db-admin/go.mod apps/db-admin/go.sum ./apps/db-admin/
COPY packages/database/go.mod packages/database/go.sum ./packages/database/
COPY packages/domain/go.mod ./packages/domain/
RUN cd apps/db-admin && go mod download

COPY apps/db-admin/ ./apps/db-admin/
COPY packages/database/ ./packages/database/
COPY packages/domain/ ./packages/domain/

# A static binary, so the runtime image needs no C library at all.
RUN cd apps/db-admin && CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags "-s -w -X main.version=${VERSION}" \
        -o /out/paca-db-admin \
        ./cmd/paca-db-admin

# --- Stage 2: runtime ----------------------------------------------------
FROM ${RUNTIME_IMAGE}

ARG VERSION=dev

LABEL org.opencontainers.image.title="paca-db-admin" \
      org.opencontainers.image.description="The PACA database admin service: the writer of the schema of the monorepository, with a health endpoint" \
      org.opencontainers.image.base.name="gcr.io/distroless/static-debian12:nonroot" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=build /out/paca-db-admin /app/paca-db-admin

# The port the health endpoint listens on. It is never published: the
# service is reachable from the other containers of the stack only.
EXPOSE 8082

# The user of the base image, not root.
USER nonroot:nonroot

ENV PACA_DB_ADMIN_HOST=0.0.0.0 \
    PACA_DB_ADMIN_PORT=8082 \
    PACA_LOG_LEVEL=info

# The health check asks the service, from inside the container, whether it
# is alive. It needs the binary itself and nothing else.
#
# The start period covers the wait of the writer for its database: the
# service waits for the database before it serves, and the container is
# not declared unhealthy while it waits.
HEALTHCHECK --interval=10s --timeout=5s --start-period=90s --retries=3 \
    CMD ["/app/paca-db-admin", "healthcheck"]

ENTRYPOINT ["/app/paca-db-admin"]
