-- The queries of the movements of the book.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it. The write tells the truth by itself -- the constraints of the
-- table -- and the library maps that truth to its own errors.
--
-- The balance of the accounts is a sum, and the sum is done by the database:
-- the money of a person is read from the rows of the book, and the rows are
-- the ones the library wrote.

-- name: RecordEntry :one
INSERT INTO ledger_entries (business_id, debit_account_id, credit_account_id, amount, occurred_on, description)
VALUES (sqlc.arg(business_id), sqlc.arg(debit_account_id), sqlc.arg(credit_account_id),
        sqlc.arg(amount), sqlc.arg(occurred_on)::date, sqlc.arg(description))
RETURNING *;

-- name: ListEntriesByBusiness :many
SELECT *
  FROM ledger_entries
 WHERE business_id = sqlc.arg(business_id)
   AND occurred_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
 ORDER BY occurred_on, id;

-- name: ListAccountBalances :many
SELECT a.id AS account_id,
       a.kind,
       a.name,
       COALESCE(SUM(CASE WHEN e.debit_account_id = a.id THEN e.amount ELSE 0 END), 0)::bigint AS debits,
       COALESCE(SUM(CASE WHEN e.credit_account_id = a.id THEN e.amount ELSE 0 END), 0)::bigint AS credits
  FROM accounts a
  LEFT JOIN ledger_entries e
    ON (e.debit_account_id = a.id OR e.credit_account_id = a.id)
   AND e.business_id = a.business_id
   AND e.occurred_on <= sqlc.arg(as_of)::date
 WHERE a.business_id = sqlc.arg(business_id)
 GROUP BY a.id, a.kind, a.name
 ORDER BY a.id;

-- The money of a period: what the person earned and what they spent
-- between two days.
--
-- The income is the money that landed in the accounts of an income, and the
-- expenses are the money that landed in the accounts of an expense. Each
-- movement is counted once in each of the two sums, and a movement between
-- two accounts of the same kind is counted once: the two sides of a
-- movement are read apart, never joined into one row.
--
-- name: SummarizePeriod :one
SELECT income.total AS income,
       expenses.total AS expenses
  FROM (SELECT COALESCE(SUM(e.amount), 0)::bigint AS total
          FROM ledger_entries e
          JOIN accounts a ON a.id = e.credit_account_id
         WHERE e.business_id = sqlc.arg(business_id)
           AND e.occurred_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
           AND a.kind = 'income') AS income,
       (SELECT COALESCE(SUM(e.amount), 0)::bigint AS total
          FROM ledger_entries e
          JOIN accounts a ON a.id = e.debit_account_id
         WHERE e.business_id = sqlc.arg(business_id)
           AND e.occurred_on BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
           AND a.kind = 'expense') AS expenses;
