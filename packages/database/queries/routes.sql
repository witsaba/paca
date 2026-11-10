-- The queries of the routes of the bakery.
--
-- The route and its stops are two tables of their own, and the queries of
-- them are here: the route is written with its stops, and the stops are
-- read in the order the agent will visit the clients in.
--
-- Every query writes or reads one row, and every one of them is a single
-- statement: the library does not read a row to find out whether it may
-- write it.

-- name: CreateRoute :one
INSERT INTO routes (business_id, day)
VALUES (sqlc.arg(business_id), sqlc.arg(day))
RETURNING *;

-- name: CreateRouteStop :one
INSERT INTO route_stops (route_id, business_id, order_id, position, cost)
VALUES (sqlc.arg(route_id), sqlc.arg(business_id), sqlc.arg(order_id), sqlc.arg(position), sqlc.arg(cost))
RETURNING *;

-- name: ListRoutesByBusiness :many
--
-- The routes of the business and of no other one, in the order of their
-- identity: the answer is the same every time it is read.
SELECT * FROM routes WHERE business_id = sqlc.arg(business_id) ORDER BY id;

-- name: ListRouteStopsByRoute :many
--
-- The stops of the route, in the order the agent will visit the clients in.
SELECT * FROM route_stops WHERE route_id = sqlc.arg(route_id) ORDER BY position;
