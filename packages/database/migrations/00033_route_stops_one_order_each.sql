-- The route visits every order ONCE.
--
-- The total of the route is derived from the stops: it is never typed and
-- it is never a column. A route that names the same order in two stops
-- would carry the money of that visit twice, and the owner would be told
-- that the round costs more than it does.
--
-- The domain refuses it first, with its own error
-- (`order.ErrDuplicateStop`), and the unique index below holds the rule as
-- its defense in depth: it guards the writer that does not know the domain.
--
-- The migration of the routes (00032) is already applied, and the schema of
-- a table that exists is never rewritten: the rule of the stops arrives with
-- this migration.

-- +goose Up
CREATE UNIQUE INDEX route_stops_one_stop_per_order ON route_stops (route_id, order_id);

-- +goose Down
DROP INDEX route_stops_one_stop_per_order;
