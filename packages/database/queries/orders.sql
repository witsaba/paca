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
