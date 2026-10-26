# syntax=docker/dockerfile:1.7

# PostgreSQL 18.6 + pgvector 0.8.7
#
# Versions (checked 2026-10-26):
#   PostgreSQL 18.6  -> latest stable minor release (18.5 was skipped due to a
#                       post-wrap regression; PostgreSQL 19 is still beta).
#   pgvector    0.8.7 -> latest release (2026-10-01).
#
# The base image is pinned by digest so a future tag move cannot silently
# change what we build on top of.
ARG POSTGRES_IMAGE=postgres:18.6-bookworm@sha256:bbf83594798e144bff365a9c191c5d779f181fb3135902064ab8ff11918020fc

# --- Stage 1: compile pgvector ------------------------------------------
FROM ${POSTGRES_IMAGE} AS pgvector-builder

ARG PGVECTOR_VERSION=0.8.7
ARG PG_MAJOR=18
# sha256 of the official v0.8.7 source tarball (github.com/pgvector/pgvector).
ARG PGVECTOR_SHA256=cac0b10c360f05b2d521200105ba3697e773d4cd3731f5a915a7e37ebe0bea85

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        curl \
        "postgresql-server-dev-${PG_MAJOR}"; \
    rm -rf /var/lib/apt/lists/*; \
    curl -fsSL -o /tmp/pgvector.tar.gz \
        "https://github.com/pgvector/pgvector/archive/refs/tags/v${PGVECTOR_VERSION}.tar.gz"; \
    echo "${PGVECTOR_SHA256}  /tmp/pgvector.tar.gz" | sha256sum -c -; \
    mkdir -p /tmp/pgvector; \
    tar -xzf /tmp/pgvector.tar.gz -C /tmp/pgvector --strip-components=1; \
    cd /tmp/pgvector; \
    make; \
    make install DESTDIR=/tmp/install; \
    rm -rf /tmp/pgvector /tmp/pgvector.tar.gz

# --- Stage 2: runtime ----------------------------------------------------
FROM ${POSTGRES_IMAGE}

ARG PGVECTOR_VERSION=0.8.7

LABEL org.opencontainers.image.title="paca-postgres" \
      org.opencontainers.image.description="PostgreSQL 18.6 with pgvector ${PGVECTOR_VERSION}, UUID and UTC support, internal network only" \
      org.opencontainers.image.base.name="docker.io/library/postgres:18.6-bookworm" \
      org.opencontainers.image.licenses="PostgreSQL"

COPY --from=pgvector-builder /tmp/install /

# The init scripts are baked into the image (not bind-mounted) so the cluster
# is initialised identically on every machine.
COPY initdb/ /docker-entrypoint-initdb.d/

# Fail the build early if the base image no longer ships the extensions the
# init scripts rely on.
RUN set -eux; \
    test -f /usr/share/postgresql/18/extension/vector.control; \
    test -f /usr/share/postgresql/18/extension/uuid-ossp.control; \
    test -f /usr/share/postgresql/18/extension/pgcrypto.control; \
    pg_config --version

# UTF-8 cluster, UTC everywhere, data checksums for corruption detection.
ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8 \
    TZ=UTC \
    PGTZ=UTC \
    POSTGRES_INITDB_ARGS="--encoding=UTF8 --locale=C.UTF-8 --data-checksums"

EXPOSE 5432

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=5 \
    CMD pg_isready -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-postgres}" -h 127.0.0.1 || exit 1
