#!/usr/bin/env bash
# =============================================================================
# PACA — verify_schema.sh
#
# Applies the initial migration to a throwaway database and asserts that the
# schema behaves as documented in packages/db/README.md.
#
# This is a database-level check only. It does not need an application, a
# language runtime, or any dependency beyond the sqlite3 CLI.
#
# Usage:
#   bash packages/db/tools/verify_schema.sh
#
# Options:
#   --keep     leave the scratch database in place for inspection
#
# Requires: sqlite3 >= 3.37.0 (STRICT tables).
# =============================================================================
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATION="${PKG_DIR}/migrations/001_initial_schema.sql"

KEEP=0
[ "${1:-}" = "--keep" ] && KEEP=1

PASS=0
FAIL=0

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
head1() { printf '\n\033[1m%s\033[0m\n' "$*"; }

# --- preconditions -----------------------------------------------------------
if ! command -v sqlite3 >/dev/null 2>&1; then
  red "sqlite3 not found on PATH"; exit 2
fi

SQLITE_VERSION="$(sqlite3 -version | awk '{print $1}')"
SQLITE_NUM="$(printf '%s' "$SQLITE_VERSION" | cut -d. -f1,2)"
if [ "$(printf '%s\n3.37\n' "$SQLITE_NUM" | sort -V | head -1)" != "3.37" ]; then
  red "sqlite3 $SQLITE_VERSION is too old; 3.37.0 or newer is required (STRICT tables)"
  exit 2
fi

WORKDIR="$(mktemp -d)"
DB="${WORKDIR}/verify.db"
cleanup() { [ "$KEEP" -eq 1 ] || rm -rf "$WORKDIR"; }
trap cleanup EXIT

echo "sqlite3 version : ${SQLITE_VERSION}"
echo "migration       : ${MIGRATION}"
echo "scratch database: ${DB}"

# --- helpers -----------------------------------------------------------------
# run <expected: OK|ERR> <description> <sql...>
run() {
  local expected="$1" desc="$2" sql="$3"
  local out rc
  out="$(sqlite3 -bail "$DB" "PRAGMA foreign_keys = ON; $sql" 2>&1)"
  rc=$?
  if [ "$expected" = "OK" ] && [ "$rc" -eq 0 ]; then
    green "  PASS  ${desc}"; PASS=$((PASS + 1))
  elif [ "$expected" = "ERR" ] && [ "$rc" -ne 0 ]; then
    green "  PASS  ${desc}"; PASS=$((PASS + 1))
  else
    red   "  FAIL  ${desc}  (rc=${rc}) ${out}"; FAIL=$((FAIL + 1))
  fi
}

# --- 1. the migration applies ------------------------------------------------
head1 "1. Migration applies to an empty database"
if sqlite3 -bail "$DB" < "$MIGRATION" >/dev/null 2>"${WORKDIR}/err.txt"; then
  green "  PASS  migration applied cleanly"; PASS=$((PASS + 1))
else
  red "  FAIL  migration failed: $(cat "${WORKDIR}/err.txt")"; FAIL=$((FAIL + 1))
  exit 1
fi

# --- 2. structure ------------------------------------------------------------
head1 "2. Structure"
VERSION="$(sqlite3 "$DB" "PRAGMA user_version;")"
[ "$VERSION" = "1" ] && { green "  PASS  user_version = 1"; PASS=$((PASS+1)); } \
                     || { red "  FAIL  user_version is '${VERSION}', expected 1"; FAIL=$((FAIL+1)); }

INTEGRITY="$(sqlite3 "$DB" "PRAGMA integrity_check;")"
[ "$INTEGRITY" = "ok" ] && { green "  PASS  integrity_check = ok"; PASS=$((PASS+1)); } \
                        || { red "  FAIL  integrity_check = ${INTEGRITY}"; FAIL=$((FAIL+1)); }

FKVIOL="$(sqlite3 "$DB" "PRAGMA foreign_key_check;")"
[ -z "$FKVIOL" ] && { green "  PASS  foreign_key_check clean"; PASS=$((PASS+1)); } \
                  || { red "  FAIL  foreign_key_check: ${FKVIOL}"; FAIL=$((FAIL+1)); }

for t in accounts contacts loans movements loan_payments; do
  SQL="$(sqlite3 "$DB" "SELECT sql FROM sqlite_master WHERE name='${t}';")"
  if printf '%s' "$SQL" | grep -q 'STRICT'; then
    green "  PASS  ${t} is STRICT"; PASS=$((PASS+1))
  else
    red "  FAIL  ${t} is not STRICT"; FAIL=$((FAIL+1))
  fi
done

# --- 3. the discriminator and shape constraints ------------------------------
head1 "3. accounts: discriminator and shape"
run OK  "nequi with phone"                 "INSERT INTO accounts(kind,name,opened_on,phone) VALUES('nequi','Nequi','2024-01-01','3001234567');"
run ERR "nequi without phone"               "INSERT INTO accounts(kind,name,opened_on) VALUES('nequi','N2','2024-01-01');"
run OK  "bank with every bank field"        "INSERT INTO accounts(kind,name,opened_on,bank_code,account_number,account_type) VALUES('bank','Banco','2024-01-01','001','0001','ahorro');"
run ERR "bank missing account_number"       "INSERT INTO accounts(kind,name,opened_on,bank_code,account_type) VALUES('bank','B2','2024-01-01','001','ahorro');"
run ERR "nequi carrying a bank_code"        "INSERT INTO accounts(kind,name,opened_on,phone,bank_code) VALUES('nequi','N3','2024-01-01','300999','001');"
run ERR "bank carrying a phone"             "INSERT INTO accounts(kind,name,opened_on,phone,bank_code,account_number,account_type) VALUES('bank','B3','2024-01-01','300111','001','0003','corriente');"
run OK  "cash with no subtype field"        "INSERT INTO accounts(kind,name,opened_on) VALUES('cash','Efectivo','2024-01-01');"
run ERR "cash carrying a phone"             "INSERT INTO accounts(kind,name,opened_on,phone) VALUES('cash','C2','2024-01-01','300222');"
run ERR "unknown kind"                      "INSERT INTO accounts(kind,name,opened_on) VALUES('crypto','C','2024-01-01');"
run ERR "inactive without closed_on"        "INSERT INTO accounts(kind,name,opened_on,is_active) VALUES('cash','C3','2024-01-01',0);"
run OK  "inactive with closed_on"           "INSERT INTO accounts(kind,name,opened_on,is_active,closed_on) VALUES('cash','C4','2024-01-01',0,'2024-06-01');"
run ERR "duplicate nequi phone"             "INSERT INTO accounts(kind,name,opened_on,phone) VALUES('nequi','Dup','2024-01-01','3001234567');"
run ERR "malformed date"                    "INSERT INTO accounts(kind,name,opened_on) VALUES('cash','C5','01/01/2024');"
run ERR "impossible calendar date"          "INSERT INTO accounts(kind,name,opened_on) VALUES('cash','C6','2024-02-30');"

head1 "4. contacts: discriminator and shape"
run OK  "person with id_document"           "INSERT INTO contacts(kind,name,phone,id_document) VALUES('person','Pedro','3101','CC1');"
run ERR "person without id_document"        "INSERT INTO contacts(kind,name,phone) VALUES('person','P2','3102');"
run OK  "entity with tax_id and legal_name" "INSERT INTO contacts(kind,name,phone,tax_id,legal_name) VALUES('entity','Banco','3301','NIT1','Banco S.A.');"
run ERR "entity missing legal_name"         "INSERT INTO contacts(kind,name,phone,tax_id) VALUES('entity','E2','3302','NIT2');"
run ERR "duplicate entity NIT"              "INSERT INTO contacts(kind,name,phone,tax_id,legal_name) VALUES('entity','Banco3','3303','NIT1','Banco S.A.');"
run OK  "duplicate person cedula allowed"   "INSERT INTO contacts(kind,name,phone,id_document) VALUES('person','Pedro2','3103','CC1');"

head1 "5. movements"
# Fixture: loan 1 must exist before a movement may reference it.
sqlite3 -bail "$DB" "PRAGMA foreign_keys = ON;
INSERT INTO loans(contact_id,direction,kind,principal,granted_on)
  SELECT id,'borrowed','informal',50000,'2024-02-01' FROM contacts WHERE kind='person' ORDER BY id LIMIT 1;"
run ERR "amount of zero"                    "INSERT INTO movements(account_id,kind,amount,occurred_on) VALUES(1,'expense',0,'2024-01-01');"
run ERR "negative amount"                   "INSERT INTO movements(account_id,kind,amount,occurred_on) VALUES(1,'expense',-100,'2024-01-01');"
run ERR "unknown movement kind"             "INSERT INTO movements(account_id,kind,amount,occurred_on) VALUES(1,'gift',100,'2024-01-01');"
run ERR "non-numeric amount (STRICT)"       "INSERT INTO movements(account_id,kind,amount,occurred_on) VALUES(1,'income','abc','2024-01-01');"
run ERR "loan_id on a non-loan kind"        "INSERT INTO movements(account_id,kind,amount,occurred_on,loan_id) VALUES(1,'income',100,'2024-01-01',1);"
run OK  "loan_id on a loan kind"            "INSERT INTO movements(account_id,kind,amount,occurred_on,loan_id) VALUES(1,'loan_received',100,'2024-01-01',1);"
run ERR "loan_id pointing at no loan"       "INSERT INTO movements(account_id,kind,amount,occurred_on,loan_id) VALUES(1,'loan_received',100,'2024-01-01',9999);"
run ERR "account_id pointing at no account" "INSERT INTO movements(account_id,kind,amount,occurred_on) VALUES(9999,'income',100,'2024-01-01');"

head1 "6. loans"
run OK  "informal credit"                   "INSERT INTO loans(contact_id,direction,kind,principal,granted_on) VALUES(1,'borrowed','informal',50000,'2024-02-01');"
run ERR "unknown direction"                 "INSERT INTO loans(contact_id,direction,kind,principal,granted_on) VALUES(1,'sideways','informal',50000,'2024-02-01');"
run ERR "unknown loan kind"                 "INSERT INTO loans(contact_id,direction,kind,principal,granted_on) VALUES(1,'borrowed','crypto',50000,'2024-02-01');"
run ERR "principal of zero"                 "INSERT INTO loans(contact_id,direction,kind,principal,granted_on) VALUES(1,'borrowed','informal',0,'2024-02-01');"
run ERR "negative interest rate"            "INSERT INTO loans(contact_id,direction,kind,principal,interest_rate_bp,granted_on) VALUES(2,'borrowed','formal',1000,-5,'2024-02-01');"
run OK  "formal loan with rate"             "INSERT INTO loans(contact_id,direction,kind,principal,interest_rate_bp,granted_on,due_on) VALUES(2,'borrowed','formal',100000,1800,'2024-02-01','2025-02-01');"
run ERR "unknown status"                    "INSERT INTO loans(contact_id,direction,kind,principal,granted_on,status) VALUES(1,'borrowed','informal',100,'2024-02-01','wat');"

head1 "7. loan_payments"
run OK  "scheduled obligation"              "INSERT INTO loan_payments(loan_id,seq,amount,due_on) VALUES(1,1,25000,'2024-03-01');"
run ERR "duplicate position in schedule"    "INSERT INTO loan_payments(loan_id,seq,amount,due_on) VALUES(1,1,25000,'2024-04-01');"
run ERR "sequence of zero"                  "INSERT INTO loan_payments(loan_id,seq,amount,due_on) VALUES(1,0,25000,'2024-04-01');"
run ERR "amount of zero"                    "INSERT INTO loan_payments(loan_id,seq,amount,due_on) VALUES(1,2,0,'2024-04-01');"
run OK  "payment made before it falls due"  "INSERT INTO loan_payments(loan_id,seq,amount,due_on,paid_on) VALUES(1,2,25000,'2024-04-01','2024-03-15');"
run ERR "loan_id pointing at no loan"        "INSERT INTO loan_payments(loan_id,seq,amount,due_on) VALUES(9999,1,100,'2024-04-01');"

# --- 4. referential actions --------------------------------------------------
head1 "8. Referential actions"
# A contact that already has movements or loans must not be deletable: the
# foreign keys are RESTRICT, so the delete is refused and the history is kept.
run ERR "contact with movements cannot be deleted" "DELETE FROM contacts WHERE id = 1;"
run ERR "account with movements cannot be deleted"  "DELETE FROM accounts WHERE id = 1;"
run OK  "loan can still be cancelled, not deleted"   "UPDATE loans SET status='cancelled' WHERE id=1;"

# --- 5. the guard ------------------------------------------------------------
head1 "9. Guard: refuses a non-empty database"
GUARD_DB="${WORKDIR}/guard.db"
sqlite3 "$GUARD_DB" "CREATE TABLE preexisting(a);"
if sqlite3 -bail "$GUARD_DB" < "$MIGRATION" >/dev/null 2>"${WORKDIR}/guard_err.txt"; then
  red "  FAIL  migration applied on top of an existing schema"; FAIL=$((FAIL + 1))
else
  if grep -q "already contains tables" "${WORKDIR}/guard_err.txt"; then
    green "  PASS  refused with a clear message: $(head -1 "${WORKDIR}/guard_err.txt")"; PASS=$((PASS + 1))
  else
    red "  FAIL  refused, but the message was unclear: $(head -1 "${WORKDIR}/guard_err.txt")"; FAIL=$((FAIL + 1))
  fi
  LEFTOVER="$(sqlite3 "$GUARD_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='accounts';")"
  [ "$LEFTOVER" = "0" ] && { green "  PASS  no partial schema was left behind"; PASS=$((PASS+1)); } \
                        || { red "  FAIL  a partial schema was applied"; FAIL=$((FAIL+1)); }
  GV="$(sqlite3 "$GUARD_DB" "PRAGMA user_version;")"
  [ "$GV" = "0" ] && { green "  PASS  schema version was not stamped"; PASS=$((PASS+1)); } \
                  || { red "  FAIL  schema version was stamped on a refused migration"; FAIL=$((FAIL+1)); }
fi

# The same guard must hold when the client keeps going after an error instead
# of aborting, which is what a driver that ignores errors would do.
head1 "10. Guard: refuses a non-empty database even without bail-on-error"
NOBAIL_DB="${WORKDIR}/nobail.db"
sqlite3 "$NOBAIL_DB" "CREATE TABLE preexisting(a);"
sqlite3 "$NOBAIL_DB" < "$MIGRATION" >/dev/null 2>&1
NB_TABLES="$(sqlite3 "$NOBAIL_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name <> 'preexisting';")"
NB_VERSION="$(sqlite3 "$NOBAIL_DB" "PRAGMA user_version;")"
if [ "$NB_TABLES" = "0" ] && [ "$NB_VERSION" = "0" ]; then
  green "  PASS  refused, and the database is untouched"; PASS=$((PASS + 1))
else
  red "  FAIL  a client that continues past errors applied the schema (tables=${NB_TABLES}, version=${NB_VERSION})"; FAIL=$((FAIL + 1))
fi

# --- summary -----------------------------------------------------------------
head1 "Summary"
echo "  passed: ${PASS}"
echo "  failed: ${FAIL}"
if [ "$FAIL" -eq 0 ]; then
  green "  ALL CHECKS PASSED"; exit 0
else
  red "  THERE ARE FAILURES"; exit 1
fi
