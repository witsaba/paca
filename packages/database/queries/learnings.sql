-- The queries of the learning log of the bakery.
--
-- The entry is one row of its own table: the day it is about, the batch, the
-- production it is about when it is about one run of the batch, what
-- happened, what was learned and the entry it answers.
--
-- Every query here reads or writes one row, and every one of them is a
-- single statement: the library does not read a row to find out whether it
-- may write it.
--
-- There is NO query that updates the words of an entry and NO query that
-- deletes one: the log is never rewritten, and a correction is a new entry
-- that answers the old one.
--
-- The name of the batch reads with the entry: the recipe is the owner of its
-- name, and the entry of the log never becomes a second truth about it.
--
-- The production of the entry is NULL when the entry is about the batch
-- alone, and the entry it answers is NULL when it answers no other one:
-- `sqlc.narg` for them, so the library passes NULL and the columns keep the
-- rule.

-- name: CreateLearning :one
WITH written AS (
    INSERT INTO learnings (business_id, day, recipe_id, production_id, happened, learned, answers_id)
    VALUES (
        sqlc.arg(business_id), sqlc.arg(day), sqlc.arg(recipe_id),
        sqlc.narg(production_id), sqlc.arg(happened), sqlc.arg(learned), sqlc.narg(answers_id)
    )
    RETURNING id, business_id, day, recipe_id, production_id, happened, learned, answers_id, created_at
)
SELECT written.id, written.business_id, written.day, written.recipe_id, written.production_id,
       written.happened, written.learned, written.answers_id, written.created_at,
       recipes.name AS recipe_name
FROM written
JOIN recipes ON recipes.id = written.recipe_id;

-- name: ListLearningsByBusiness :many
--
-- The entries of the business and of no other one, in the order they were
-- written: the answer is the same every time it is read. The entries that
-- were answered are in it: the log never hides the past.
SELECT learnings.id, learnings.business_id, learnings.day, learnings.recipe_id,
       learnings.production_id, learnings.happened, learnings.learned, learnings.answers_id,
       learnings.created_at,
       recipes.name AS recipe_name
FROM learnings
JOIN recipes ON recipes.id = learnings.recipe_id
WHERE learnings.business_id = sqlc.arg(business_id)
ORDER BY learnings.id;
