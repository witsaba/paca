#!/usr/bin/env bash
# Smoke test for the PACA PostgreSQL container.
#
# Asserts the guarantees this database promises: UTF-8, UTC, UUID, vector
# search, checksums, and that nothing is published on the host.
#
#   ./db/postgres/scripts/smoke-test.sh
#
# Exit code 0 means every assertion passed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE=(docker compose -f "${DB_DIR}/compose.yaml")

failures=0

pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; failures=$((failures + 1)); }

# Runs one SQL statement and prints the single value it returned.
query() {
	"${COMPOSE[@]}" exec -T postgres sh -c "psql -X -q -A -t -v ON_ERROR_STOP=1 -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"$1\""
}

assert_equals() {
	local what="$1" expected="$2" actual="$3"
	if [[ "${actual}" == "${expected}" ]]; then
		pass "${what} = ${actual}"
	else
		fail "${what} = ${actual} (expected ${expected})"
	fi
}

assert_contains() {
	local what="$1" needle="$2" haystack="$3"
	if [[ "${haystack}" == *"${needle}"* ]]; then
		pass "${what} contains ${needle}"
	else
		fail "${what} does not contain ${needle} (got: ${haystack})"
	fi
}

echo "== container state =="
state="$("${COMPOSE[@]}" ps --format '{{.Name}} {{.State}} {{.Health}}' postgres 2>/dev/null || true)"
if [[ -z "${state}" ]]; then
	echo "  FAIL the postgres container is not running (run 'make db-up' first)"
	exit 1
fi
assert_contains "container" "running" "${state}"
assert_contains "container health" "healthy" "${state}"

echo "== version =="
assert_contains "server_version" "18.6" "$(query 'show server_version')"
assert_contains "pgvector version" "0.8.7" "$(query "select extversion from pg_extension where extname = 'vector'")"

echo "== encoding =="
assert_equals "server_encoding" "UTF8" "$(query 'show server_encoding')"
assert_equals "client_encoding" "UTF8" "$(query 'show client_encoding')"
assert_contains "database encoding" "UTF8" "$(query "select pg_encoding_to_char(encoding) from pg_database where datname = current_database()")"

echo "== timezone =="
assert_equals "timezone" "UTC" "$(query 'show timezone')"

echo "== extensions =="
extensions="$(query "select string_agg(extname, ',' order by extname) from pg_extension")"
for ext in vector pgcrypto uuid-ossp plpgsql; do
	assert_contains "extensions" "${ext}" "${extensions}"
done

echo "== uuid =="
uuid="$(query 'select gen_random_uuid()')"
if [[ "${uuid}" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]; then
	pass "gen_random_uuid() returns a v4 UUID"
else
	fail "gen_random_uuid() returned '${uuid}'"
fi
assert_equals "uuid_generate_v4()" "1" "$(query 'select count(*) from (select uuid_generate_v4()) s')"

echo "== vector =="
assert_equals "vector literal" "[1,2,3]" "$(query "select '[1,2,3]'::vector")"
assert_equals "vector distance to itself" "0" "$(query "select round((('[1,2,3]'::vector <-> '[1,2,3]'::vector))::numeric, 0)")"
# L2 distance between [1,2,3] and [2,4,6] is sqrt(14) = 3.7416...
assert_equals "vector distance [1,2,3] <-> [2,4,6]" "4" "$(query "select round((('[1,2,3]'::vector <-> '[2,4,6]'::vector))::numeric, 0)")"

# A real vector column with an ANN index must answer a nearest-neighbour query.
nearest="$("${COMPOSE[@]}" exec -T postgres sh -c "psql -X -q -A -t -v ON_ERROR_STOP=1 -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\"" <<'SQL'
create temp table smoke_vectors (id int primary key, embedding vector(3));
insert into smoke_vectors values (1, '[1,2,3]'), (2, '[4,5,6]');
create index on smoke_vectors using ivfflat (embedding vector_cosine_ops) with (lists=1);
select id from smoke_vectors order by embedding <-> '[1,1,1]'::vector limit 1;
SQL
)"
assert_equals "ivfflat nearest neighbour" "1" "${nearest}"

echo "== storage =="
assert_equals "data checksums" "on" "$(query 'show data_checksums')"

echo "== isolation =="
# Docker publishes a port only when NetworkSettings.Ports maps it to a host
# address; for this service it must be null.
ports_json="$(docker inspect -f '{{json .NetworkSettings.Ports}}' paca-postgres 2>/dev/null || echo '{}')"
# The container may expose 5432 internally; what matters is that no host
# address/port binding exists. This template lists the host bindings only.
host_bindings="$(docker inspect -f '{{range $p, $conf := .NetworkSettings.Ports}}{{range $conf}}{{.HostIp}}:{{.HostPort}} {{end}}{{end}}' paca-postgres 2>/dev/null || true)"
if [[ -z "${host_bindings// /}" ]]; then
	pass "no host port is published (${ports_json})"
else
	fail "a host port is published: ${host_bindings}"
fi

# A container on the internal network must be able to connect.
password="$("${COMPOSE[@]}" exec -T postgres sh -c 'printf %s "$POSTGRES_PASSWORD"')"
if PGPASSWORD="${password}" docker run --rm --network paca-postgres_internal \
	-e PGPASSWORD="${password}" postgres:18.6-bookworm \
	psql -X -q -A -t -h postgres -U paca -d paca -c 'select 1' 2>/dev/null | grep -q '^1$'; then
	pass "another container on the internal network can connect"
else
	fail "a container on the internal network could not connect"
fi

# The host must not be able to reach the published port.
if nc -z -G 2 127.0.0.1 5432 >/dev/null 2>&1; then
	fail "the host can reach 127.0.0.1:5432 (port leaked)"
else
	pass "the host cannot reach 127.0.0.1:5432"
fi

echo
if [[ "${failures}" -eq 0 ]]; then
	echo "All assertions passed."
	exit 0
fi
echo "${failures} assertion(s) failed."
exit 1
