-- The route of the round of a day: the stops in the order the agent will
-- visit the clients in, and what each stop costs.
--
-- The route is a fact of the business: the agent computes it outside, where
-- it knows the streets, and it writes it back into the round. The route
-- carries the stops of the day, and the stop carries the order it visits
-- and the money of the transport to that client.
--
-- The stops of the route are rows of their own table: the orders of the
-- round, in the order the agent will visit them, with the cost of each
-- one. The position of a stop is the order the agent visits it in.
--
-- The cost of a stop is the money of the book: whole pesos. The check
-- `route_stops_cost_check` holds that the cost is never negative, and the
-- domain is the one that decides it: a stop the owner walks to costs
-- nothing, and no stop costs less than nothing.
--
-- The route of a business and a day is written ONCE: the unique index
-- `routes_one_route_per_business_and_day` refuses a second route of the
-- same day, and the domain refuses it too with its own error.
--
-- The order of a stop is an order of the business that owns the route, and
-- the composite foreign key `route_stops_order_of_its_business_fkey`
-- refuses the stop of a foreign order, the way
-- `orders_party_of_its_business_fkey` refuses the party of a foreign order.
--
-- The migration of the orders is not edited: the rules of the route arrived
-- with this migration, and the schema of a table that exists is never
-- rewritten.

-- +goose Up
CREATE TABLE routes (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    day         DATE        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX routes_one_route_per_business_and_day ON routes (business_id, day);

CREATE UNIQUE INDEX routes_business_and_id ON routes (business_id, id);

CREATE TABLE route_stops (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    route_id    BIGINT      NOT NULL,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    order_id    BIGINT      NOT NULL,
    position    INTEGER     NOT NULL,
    cost        BIGINT      NOT NULL,

    CONSTRAINT route_stops_position_check CHECK (position > 0),
    CONSTRAINT route_stops_cost_check CHECK (cost >= 0)
);

CREATE UNIQUE INDEX route_stops_one_position_per_route ON route_stops (route_id, position);

ALTER TABLE route_stops
    ADD CONSTRAINT route_stops_route_of_its_business_fkey
    FOREIGN KEY (business_id, route_id) REFERENCES routes (business_id, id);

ALTER TABLE route_stops
    ADD CONSTRAINT route_stops_order_of_its_business_fkey
    FOREIGN KEY (business_id, order_id) REFERENCES orders (business_id, id);

-- +goose Down
ALTER TABLE route_stops
    DROP CONSTRAINT route_stops_order_of_its_business_fkey;

ALTER TABLE route_stops
    DROP CONSTRAINT route_stops_route_of_its_business_fkey;

DROP INDEX route_stops_one_position_per_route;

DROP TABLE route_stops;

DROP INDEX routes_business_and_id;

DROP INDEX routes_one_route_per_business_and_day;

DROP TABLE routes;
