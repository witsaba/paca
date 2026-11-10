-- The queries of the orders of the bakery.
--
-- The order and its lines are two tables of their own, and the queries of
-- them are here: the order is written with its lines, and the lines are
-- read in the order the owner said them.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: CreateOrder :one
INSERT INTO orders (business_id, party_id, party_name, due_on, state)
VALUES (sqlc.arg(business_id), sqlc.arg(party_id), sqlc.arg(party_name), sqlc.arg(due_on), sqlc.arg(state))
RETURNING *;

-- name: CreateOrderLine :one
INSERT INTO order_lines (order_id, business_id, product_id, position, product_name, quantity, unit)
VALUES (sqlc.arg(order_id), sqlc.arg(business_id), sqlc.arg(product_id), sqlc.arg(position),
        sqlc.arg(product_name), sqlc.arg(quantity), sqlc.arg(unit))
RETURNING *;

-- name: ListOrdersByBusiness :many
--
-- The orders of the business and of no other one, in the order of their
-- identity: the answer is the same every time it is read.
SELECT * FROM orders WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: ListOrderLinesByOrder :many
--
-- The lines of the order, in the order the owner said them.
SELECT * FROM order_lines WHERE order_id = sqlc.arg(order_id) ORDER BY position;

-- name: DeliverOrder :one
--
-- The delivery of an order: the state of the order moves to the delivered
-- one, and the order is read back as it is now.
--
-- The delivery of an order is written with the ROUTE of the round, and the
-- state that is written is the one the domain decided.
--
-- The delivery moves an order that is OPEN, and only one: the state of the
-- row before the write is the one the domain owes, and an order that is not
-- open is never reopened. The order of the rows whose state is not the one
-- the delivery expects is left as it is, and the adapter tells the caller
-- what the row carries.
UPDATE orders
SET state = sqlc.arg(state), updated_at = now()
WHERE id = sqlc.arg(id) AND business_id = sqlc.arg(business_id) AND state = sqlc.arg(state_before)
RETURNING *;

-- name: TheStateOfTheOrder :one
--
-- The state the order carries now. It is read when the delivery did not find
-- the order it may deliver, so the adapter can tell an order that is not
-- there from an order that is not open any more.
SELECT state FROM orders WHERE id = sqlc.arg(id) AND business_id = sqlc.arg(business_id);
