-- =============================================================================
-- PACA — 001_initial_schema.sql
--
-- Personal finance manager for the Colombian market.
-- Storage engine: SQLite. Currency: COP (Colombian pesos).
--
-- Model: Single Table Inheritance (Model B). Each entity lives in one table; a
-- `kind` discriminator selects the subtype, and a CHECK constraint makes the
-- subtype column set valid for that kind. SQLite has no table inheritance, so
-- this is the declarative way to keep "one base table plus subtype columns"
-- free of invalid states.
--
-- Requires: SQLite >= 3.37.0 (STRICT tables).
--   The 3.53.0 floor is NOT needed to create this schema. It is only needed to
--   evolve it later without a full table rebuild; see README.md.
--
-- Apply (bail on error is REQUIRED, see "Safety" below):
--   sqlite3 -bail paca.db < packages/db/migrations/001_initial_schema.sql
--
-- Safety
--   This migration is atomic: all DDL runs in one transaction, so a failure
--   leaves the database exactly as it was. It refuses to run against a
--   database that already holds tables, and it stamps the schema version in
--   PRAGMA user_version for the migration runner to check later.
--
--   The `-bail` flag matters. The sqlite3 CLI keeps going after an error by
--   default, and a migration file that continues past a failed statement can
--   leave a half-applied schema. With -bail the script stops at the first
--   error and the open transaction is rolled back when the connection closes.
--   Any future runner must behave the same way: abort on the first error.
--
-- Conventions
--   * Surrogate key  : INTEGER PRIMARY KEY (a rowid alias -- fastest lookup).
--   * Money          : INTEGER Colombian pesos. Never a float.
--   * Business date  : TEXT, ISO-8601 'YYYY-MM-DD'. Validated with
--                      strftime('%Y-%m-%d', d) IS d, which rejects malformed
--                      input and impossible calendar dates such as 2024-02-30.
--                      'IS' rather than '=': a malformed date makes strftime
--                      return NULL, and a NULL comparison would satisfy a
--                      CHECK constraint instead of failing it.
--   * Audit timestamp: TEXT, ISO-8601 'YYYY-MM-DDTHH:MM:SS.sssZ' (UTC).
--   * Booleans       : INTEGER 0/1 with a CHECK constraint.
--   * Every table    : STRICT, so a wrong type is rejected instead of coerced.
-- =============================================================================

-- Foreign key enforcement is off by default in SQLite. This pragma is a no-op
-- inside a transaction, so it must be set before BEGIN.
PRAGMA foreign_keys = ON;

BEGIN IMMEDIATE;

-- -----------------------------------------------------------------------------
-- Precheck: record how many user tables already exist.
--
-- The count is not checked here. It is checked at the very END of this
-- transaction, because the check has to be the last thing that can fail.
--
-- Two failure modes have to be handled, and they need different placement:
--
--   * A CHECK violation aborts only the offending statement. The sqlite3 CLI
--     then carries on with the next one, so a guard that aborts early leaves
--     the rest of the DDL to run.
--   * RAISE(ROLLBACK) inside a trigger rolls the whole transaction back, but
--     the CLI still continues, and the statements that follow it then run
--     outside any transaction, where each one commits on its own.
--
-- Checking at the end defeats both: a failure rolls back every change this
-- migration made, including the schema version, whether the client aborts on
-- the first error or keeps going. Verified against the sqlite3 CLI with and
-- without -bail.
--
-- The bookkeeping table is temporary, so it never appears in the schema of
-- the migrated database and the connection drops it when it closes.
-- -----------------------------------------------------------------------------
CREATE TEMP TABLE _paca_precheck (
  preexisting_user_tables INTEGER NOT NULL
) STRICT;

INSERT INTO _paca_precheck (preexisting_user_tables)
SELECT COUNT(*) FROM sqlite_master
WHERE type = 'table' AND name NOT LIKE 'sqlite_%';

-- -----------------------------------------------------------------------------
-- accounts -- where money is held.
--
-- kind = 'nequi'  -> phone required; the bank columns must be NULL.
-- kind = 'bank'   -> bank_code, account_number, account_type required.
-- kind = 'cash'   -> no subtype column may be populated.
--
-- The kind CHECK and the shape CHECK are separate on purpose: the kind CHECK
-- keeps the discriminator honest, the shape CHECK keeps the column set honest.
--
-- Both are NAMED. An unnamed inline CHECK cannot be dropped or replaced, so
-- naming them is what makes it possible to add a new kind later with
-- ALTER TABLE alone, on SQLite 3.53.0 and newer. See "Extending the schema"
-- in the README.
-- -----------------------------------------------------------------------------
CREATE TABLE accounts (
  id             INTEGER PRIMARY KEY,
  kind           TEXT    NOT NULL CONSTRAINT accounts_kind
                 CHECK (kind IN ('nequi', 'bank', 'cash')),
  name           TEXT    NOT NULL CHECK (length(trim(name)) > 0),
  currency       TEXT    NOT NULL DEFAULT 'COP'
                 CHECK (length(currency) = 3 AND currency = upper(currency)),
  opened_on      TEXT    NOT NULL
                 CHECK (strftime('%Y-%m-%d', opened_on) IS opened_on),
  closed_on      TEXT
                 CHECK (closed_on IS NULL OR
                        strftime('%Y-%m-%d', closed_on) IS closed_on),
  is_active      INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
  notes          TEXT,

  -- subtype: nequi
  phone          TEXT,
  holder         TEXT,

  -- subtype: bank
  bank_code      TEXT,
  account_number TEXT,
  account_type   TEXT CHECK (account_type IS NULL OR
                             account_type IN ('ahorro', 'corriente')),

  created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

  -- An account that is closed must say when it was closed.
  CHECK (is_active = 1 OR closed_on IS NOT NULL),

  -- The subtype column set must match the discriminator exactly.
  CONSTRAINT accounts_shape CHECK (
       (kind = 'nequi' AND phone IS NOT NULL
                           AND holder IS NULL
                           AND bank_code IS NULL
                           AND account_number IS NULL
                           AND account_type IS NULL)
    OR (kind = 'bank'  AND phone IS NULL
                           AND holder IS NULL
                           AND bank_code IS NOT NULL
                           AND account_number IS NOT NULL
                           AND account_type IS NOT NULL)
    OR (kind = 'cash'  AND phone IS NULL
                           AND holder IS NULL
                           AND bank_code IS NULL
                           AND account_number IS NULL
                           AND account_type IS NULL)
  )
) STRICT;

-- A Nequi account is identified by its phone number.
CREATE UNIQUE INDEX idx_accounts_nequi_phone
  ON accounts (phone) WHERE kind = 'nequi';

-- A bank account is identified by bank plus account number.
CREATE UNIQUE INDEX idx_accounts_banco_identity
  ON accounts (bank_code, account_number) WHERE kind = 'bank';

-- -----------------------------------------------------------------------------
-- contacts -- people and entities involved in a transaction.
--
-- kind = 'person' -> id_document (cédula) required; tax_id and legal_name NULL.
-- kind = 'entity' -> tax_id (NIT) and legal_name required; id_document NULL.
--
-- phone is required for both kinds: a contact is someone you can reach.
-- -----------------------------------------------------------------------------
CREATE TABLE contacts (
  id             INTEGER PRIMARY KEY,
  kind           TEXT    NOT NULL CONSTRAINT contacts_kind
                 CHECK (kind IN ('person', 'entity')),
  name           TEXT    NOT NULL CHECK (length(trim(name)) > 0),
  phone          TEXT    NOT NULL CHECK (length(trim(phone)) > 0),
  id_document    TEXT,
  tax_id         TEXT,
  legal_name     TEXT,
  notes          TEXT,
  created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

  CONSTRAINT contacts_shape CHECK (
       (kind = 'person' AND id_document IS NOT NULL
                           AND tax_id IS NULL
                           AND legal_name IS NULL)
    OR (kind = 'entity' AND id_document IS NULL
                           AND tax_id IS NOT NULL
                           AND legal_name IS NOT NULL)
  )
) STRICT;

-- An entity is unambiguously identified by its NIT.
CREATE UNIQUE INDEX idx_contacts_entity_tax_id
  ON contacts (tax_id) WHERE kind = 'entity';

-- A natural person's cédula is NOT unique: the same person may legitimately
-- appear twice (for example as a service provider and as a lender). Lookups by
-- cédula are indexed but not constrained.
CREATE INDEX idx_contacts_person_id_document
  ON contacts (id_document) WHERE kind = 'person';

-- -----------------------------------------------------------------------------
-- loans -- a debt contract. Direction says which side of it PACA is on.
--
--   borrowed  PACA owes the contact: an informal credit from the gardener, or
--             a formal loan from a financial institution.
--   lent      the contact owes PACA.
--
-- principal is the amount originally lent. interest_rate_bp is the agreed
-- annual rate in basis points (100 = 1%); NULL means no rate is recorded.
--
-- status is maintained by the application and reconciled against the derived
-- outstanding balance; README.md gives the reconciliation query.
--
-- A loan is never deleted once it has movements: movements.loan_id is
-- RESTRICT. To retire a loan, set status = 'cancelled'.
-- -----------------------------------------------------------------------------
CREATE TABLE loans (
  id                INTEGER PRIMARY KEY,
  contact_id        INTEGER NOT NULL
                    REFERENCES contacts (id) ON DELETE RESTRICT,
  direction         TEXT    NOT NULL CONSTRAINT loans_direction
                    CHECK (direction IN ('borrowed', 'lent')),
  kind              TEXT    NOT NULL CONSTRAINT loans_kind
                    CHECK (kind IN ('informal', 'formal')),
  principal         INTEGER NOT NULL CHECK (principal > 0),
  interest_rate_bp  INTEGER CHECK (interest_rate_bp IS NULL OR interest_rate_bp >= 0),
  granted_on        TEXT    NOT NULL
                    CHECK (strftime('%Y-%m-%d', granted_on) IS granted_on),
  due_on            TEXT
                    CHECK (due_on IS NULL OR
                           strftime('%Y-%m-%d', due_on) IS due_on),
  status            TEXT    NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'settled', 'cancelled')),
  notes             TEXT,
  created_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

-- Who owes whom, per contact.
CREATE INDEX idx_loans_contact
  ON loans (contact_id);

-- Open loans, for the outstanding-balance report.
CREATE INDEX idx_loans_status
  ON loans (status, direction);

-- -----------------------------------------------------------------------------
-- movements -- every money event. This is the table that grows.
--
-- kind determines the sign of the cash effect; `amount` is always positive so
-- the direction of the flow is never ambiguous:
--
--   income         money in from outside       (+)
--   expense        money out to outside        (-)
--   transfer_in    money in from my account    (+)
--   transfer_out   money out to my account     (-)
--   loan_received  cash in for a loan I owe    (+)
--   loan_lent      cash out for a loan I made  (-)
--   loan_repayment cash out to repay a loan    (-)
--   loan_recovered cash in from a loan I made  (+)
--
-- loan_id links a movement to the loan it belongs to. A repayment movement and
-- its loan_payments row describe one event from two angles: the movement is
-- the cash that actually moved, the loan payment is the obligation. When both
-- exist the amounts may differ, which is how a partial payment is recorded.
-- -----------------------------------------------------------------------------
CREATE TABLE movements (
  id                      INTEGER PRIMARY KEY,
  account_id              INTEGER NOT NULL
                          REFERENCES accounts (id) ON DELETE RESTRICT,
  kind                    TEXT    NOT NULL CONSTRAINT movements_kind
                          CHECK (kind IN ('income', 'expense',
                                          'transfer_in', 'transfer_out',
                                          'loan_received', 'loan_lent',
                                          'loan_repayment', 'loan_recovered')),
  amount                  INTEGER NOT NULL CHECK (amount > 0),
  occurred_on             TEXT    NOT NULL
                          CHECK (strftime('%Y-%m-%d', occurred_on) IS occurred_on),
  category                TEXT,
  counterparty_contact_id INTEGER REFERENCES contacts (id) ON DELETE SET NULL,
  loan_id                 INTEGER REFERENCES loans (id) ON DELETE RESTRICT,
  notes                   TEXT,
  created_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

  -- A movement may only be attached to a loan for a loan-related kind.
  CHECK (loan_id IS NULL OR kind IN ('loan_received', 'loan_lent',
                                     'loan_repayment', 'loan_recovered'))
) STRICT;

-- Per-account history and the per-account balance query (covering).
CREATE INDEX idx_movements_account_date
  ON movements (account_id, occurred_on);

-- Time-range scans for reporting.
CREATE INDEX idx_movements_date
  ON movements (occurred_on);

-- Partial index: the income/expense report only ever touches two kinds, so
-- only those rows are indexed and the index stays small.
CREATE INDEX idx_movements_income_expense
  ON movements (occurred_on, kind, amount)
  WHERE kind IN ('income', 'expense');

-- All movements attached to one loan.
CREATE INDEX idx_movements_loan
  ON movements (loan_id) WHERE loan_id IS NOT NULL;

-- All movements with a given counterparty.
CREATE INDEX idx_movements_counterparty
  ON movements (counterparty_contact_id) WHERE counterparty_contact_id IS NOT NULL;

-- -----------------------------------------------------------------------------
-- loan_payments -- the repayment obligations of a loan.
--
-- A row records what is owed and when it falls due; `paid_on` is set once it
-- is settled. `movement_id` links the row to the cash movement that actually
-- settled it, when that money moved through a tracked account. It is NULL when
-- the repayment happened outside the tracked accounts, or has not happened yet.
--
-- `amount` is the scheduled obligation. The cash amount that was actually paid
-- lives in movements.amount, so a partial payment is representable.
--
-- The repayment schedule is owned by the loan and is deleted with it; the
-- movements that recorded the cash are not, and are protected by RESTRICT.
-- -----------------------------------------------------------------------------
CREATE TABLE loan_payments (
  id          INTEGER PRIMARY KEY,
  loan_id     INTEGER NOT NULL REFERENCES loans (id) ON DELETE CASCADE,
  seq         INTEGER NOT NULL CHECK (seq > 0),
  amount      INTEGER NOT NULL CHECK (amount > 0),
  due_on      TEXT    NOT NULL
              CHECK (strftime('%Y-%m-%d', due_on) IS due_on),
  paid_on     TEXT
              CHECK (paid_on IS NULL OR
                     strftime('%Y-%m-%d', paid_on) IS paid_on),
  movement_id INTEGER REFERENCES movements (id) ON DELETE SET NULL,
  notes       TEXT,
  created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),

  -- One obligation per position in the schedule.
  UNIQUE (loan_id, seq)
) STRICT;

-- The schedule of a loan.
CREATE INDEX idx_loan_payments_loan
  ON loan_payments (loan_id, seq);

-- Obligations still outstanding.
CREATE INDEX idx_loan_payments_unpaid
  ON loan_payments (due_on) WHERE paid_on IS NULL;

-- The cash movement that settled an obligation.
CREATE INDEX idx_loan_payments_movement
  ON loan_payments (movement_id) WHERE movement_id IS NOT NULL;

-- =============================================================================
-- Schema version. The migration runner compares this against the migration
-- files it knows about to decide what still has to be applied.
--
-- It is set BEFORE the final validation below, and inside the transaction, so
-- that a refused migration rolls the version back as well. Leaving it last
-- would stamp the version even when the migration was refused.
-- =============================================================================
PRAGMA user_version = 1;

-- =============================================================================
-- Final validation. This is the last statement that can fail, and it is
-- deliberately last: the check has to run after every change is made so that
-- a failure can roll all of them back.
--
-- If the database was not empty, RAISE(ROLLBACK) undoes the entire
-- migration: the tables, the indexes and the schema version above. A refused
-- migration leaves the database exactly as it was.
--
-- The trigger fires on the INSERT that re-reads the recorded count. A count
-- of zero means the database was empty and the migration may proceed.
-- =============================================================================
CREATE TRIGGER _paca_precheck_fail
BEFORE INSERT ON _paca_precheck
WHEN (SELECT preexisting_user_tables FROM _paca_precheck) > 0
BEGIN
  SELECT RAISE(ROLLBACK,
    'paca: database already contains tables; refusing to apply the initial schema');
END;

INSERT INTO _paca_precheck (preexisting_user_tables)
SELECT preexisting_user_tables FROM _paca_precheck;

-- The bookkeeping table and its trigger are temporary and are dropped here so
-- they do not survive in the session. IF EXISTS keeps this a no-op when the
-- rollback above already removed them.
DROP TRIGGER IF EXISTS _paca_precheck_fail;
DROP TABLE IF EXISTS _paca_precheck;

COMMIT;
