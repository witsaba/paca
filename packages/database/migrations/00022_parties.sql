-- The parties of the business: the person, the store and the farm that buy
-- from the bakery, or that the bakery buys from.
--
-- A party belongs to a business of its own: the one that knows the clients
-- and the providers of the bakery. The Personal Business of a person holds
-- the money of that person, and not the book of the bakery.
--
-- The legal human that represents a store or a farm is a FACT OF THE PARTY:
-- it is a name, and the legal figure it has when it has one. It is never a
-- second person: it is not a user of PACA, it is not a party of its own,
-- and it never becomes a row of the people of PACA. The table holds it in
-- the party it belongs to, and nowhere else.
--
-- The rules of the domain are that the kind of the party is one of the
-- three, that the name of the party is not empty, that the party holds at
-- least one of the two roles, that a store carries the legal human that
-- represents it, that a farm carries it only when it has one and that a
-- person carries none. The table holds the same rules as a defense in
-- depth.
--
-- The name of a party is NOT unique inside the business: the book never
-- refuses two parties that share a name, and the reading of the parties
-- lists them all. There is no index of the name here, and there is none on
-- purpose.
--
-- The optional columns are NULL when the value is not set, as in the table
-- of the people. An empty string is a different fact, and the library never
-- writes one.

-- +goose Up
CREATE TABLE parties (
    id               BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id      BIGINT      NOT NULL REFERENCES businesses (id),
    kind             TEXT        NOT NULL,
    name             TEXT        NOT NULL,
    roles            JSONB       NOT NULL DEFAULT '[]'::jsonb,
    phone            TEXT        NULL,
    email            TEXT        NULL,
    address          TEXT        NULL,
    legal_human_name TEXT        NULL,
    legal_figure     TEXT        NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT parties_kind_check CHECK (kind IN ('person', 'store', 'farm')),
    CONSTRAINT parties_name_not_empty_check CHECK (name <> ''),
    CONSTRAINT parties_roles_array_check CHECK (jsonb_typeof(roles) = 'array'),
    CONSTRAINT parties_roles_check CHECK (roles @> '["client"]'::jsonb OR roles @> '["provider"]'::jsonb),
    CONSTRAINT parties_legal_human_check CHECK (
        (kind = 'person'
            AND legal_human_name IS NULL
            AND legal_figure IS NULL)
        OR (kind = 'store'
            AND legal_human_name IS NOT NULL
            AND legal_human_name <> '')
        OR (kind = 'farm'
            AND (legal_human_name IS NULL OR legal_human_name <> ''))
    ),
    CONSTRAINT parties_phone_format_check CHECK (phone IS NULL OR phone ~ '^[0-9]{11,15}$'),
    CONSTRAINT parties_email_format_check
        CHECK (email IS NULL OR email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'),
    CONSTRAINT parties_optional_not_empty_check CHECK (
        (phone            IS NULL OR phone            <> '')
        AND (email          IS NULL OR email          <> '')
        AND (address        IS NULL OR address        <> '')
        AND (legal_human_name IS NULL OR legal_human_name <> '')
        AND (legal_figure   IS NULL OR legal_figure   <> '')
    )
);

-- +goose Down
DROP TABLE parties;
