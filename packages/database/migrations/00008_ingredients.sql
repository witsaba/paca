-- The catalog of the names the business buys.
--
-- The catalog begins empty and grows with what the owner buys: the name
-- she reads and the names she uses for it. An ingredient belongs to a
-- business, and the names of the catalog are the names the person reads in
-- her book.
--
-- The aliases are the names the owner uses for the thing she buys. They
-- travel as JSONB: they are a list of names, and the list is what the
-- domain reads back.
--
-- The rule of the domain is that two ingredients of the same business never
-- carry the same name, and that the name is compared the way a person
-- reads it: trimmed, and without case. The index holds the rule as a
-- defense in depth.
--
-- The lines of a purchase carry the ingredient they name when the catalog
-- knows it, and they carry nothing when it does not: the name of the line
-- is the name the owner said.

-- +goose Up
CREATE TABLE ingredients (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id),
    name        TEXT        NOT NULL,
    aliases     JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT ingredients_name_not_empty_check CHECK (name <> ''),
    CONSTRAINT ingredients_aliases_check CHECK (jsonb_typeof(aliases) = 'array')
);

CREATE UNIQUE INDEX ingredients_one_name_per_business ON ingredients (business_id, lower(btrim(name)));

ALTER TABLE purchase_lines ADD COLUMN ingredient_id BIGINT REFERENCES ingredients (id);

-- +goose Down
ALTER TABLE purchase_lines DROP COLUMN ingredient_id;

DROP INDEX ingredients_one_name_per_business;
DROP TABLE ingredients;
