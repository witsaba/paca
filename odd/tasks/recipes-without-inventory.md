# PACA — The recipe that is saved without the inventory

Feature document for the change of the rule that holds a recipe hostage to the catalog of the
purchases: today a recipe may only name an ingredient that the business already bought, and the
owner wants a recipe to be knowledge first — a guide to learn, to teach, to publish — even when
nothing has been bought yet.

- **Project:** `packages/domain/recipe`, `packages/database`, `apps/mcp-server` (Go modules of the
  monorepository).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/recipes-without-catalog`,
  branch `feat/recipes-without-catalog`.
- **Scope:** the rule that binds the ingredients of a recipe to the catalog of the purchases, and
  the tool `record_recipe`. The courses, the marketing, the images and the onboarding trainings
  are **out of scope**: this document ships the capability that they would need, not the content.
- **Feature document of the bakery:** `odd/tasks/bakery-business.md`. This document changes one of
  its rules (the acceptance of T8) and does not touch the rest.

---

## Specs

Each spec is the user's own wording, quoted verbatim. The task entries expand them into
implementation rules and verification.

### S1 — The constraint of the inventory is not wanted

> Paca told me that is needed to have inventory for adding a new reciepe, but, i dont want to have this constraint

The catalog of the ingredients is the set of the names the business **buys**. It begins empty and
grows with the purchases. A recipe that may only name what has already been bought is a recipe
that cannot be written before the first market visit, and a recipe that cannot be written before
the first market visit is a recipe that is never written.

### S2 — The recipe is knowledge, and the knowledge is not stock

> I wanna use the reciepes for getting more information and can use it even as a learning for new people, or specify to the customers the ingredients of the product they are buying

A recipe is a guide. It is read to teach a new person, to answer what a product carries and to
keep a knowledge that outlives the stock. The knowledge of a recipe does not depend on the money
of the business, and the rule of the domain must say so.

This is the rule of the bakery feature document, quoted verbatim (`odd/tasks/bakery-business.md`,
S34):

> a reciepe is not only a list of ingredients but too a guide to learn and teach later.

### S3 — The scope of the work is the capability, not the content

> I mean, not the courses and marketing, but the feature that allows to save reciepes without restrictions.

The courses, the marketing material, the images and the internal onboarding trainings are
**not** built here. What is built is the capability they need: a recipe that can be written with
the ingredients the owner names, whether she bought them or not.

### S4 — The work happens on its own worktree

> Work in a apart git worktree.

The work happens on the branch `feat/recipes-without-catalog`, in the worktree
`/Users/braejan/workspace/witsaba/repositories/paca-worktrees/recipes-without-catalog`. The main
tree is never touched by this session.

### S5 — The rule that changes, in the words of the tree

The rule of the bakery feature document that this change relaxes, quoted verbatim
(`odd/tasks/bakery-business.md`, the acceptance of T8):

> The ingredient of a recipe is the ingredient of the catalog, never a second copy of the flour.

And the rule of the domain package that carries it (`packages/domain/recipe/recipe.go`):

> The ingredient of a recipe is the ingredient of the catalog: the recipe carries the identity of
> the ingredient and the grams of it, and never a second copy of the flour.

**What the rule becomes.** The ingredient of a recipe carries the **name the owner says** and its
grams. The catalog identity is a **memory**, not a requirement: it is kept only when the agent had
to disambiguate a name the catalog answers several times. The catalog is consulted **when the
money or the stock is needed** — the costing and the production resolve every ingredient of the
recipe by name, every time — and never when the recipe is written. The recipe never carries a
second copy of the *data* of the catalog — never its unit, never its conversions, never its price
— and it never carries a second copy of the flour: it carries the name of the flour.

**The rule that survives.** A recipe that names an ingredient the catalog does not hold is a
recipe whose **cost is unknown**, and it is never a recipe whose cost is zero. The costing layer
already knows how to say that: the cost line of an ingredient that the stock cannot value carries
`Unknown`, and the cost of the batch is `Complete = false` with the reason in `Missing`. The
production of a recipe whose ingredient is not in the catalog stays refused: a batch cannot be
baked out of an ingredient the business does not have.

---

## Tasks

| ID  | Specs     | Rule                                                                              | Route  | Commit |
|-----|-----------|-----------------------------------------------------------------------------------|--------|--------|
| T1  | S1, S2, S5 | The ingredient of the domain carries its name; the link to the catalog is optional | inline | `9ea90d6` |
| T2  | S1, S3     | The table of the ingredients of a recipe holds the name, and the link may be empty | inline | `9ea90d6` |
| T3  | S1, S2, S3 | The use case records a recipe whose ingredient the catalog does not hold          | inline | `2bcad05` |
| T4  | S3        | The tool `record_recipe` takes the name of the ingredient, and the link when it knows it | inline | `2bcad05` |

### T1 — The name of the ingredient of a recipe

**The rules:** S1, S2, S5.

**The acceptance:**

- `recipe.Ingredient` carries `Name` (the name the owner says) and `IngredientID` (the link to
  the catalog, zero when there is none).
- `recipe.NewIngredient` refuses an empty name and a non-positive weight of grams, and accepts an
  empty link.
- `recipe.New` refuses a recipe with no ingredient and an ingredient without its name; it never
  refuses an ingredient that is not in the catalog.
- The costing layer answers a line of an unlinked ingredient as `Unknown`, and it never counts
  two different unlinked ingredients as one: the deduplication of the grams of the recipe keys the
  linked ingredient by its identity and the unlinked one by its name.
- The tests of the package pin the new rule and the rule that survives.

### T2 — The storage of the name

**The rules:** S1, S3.

**The acceptance:**

- The migration adds the column `name` to `recipe_ingredients`, backfills it from the catalog and
  lets the link `ingredient_id` be empty.
- The table holds the same rules as the domain, as a defense in depth: the name is not empty.
- The repository writes the name and reads it, and the generated code is the one the schema
  describes.

### T3 — The use case that never refuses a recipe for the catalog

**The rules:** S1, S2, S3.

**The acceptance:**

- The use case resolves the name of every ingredient against the catalog of the business: an exact
  answer links the ingredient, an unknown or ambiguous answer keeps the name and leaves the link
  empty.
- The use case never refuses a recipe because the catalog does not hold a name.
- The answer of the tool carries the name of every ingredient and says which ingredients are not
  linked to the catalog, so the agent can tell the owner what Paca does not know yet.

### T4 — The tool that takes the name

**The rules:** S3.

**The acceptance:**

- The tool `record_recipe` takes the `name` of the ingredient (required) and the `ingredient_id`
  (optional), and the description of the tool says so.
- The answer carries the name and the link of every ingredient.
- The message of the domain for an ingredient without a name reaches the agent.

---

## Log

**L1** (2026-11-13): The request of the maintainer, verbatim:

> analyze and explore how Paca works with recipes. Paca told me that is needed to have inventory for adding a new reciepe, but, i dont want to have this constraint due to, I wanna use the reciepes for getting more information and can use it even as a learning for new people, or specify to the customers the ingredients of the product they are buying, create marketing, images, or even create my own internal onboarding trainings. Review how is this implemented and allows to start creating reciepes with no a inventory required. Check in engram first, explore and build this feature, I mean, not the courses and marketing, but the feature that allows to save reciepes without restrictions. Work in a apart git worktree.

**L2** (2026-11-13): What the memory held. The memory of the bakery (observations 4711, 4712, 4719
and 4736) says the recipe is a guide and not a list — "a recipe is a guide to learn and to teach,
not a list of ingredients" — and that the ingredient of a recipe is the identity of the catalog
plus its grams, never a second copy of the flour. The rule that the maintainer asks to relax is
therefore not a slip of the implementation: it is a deliberate rule of T8, written when the recipe
was only ever the first step of the costing of a batch.

**L3** (2026-11-13): The decision of this document. The rule is **relaxed, not deleted**: the
ingredient of a recipe carries the name the owner says, and the link to the catalog is optional.
The catalog keeps its ownership of the ingredient — its unit, its conversions, its price and its
stock stay in one place — and the recipe keeps its own ownership of the knowledge: the name, the
grams, the steps, the cues, the yield and the shelf life.

The consequences are deliberate and they are part of the acceptance:

- A recipe with an unlinked ingredient is **legitimate** and is written.
- The cost of a batch with an unlinked ingredient is **unknown**, not zero: the line carries
  `Unknown` and the batch is not `Complete`. The money is never invented.
- The **production** of a recipe with an unlinked ingredient stays **refused**: a batch cannot be
  baked out of an ingredient the business does not have. This is the rule that survives, and it
  is the one that keeps the stock honest.
- The agent is told, in the answer of the tool, which ingredients are not linked to the catalog, so
  that Paca can tell the owner that she has not bought that ingredient yet.

**L4** (2026-11-13): The slice boundary of the delivery. The feature landed in **two work units**
on the branch `feat/recipes-without-catalog`:

- `9ea90d6` — *the ingredient of a recipe carries its name and an optional link to the catalog*:
  the domain of the recipe, the storage of the table, the migration `00036`, the generated code
  and the container tests.
- `2bcad05` — *the recipe is written without the catalog of the purchases*: the port of the
  use case, the use case, the tool `record_recipe`, the messages of the agent and the tests of
  both.

The first unit is 741 authored changed lines: it is one vertical slice of the rule, and it was
not cut in two because the domain of the recipe and the table of its ingredients are the same
rule written in two languages — a slice that cut between them would leave a domain that cannot be
stored.

**L6** (2026-11-13): The correction of the maintainer, and the rule it settled. The maintainer
answered the first cut of this feature with a question that carried the real intention:

> but that is the idea, the reciept must to live unlinked, i mean, if the ingredients doenst exist, the reciept and the preparation is the same, why must to be linked always?

The first cut stored a link and read the cost **through that link**: a recipe written before the
first purchase would have stayed without money for ever, and the answer to the maintainer was a
"relink" tool. That was the wrong shape: it made the link the goal, and the goal of a recipe is to
be read, to be taught and to be published.

The rule that settled is:

- the recipe carries the **name** the owner says, and it **lives unlinked**;
- the catalog is consulted **when the money or the stock is needed**, and the resolution is
  **derived every time** by name: a recipe written before the purchase is costed and baked as soon
  as the owner buys the ingredient, and **nothing has to be linked again**;
- the link that the recipe may carry is the memory of which one of several ingredients the catalog
  answers the owner meant — it decides only when the name does not answer, and it is never a
  requirement;
- the money of an ingredient the catalog does not hold is **unknown and never zero**, and the
  production of a recipe the catalog does not hold stays **refused**.

The consequence that was withdrawn: there is **no "relink the recipe" tool**, and there is no
second truth about the ingredient of a recipe. The recipe is the owner's document, the catalog is
the book's, and the domain asks the catalog every time it needs the money or the stock.

**L7** (2026-11-13): The evidence of the correction.

- `packages/domain/recipe/resolve.go` (new): the resolution of the ingredients of a recipe against
  the catalog — the name decides, the link disambiguates, and two ingredients the recipe names
  twice are one ingredient of the batch.
- The tests that pin it: the cost of an unlinked recipe is derived when the ingredient is bought
  (`TestTheCostOfAnUnlinkedRecipeIsDerivedWhenTheIngredientIsBought`), the production of an
  unlinked recipe takes the stock
  (`TestTheProductionOfAnUnlinkedRecipeTakesTheStock`), and the recipe is written unlinked even
  when the catalog knows the name (`TestTheRecipeLivesUnlinkedEvenWhenTheCatalogKnowsTheName`).
- The answer of the tool `record_recipe` no longer carries the names of the ingredients without a
  link: with the rule above, a recipe without a link is the normal recipe, and the tool
  `read_batch_cost` is the one that says what the money still misses.

**L5** (2026-11-13): The verification of the slice.

- The gate of the three modules is green: the format, the analyzers, the linter, the build, the
  tests, the race detector, the coverage floor and the vulnerability scan of `packages/domain`,
  `packages/database` and `apps/mcp-server`.
- The structure test of the repository is green: 792 assertions, 0 failures.
- The **container tests of the database were not run**: there is no Docker daemon in the
  environment of the session. The container tests of the recipes were written for the new rule —
  the round trip of a recipe whose ingredient was never bought, and the foreign key that still
  refuses a link that is not an ingredient — and they are the proof that CI owes.
