-- The unit of every measurement the book stores is the unit the book knows.
--
-- The owner says the unit the way she says it ("kilos", "gramos"), and the
-- book stores the unit of the book ("kg", "g"): the domain is what turns
-- the wording into the unit, and the table holds the same rule as a
-- defense in depth.
--
-- The units of the book are the four of them: the gram, the kilogram, the
-- package and the piece. A unit the book does not know is refused by the
-- domain before the row is written, and the table refuses it too.

-- +goose Up
-- The rows that exist already are normalised before the rule is added:
-- the migration carries the same vocabulary the domain carries, so a
-- database that was written before the rule keeps its rows and gains the
-- rule. A row with a unit the book does not know stays as it is: the rule
-- is added NOT VALID, so it guards every row written from now on and the
-- owner decides what that old row means.
UPDATE purchase_lines
SET unit = CASE lower(btrim(unit))
    WHEN 'g' THEN 'g'
    WHEN 'gr' THEN 'g'
    WHEN 'gram' THEN 'g'
    WHEN 'gramo' THEN 'g'
    WHEN 'gramos' THEN 'g'
    WHEN 'grs' THEN 'g'
    WHEN 'kg' THEN 'kg'
    WHEN 'kgs' THEN 'kg'
    WHEN 'kilo' THEN 'kg'
    WHEN 'kilos' THEN 'kg'
    WHEN 'kilogramo' THEN 'kg'
    WHEN 'kilogramos' THEN 'kg'
    WHEN 'paquete' THEN 'paquete'
    WHEN 'paquetes' THEN 'paquete'
    WHEN 'unidad' THEN 'unidad'
    WHEN 'unidades' THEN 'unidad'
    WHEN 'pieza' THEN 'unidad'
    WHEN 'piezas' THEN 'unidad'
    ELSE unit
END
WHERE lower(btrim(unit)) <> unit
  AND lower(btrim(unit)) IN (
    'g', 'gr', 'gram', 'gramo', 'gramos', 'grs',
    'kg', 'kgs', 'kilo', 'kilos', 'kilogramo', 'kilogramos',
    'paquete', 'paquetes', 'unidad', 'unidades', 'pieza', 'piezas'
  );

UPDATE price_references
SET unit = CASE lower(btrim(unit))
    WHEN 'g' THEN 'g'
    WHEN 'gr' THEN 'g'
    WHEN 'gram' THEN 'g'
    WHEN 'gramo' THEN 'g'
    WHEN 'gramos' THEN 'g'
    WHEN 'grs' THEN 'g'
    WHEN 'kg' THEN 'kg'
    WHEN 'kgs' THEN 'kg'
    WHEN 'kilo' THEN 'kg'
    WHEN 'kilos' THEN 'kg'
    WHEN 'kilogramo' THEN 'kg'
    WHEN 'kilogramos' THEN 'kg'
    WHEN 'paquete' THEN 'paquete'
    WHEN 'paquetes' THEN 'paquete'
    WHEN 'unidad' THEN 'unidad'
    WHEN 'unidades' THEN 'unidad'
    WHEN 'pieza' THEN 'unidad'
    WHEN 'piezas' THEN 'unidad'
    ELSE unit
END
WHERE lower(btrim(unit)) <> unit
  AND lower(btrim(unit)) IN (
    'g', 'gr', 'gram', 'gramo', 'gramos', 'grs',
    'kg', 'kgs', 'kilo', 'kilos', 'kilogramo', 'kilogramos',
    'paquete', 'paquetes', 'unidad', 'unidades', 'pieza', 'piezas'
  );

ALTER TABLE purchase_lines
    ADD CONSTRAINT purchase_lines_unit_check CHECK (unit IN ('g', 'kg', 'paquete', 'unidad')) NOT VALID;

ALTER TABLE price_references
    ADD CONSTRAINT price_references_unit_check CHECK (unit IN ('g', 'kg', 'paquete', 'unidad')) NOT VALID;

-- +goose Down
ALTER TABLE price_references DROP CONSTRAINT price_references_unit_check;

ALTER TABLE purchase_lines DROP CONSTRAINT purchase_lines_unit_check;
