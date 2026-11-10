-- The rules of the parties that the table of `00022_parties.sql` did not
-- hold.
--
-- The table holds the rules of the domain as a defense in depth: the domain
-- refuses what the table refuses, and the table refuses what the domain
-- refuses. Two of the rules were missing, and this migration closes them.
--
-- The first rule is the legal figure: a legal figure without the name of the
-- human it belongs to is a figure of nobody. The domain refuses it with
-- `ErrMissingLegalHuman`, and the table now refuses it too: the check
-- `parties_legal_figure_needs_human_check` holds that a row with a legal
-- figure carries the name of the human it belongs to.
--
-- The second rule is the roles: every role of the party is the client or
-- the provider. The check `parties_roles_check` of `00022_parties.sql` held
-- that the array holds one of the two roles, and it let an array that holds
-- one of them and another role through. The check of this migration holds
-- that every role of the array is one of the two, and that the array holds
-- at least one of them.
--
-- The migration of `00022_parties.sql` is not edited: it is already applied,
-- and the history of the schema is a history that is not rewritten. The
-- Down of this migration restores the rules of `00022_parties.sql`.

-- +goose Up
ALTER TABLE parties
    ADD CONSTRAINT parties_legal_figure_needs_human_check CHECK (
        legal_figure IS NULL OR legal_human_name IS NOT NULL
    );

ALTER TABLE parties
    DROP CONSTRAINT parties_roles_check;

ALTER TABLE parties
    ADD CONSTRAINT parties_roles_check CHECK (
        (roles @> '["client"]'::jsonb OR roles @> '["provider"]'::jsonb)
        AND roles <@ '["client", "provider"]'::jsonb
    );

-- +goose Down
ALTER TABLE parties
    DROP CONSTRAINT parties_roles_check;

ALTER TABLE parties
    ADD CONSTRAINT parties_roles_check CHECK (roles @> '["client"]'::jsonb OR roles @> '["provider"]'::jsonb);

ALTER TABLE parties
    DROP CONSTRAINT parties_legal_figure_needs_human_check;
