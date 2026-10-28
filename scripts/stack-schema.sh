#!/usr/bin/env bash
# The assertion of the schema of the monorepository.
#
# It runs against the running project and asserts the guarantees the stack
# promises:
#
#   * every container of the project is running and healthy;
#   * the writer of the schema was not restarted: it waits for its
#     database instead of exiting and being restarted;
#   * the schema of the database is complete: every migration of
#     packages/core/migrations is applied;
#   * the table `users` exists.
#
# The usage:
#
#   ./scripts/stack-schema.sh
#
# The containers are found by their compose service label, or by their
# fixed name, so the assertions hold whichever way the project was
# started. The credentials of the database are read from inside the
# container: they are never read from the host.
#
# Exit code 0 means every assertion passed.
set -euo pipefail

# The project of the monorepository. The compose files of the stacks all
# declare it, so the labels of the containers carry it.
PROJECT=paca

# The migrations of the schema, and the writer of it.
MIGRATIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../packages/core/migrations" && pwd)"
WRITER=db-admin

failures=0

pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; failures=$((failures + 1)); }

assert_equals() {
	local what="$1" expected="$2" actual="$3"
	if [[ "${actual}" == "${expected}" ]]; then
		pass "${what} = ${actual}"
	else
		fail "${what} = ${actual} (expected ${expected})"
	fi
}

# The container of a service of the project: found by its compose labels,
# or by its fixed name when the labels are not there.
container_of() {
	local service="$1" name="$2" found
	found="$(docker ps --filter "label=com.docker.compose.project=${PROJECT}" \
		--filter "label=com.docker.compose.service=${service}" \
		--format '{{.Names}}' | head -1)"
	if [[ -z "${found}" ]]; then
		found="${name}"
	fi
	printf '%s' "${found}"
}

# Runs one SQL statement inside the container of the database and prints
# the value it returned. The credentials of the database are read from
# inside the container, never from the host.
query() {
	docker exec "${db_container}" sh -c \
		"psql -X -q -A -t -v ON_ERROR_STOP=1 -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"$1\"" || true
}

echo "== containers =="
for entry in "postgres:paca-postgres" "db-admin:paca-db-admin" \
	"mcp-server:paca-mcp-server" "jaeger:paca-jaeger"; do
	service="${entry%%:*}"
	name="${entry##*:}"
	container="$(container_of "${service}" "${name}")"
	if [[ -z "${container}" ]]; then
		fail "the ${service} container is not running"
		continue
	fi
	pass "the container ${container} is running"
	state="$(docker inspect -f '{{.State.Status}}' "${container}" 2>/dev/null || echo unknown)"
	assert_equals "the state of ${container}" "running" "${state}"
	health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' \
		"${container}" 2>/dev/null || echo unknown)"
	assert_equals "the health of ${container}" "healthy" "${health}"
done

echo "== the writer of the schema =="
writer_container="$(container_of "${WRITER}" "paca-db-admin")"
if [[ -z "${writer_container}" ]]; then
	fail "the ${WRITER} container is not running"
else
	restarts="$(docker inspect -f '{{.RestartCount}}' "${writer_container}" 2>/dev/null || echo unknown)"
	assert_equals "the restart count of ${writer_container}" "0" "${restarts}"
fi

echo "== the schema =="
db_container="$(container_of postgres paca-postgres)"
if [[ -z "${db_container}" ]]; then
	fail "the database container is not running"
else
	# The number of the migration files of the repository: the schema is
	# complete when the database carries every one of them.
	migration_count="$(find "${MIGRATIONS_DIR}" -maxdepth 1 -type f -name '*.sql' | wc -l | tr -d '[:space:]')"
	# The table of the versions carries one row per migration, plus the
	# baseline row of version 0 that the migrator writes when it creates
	# the table.
	applied="$(query 'select count(*) from goose_db_version where is_applied')"
	not_applied="$(query 'select count(*) from goose_db_version where not is_applied')"
	max_version="$(query 'select coalesce(max(version_id), 0) from goose_db_version')"
	assert_equals "the applied migrations" "$((migration_count + 1))" "${applied}"
	assert_equals "the migrations that are not applied" "0" "${not_applied}"
	assert_equals "the maximum applied version" "${migration_count}" "${max_version}"
	users="$(query "select count(*) from information_schema.tables where table_schema = 'public' and table_name = 'users'")"
	assert_equals "the table users" "1" "${users}"
fi

echo
if [[ "${failures}" -eq 0 ]]; then
	echo "All assertions passed."
	exit 0
fi
echo "${failures} assertion(s) failed."
exit 1
