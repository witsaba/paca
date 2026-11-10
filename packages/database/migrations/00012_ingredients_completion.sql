-- The completion of an ingredient: what the owner declared about the thing
-- the business buys.
--
-- The unit is the unit the ingredient is bought in, and it is one of the
-- four the book knows: the gram, the kilogram, the package and the piece.
-- The weight of a package is what the owner declares, and it is never
-- assumed: a package without its weight says that the weight is not
-- declared yet.
--
-- The allergens are the allergens the ingredient carries. They travel as
-- JSONB, like the aliases: they are a list, and the list is what the
-- domain reads back.
--
-- The certification is a claim with the document that proves it: the
-- evidence is never empty. The two fats are two ingredients, each one with
-- its own certification, and the table never merges them.
--
-- The conversion is the grams of the raw that one gram of the prepared
-- ingredient takes. It is declared, never assumed: the owner decides the
-- factor, and the one to one of the raw is the decision she took for now.
--
-- The rules of the domain hold the columns as a defense in depth: the unit
-- is one of the four, the weight of a package is not negative, the
-- conversion is positive, and the certification carries its evidence or
-- nothing at all.

-- +goose Up
ALTER TABLE ingredients
    ADD COLUMN unit TEXT NOT NULL DEFAULT '',
    ADD COLUMN package_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN allergens JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN certification_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN certification_evidence TEXT NOT NULL DEFAULT '',
    ADD COLUMN conversion DOUBLE PRECISION NOT NULL DEFAULT 1;

ALTER TABLE ingredients
    ADD CONSTRAINT ingredients_unit_check
        CHECK (unit IN ('', 'g', 'kg', 'paquete', 'unidad')),
    ADD CONSTRAINT ingredients_package_grams_check
        CHECK (package_grams >= 0),
    ADD CONSTRAINT ingredients_allergens_check
        CHECK (jsonb_typeof(allergens) = 'array'),
    ADD CONSTRAINT ingredients_conversion_check
        CHECK (conversion > 0),
    ADD CONSTRAINT ingredients_certification_check
        CHECK ((certification_name = '' AND certification_evidence = '')
            OR (certification_name <> '' AND certification_evidence <> ''));

-- +goose Down
ALTER TABLE ingredients
    DROP CONSTRAINT ingredients_certification_check,
    DROP CONSTRAINT ingredients_conversion_check,
    DROP CONSTRAINT ingredients_allergens_check,
    DROP CONSTRAINT ingredients_package_grams_check,
    DROP CONSTRAINT ingredients_unit_check,
    DROP COLUMN conversion,
    DROP COLUMN certification_evidence,
    DROP COLUMN certification_name,
    DROP COLUMN allergens,
    DROP COLUMN package_grams,
    DROP COLUMN unit;
