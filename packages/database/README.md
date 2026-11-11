# The database library of the PACA monorepository.

The library holds the implementation of the persistence: the pool of
connections, the queries, the migrations of the schema and the repositories
that write the rows. The MCP server imports it today; the worker and the
REST backend will import it tomorrow.

It is a **library**. It publishes no port on the network, it runs no
container, and it starts no process. A service maps its own environment
onto the configuration of the library, opens the pool and uses the
repositories.

The models of the domain — what a person is, and the rules a person
carries — live in another module, `packages/domain`. This one depends on
it, never the other way around.

---

## The layout

```
packages/database/
  database.go         the migrator: the single writer of the schema. It
                      speaks in the model of the domain, so the service and
                      the library share one record of a migration
  postgres/           the adapter that talks to PostgreSQL: the pool, the
                      configuration and the repositories
  config/             the assembly of the connection string, shared by the
                      services
  internal/db/        the generated code: the row model and the queries
  queries/            the SQL of the queries, read by the generator
  migrations/         the history of the schema
  test/integration/   the container tests, against a real PostgreSQL
```

The dependency rule of the hexagonal architecture holds here, and the
toolchain enforces it: the module depends on `packages/domain`, and
`packages/domain` imports nothing but the standard library. The domain
cannot reach the generated code of the queries, because the visibility rule
of `internal` does not cross the boundary of a module.

The generated code lives in `internal/db`: it belongs to this module alone.
The public surface of the library is curated by hand, because the services
are other modules.

## The tables

### `users`

The person that PACA knows. It is the first table of the repository, and it
carries the schema and the rules of the people.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, never generated                        |
| `full_name`           | `text`       | Optional                                                     |
| `status`              | `text`       | Not null, default `onboarding`, checked against four states  |
| `status_observations` | `text`       | The reason of the closed states; the open states carry none   |
| `phone`               | `text`       | Optional, unique, digits only with the country code          |
| `email`               | `text`       | Optional, unique without case, the shape of an address, canonical |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The identifier is the national document ID of the person.** The caller
supplies it, and nothing in the library generates one: the number is not
sequential because it is not a counter, it is the number the person already
carries. The column is a `bigint`, because a ten-digit document number
should never meet the ceiling of a 32-bit integer.

**The status is checked by the database**, against the four states of the
domain: `onboarding`, `active`, `inactive` and `blocked`. A writer that
does not know the states cannot store one.

**The observation belongs to the status.** The two closed states —
`inactive` and `blocked` — carry the reason why the person is not using the
product, and the two open states carry none. A CHECK on the table holds the
rule, so no writer breaks it.

**The optional columns are NULL when the value is not known.** The domain
uses the empty string for the same fact, and the library never writes an
empty string to the database.

**The email and the phone are unique when they are set.** The indexes are
partial: a person may have no email and no phone, and two people without an
email do not collide. The email is compared without case, because
`Foo@x.com` and `foo@x.com` are the same mailbox.

**The email is stored in its canonical form.** The domain owns that rule: it
writes the lower case of the address, and the constraint of the table refuses
an address that is not in it. The migration of the identity brought the rows
that existed to that form.

**The formats are checked by the database too.** The phone holds digits only
and carries its country code — eleven to fifteen digits — and the email
holds the shape of an address.

**The timestamps carry the time of the system.** The database fills them on
insert. There is no trigger: whoever updates a row is responsible for
`updated_at` after that — and the library writes `updated_at = now()`
inside the statement of the update.

The rules that a table cannot hold — the cleaning of the phone, the shape
of the email, the map of the states — live in the module of the domain, and
the domain is the one that refuses them.

### `businesses`

The business that owns a book. It is the table of the Personal Business of
a person, and of the businesses that will come.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `owner_id`            | `bigint`     | Not null, the person who owns the business                    |
| `kind`                | `text`       | Not null, checked against `personal` and `business`          |
| `name`                | `text`       | Not null, never empty                                         |
| `currency`            | `text`       | Not null, default `COP`, never empty                          |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The identity of a business is generated by the database.** It is a counter
of the businesses and not the document of a person: the owner is the owner,
and it is not the key.

**A person has exactly one personal business.** The rule is in the domain,
and the table holds it as a defense in depth: the index
`businesses_one_personal_per_owner` is partial, and it refuses a second
personal business of the same owner while leaving the other kinds free. The
bakery of the future is a second row of this table, with an identity of its
own: the key space is the businesses, never the people.

### `accounts`

The accounts of a book: the cash, the bank, the salary, the food, the money
the person owes.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the account                  |
| `kind`                | `text`       | Not null, checked against the five kinds of the accounting    |
| `name`                | `text`       | Not null, never empty, unique inside the business             |
| `counterparty`        | `text`       | Optional: the other side of a debt                            |
| `due_date`            | `date`       | Optional: the day a debt falls due                            |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The name of an account is unique inside its business.** The person reads
the accounts of their book by name, and two accounts of the same business
that share a name are one account.

**The counterparty and the due date belong to a debt.** A debt is an account
that owes or is owed — a liability or an asset — and an account of any other
kind carries neither. The CHECK of the table holds the rule of the domain as
a defense in depth.

**There is no table of the loans.** A loan with a friend is a liability
account with a counterparty, and the balance of that account is what the
person still owes.

### `purchase_drafts`

The draft of a purchase: the candidate the agent proposes from what the
owner said, and the state it waits in.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the draft                    |
| `state`               | `text`       | Not null, default `proposed`, checked against the states      |
| `revision`            | `integer`    | Not null, default 1, positive                                 |
| `utterance`           | `text`       | Not null, never empty                                         |
| `lines`               | `jsonb`      | Not null, default `[]`, an array                              |
| `missing`             | `jsonb`      | Not null, default `[]`, an array                              |
| `ambiguous`           | `jsonb`      | Not null, default `[]`, an array                              |
| `inferred`            | `jsonb`      | Not null, default `[]`, an array                              |
| `day`                 | `date`       | Optional                                                      |
| `market`              | `text`       | Optional                                                      |
| `invoice`             | `text`       | Optional                                                      |
| `proposed_on`         | `date`       | Not null                                                      |
| `expires_on`          | `date`       | Not null, never before the day it was proposed                |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**A draft is a proposal, not a fact.** Its lines, what it still needs, what
it was not sure about and what Paca filled by herself travel as JSONB: they
are the reading of a sentence, and the money enters the book with the
purchase and not with the draft.

**The state is checked by the database**, against the states of the domain:
`proposed`, `confirmed`, `cancelled` and `expired`. A draft is confirmed
once, and the confirmation of a draft that changed under the eyes of the
owner is refused by the row that carries the revision she saw.

**A draft that waits can be dropped.** The row moves to the state
`cancelled` only when it carries the state of a draft that waits: a draft
that is confirmed or that is cancelled already does not wait any more, and
the row is not touched. The expiry is computed by the domain when the draft
is read — a draft that was neither confirmed nor cancelled seven days after
the day it was proposed reads as `expired` — and the table needs no job to
expire anything.

### `purchases`

The purchase: the fact of what the business bought, and the lines it
carries.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that bought                           |
| `draft_id`            | `bigint`     | Not null, the draft the purchase came from, unique            |
| `day`                 | `date`       | Not null                                                      |
| `market`              | `text`       | Optional                                                      |
| `invoice`             | `text`       | Optional                                                      |
| `utterance`           | `text`       | Not null, never empty                                         |
| `confirmed_with`      | `text`       | Not null, never empty: the word the owner used               |
| `total`               | `bigint`     | Not null, positive                                            |
| `invoice_id`          | `bigint`     | Optional: the invoice the purchase came from                   |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The purchase points at the invoice it came from.** The column is NULL when
the invoice has not arrived yet: the purchase is written when the owner
buys, and the invoice comes later. The update that points the purchase
carries the business of the invoice, and it touches only the purchase of
that business.

### `purchase_lines`

The lines of a purchase: what was bought, how much of it and what it cost.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `purchase_id`         | `bigint`     | Not null, the purchase the line belongs to                   |
| `position`            | `integer`    | Not null, positive, unique inside the purchase                |
| `name`                | `text`       | Not null, never empty: the name the owner said                |
| `quantity`            | `double precision` | Not null, positive                                      |
| `unit`                | `text`       | Not null, one of `g`, `kg`, `paquete`, `unidad`              |
| `total`               | `bigint`     | Not null, never negative                                     |
| `unit_price`          | `bigint`     | Not null, never negative                                     |
| `ingredient_id`       | `bigint`     | Optional: the ingredient of the catalog the name belongs to   |

**A purchase never changes.** The lines are rows of their own table, and
the position of a line is the order the owner said it in.

**The name of the line is the name the owner said.** The identity of the
ingredient is optional: it is there when the catalog knows the name, and it
is empty when it does not.

**The unit of the line is the unit of the book.** The owner may say the unit
the way she says it — "kilos", "gramos" — and the domain is what turns the
wording into the unit of the book (`g`, `kg`, `paquete`, `unidad`). The
check of the table holds the same rule as a defense in depth
(`purchase_lines_unit_check`).

### `stock_layers`

The layers of the stock: what a purchase brought, and what is left of it.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that bought                           |
| `ingredient_id`       | `bigint`     | Not null, the ingredient of the catalog the layer is of       |
| `purchase_id`         | `bigint`     | Not null, the purchase the layer comes from                  |
| `position`            | `integer`    | Not null, positive, unique inside the purchase: a layer comes from one line and from no other |
| `name`                | `text`       | Not null, never empty: the name the owner said               |
| `day`                 | `date`       | Not null: the day it was bought                              |
| `quantity`            | `double precision` | Not null, positive: what the layer bought              |
| `unit`                | `text`       | Not null, one of `g`, `kg`, `paquete`, `unidad`               |
| `remaining`           | `double precision` | Not null, never more than the quantity: what is left of it |
| `cost`                | `bigint`     | Not null, never negative: the money it cost, in whole pesos   |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The purchase is the fact and the layer is derived from it.** The layer is
written with the purchase it comes from, in the same transaction, and
nobody types a layer.

**The quantity and its unit travel together.** The unit is one of the four
the book knows, and what is left of a layer is never more than what it
bought. The layer is derived, and the unit it carries is the one of the
book: the way the owner said it is kept in the purchase, and the layer of
that purchase is in the unit of the book.

**The layer belongs to the business of its purchase.** The rule is the
domain's, and the query holds it as a defense in depth: the layers of a
business are read with the purchase of every layer, and a layer whose
purchase is of another business is never in the answer.

**The money of the layer is the money of the line of its purchase.** The
rule is the domain's, and the query reads the line of the purchase with
the layer: the adapter asks the domain whether the layer is the layer of
that line, and a row another writer broke with another money - or with
another quantity - is refused and never read as a layer of the stock. The
line is read with a join on the outside, so a layer whose line is not
there is refused too, and not hidden.

### `ingredients`

The catalog of the names the business buys.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that buys it                          |
| `name`                | `text`       | Not null, never empty, unique inside the business             |
| `aliases`             | `jsonb`      | Not null, default `[]`, an array of the names she uses        |
| `unit`                | `text`       | Not null, default `''`: `g`, `kg`, `paquete` or `unidad`      |
| `package_grams`       | `double precision` | Not null, default `0`: the weight of a package, in grams |
| `allergens`           | `jsonb`      | Not null, default `[]`, an array of the allergens it carries  |
| `certification_name`  | `text`       | Not null, default `''`: the claim the ingredient carries      |
| `certification_evidence` | `text`   | Not null, default `''`: the document that proves the claim    |
| `conversion`          | `double precision` | Not null, default `1`: the raw grams of one prepared gram |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The name of an ingredient is unique inside its business.** The index is
`ingredients_one_name_per_business`, and it compares the name the way a
person reads it: trimmed, and without case. The rule lives in the domain,
and the index holds it as a defense in depth.

**The aliases are the names the owner uses for the thing she buys.** They
travel as JSONB: they are a list of names, and the list is what the domain
reads back.

**The unit of an ingredient is one of the units of the book.** The check is
`ingredients_unit_check`, and it holds the four units — the gram, the
kilogram, the package and the piece — as a defense in depth. The unit is
empty while the owner has not declared it.

**The weight of a package is what the owner declares.** The check is
`ingredients_package_grams_check`, and the weight is never assumed: zero
means that the owner has not declared it yet.

**The allergens are the allergens the ingredient carries.** They travel as
JSONB, like the aliases, and the check `ingredients_allergens_check` holds
that they are a list.

**A certification carries its evidence.** The check
`ingredients_certification_check` refuses a claim without the document
that proves it, and a claim with an evidence that is not there. The two
fats are two rows, each one with its own certification.

**The conversion is the raw grams of one prepared gram.** The check
`ingredients_conversion_check` holds it positive, and the default is the
one to one the owner decided for now: the grams of a recipe are the raw
ones, and the owner declares any other factor she needs.

**The catalog of a business begins empty.** There is no seed: the first
purchase of the business creates the first name.

### `recipes`

The recipe of the bakery: the guide to learn and to teach, and not a list of
ingredients.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the recipe                   |
| `name`                | `text`       | Not null, never empty                                         |
| `yield_units`         | `integer`    | Not null, positive: the number of units the recipe makes     |
| `yield_grams`         | `double precision` | Not null, positive: the weight of one unit, in grams   |
| `shelf_life_days`     | `integer`    | Not null, positive: the days what the recipe makes keeps     |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The yield is a declared constant.** The number of units the recipe makes
and the weight of each one are columns of the row, and they are positive:
the yield is never measured per bake, and the production of a day is a fact
of its own.

**The sentence of the owner is optional.** The recipe is the record, and the
sentence is kept with it when she said one: the column carries an empty
string and never NULL.

**The recipe is read back whole.** The recipe carries its ingredients with
their grams and its steps in the order the owner said them, and they are
read with it: the cost of a batch is derived from the recipe and from the
layers of the stock, and a recipe read without its ingredients is not a
recipe.

### `recipe_ingredients`

The ingredients of a recipe, with their grams.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `recipe_id`           | `bigint`     | Not null, the recipe that names it                           |
| `position`            | `integer`    | Not null, positive, unique inside the recipe                 |
| `ingredient_id`       | `bigint`     | Not null, the ingredient of the catalog                       |
| `grams`               | `double precision` | Not null, positive                                       |

**The ingredient of a recipe is the ingredient of the catalog.** The column
carries a foreign key to `ingredients`, and the recipe never carries a
second copy of the flour. The table holds the rule of the domain as a
defense in depth.

**The position is the order the owner said the ingredient in**, and it is
unique inside the recipe: two ingredients of a recipe never share a place.

### `recipe_steps`

The steps of the process of a recipe.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `recipe_id`           | `bigint`     | Not null, the recipe that teaches it                         |
| `position`            | `integer`    | Not null, positive, unique inside the recipe                 |
| `instruction`         | `text`       | Not null, never empty: what the owner does in the step       |
| `cue`                 | `text`       | Not null, default `''`: the sign the owner looks for         |
| `minutes`             | `integer`    | Not null, default `0`: the time of the step                  |
| `temperature`         | `double precision` | Not null, default `0`: the temperature of the step     |
| `labor_kind`          | `text`       | Not null, default `hands_on`: the kind of the labor the step is |

**A step carries its time and its temperature only when the owner named
them.** The zero of the minutes and of the temperature is the absence of
them, and the table holds the same rule as a defense in depth: the
temperature never goes below the absolute zero.

**The kind of the labor of a step is one of the three kinds of the book.**
The labour of a batch is derived from the hours of the steps of the recipe,
and the kind is what says in which of the three kinds the hours of the step
are counted: `hands_on`, `oven` or `waiting`. The step the owner did not
qualify is the labour of the hands, and the check of the table holds the
same rule as a defense in depth (`recipe_steps_labor_kind_check`).

### `labors`

The labour of the owner: the hours she worked, in the three kinds of the
labour, and the value of the hour she declared.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business the labour belongs to                  |
| `day`                 | `date`       | Not null: the day the labour was worked                       |
| `hands_on_hours`      | `double precision` | Not null, default `0`: the hours of the hands          |
| `oven_hours`          | `double precision` | Not null, default `0`: the hours of the oven           |
| `waiting_hours`       | `double precision` | Not null, default `0`: the hours of the waiting        |
| `hour_value`          | `bigint`     | Not null, positive: the value of the hour, in whole pesos     |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The hours are held apart in the three kinds of the labour.** The hands, the
oven and the waiting are three columns, and at least one of them carries
hours: a labour without hours is nothing
(`labors_hours_present_check`).

**The hours are not negative.** A negative number of hours is a number
nobody worked, and the check of the table holds the rule of the domain as a
defense in depth (`labors_hours_not_negative_check`).

**The value of the hour is declared by the owner.** The money of the work of
a person is never a number PACA invents, and the column is positive
(`labors_hour_value_positive_check`).

**The money of the labour is not stored.** It is derived from the hours and
the value of the hour, the way the cost of a batch is derived from the
layers of the stock: a second truth about the same money is a truth that
drifts.

### `overheads`

The overhead of the house: the electricity the owner paid, the hours the
oven ran and the allocation she declared.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business the overhead belongs to                |
| `day`                 | `date`       | Not null: the day the overhead was declared                   |
| `electricity`         | `bigint`     | Not null, positive: the money of the electricity of the house, in whole pesos |
| `oven_hours`          | `double precision` | Not null, positive: the hours the oven ran              |
| `allocation`          | `double precision` | Not null: the share of the electricity that belongs to the oven |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The electricity of the house is declared by the owner.** The money of the
house is not a number PACA invents, and the column is positive
(`overheads_electricity_positive_check`).

**The hours of the oven are positive.** Without them the money of the oven
has no rate, and a rate nobody declared is a rate PACA invented
(`overheads_oven_hours_positive_check`).

**The allocation is a share.** It is the share of the electricity of the
house that belongs to the oven: it is greater than zero and not greater
than one (`overheads_allocation_share_check`).

**The money of the overhead is not stored.** It is derived from the
electricity and the allocation, the way the money of the labour is derived
from the hours and the value of the hour.

### `price_references`

The reference prices: what the market was seen at, where it was read and the
day it was read.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that researched it                     |
| `ingredient_id`       | `bigint`     | Not null, the ingredient of the catalog                       |
| `price`               | `bigint`     | Not null, positive: the price that was seen, in whole pesos  |
| `unit`                | `text`       | Not null, one of `g`, `kg`, `paquete`, `unidad`: the unit of that price |
| `market`              | `text`       | Optional: the market where the price was seen                 |
| `source`              | `text`       | Not null, never empty: the place the price was read           |
| `seen_on`             | `date`       | Not null: the day the price was seen                          |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**A reference is never the price of the ingredient.** The price the owner
pays comes from her own purchases: this table holds what the research of
Paca found, and the rule lives in the domain.

**The source is never empty.** A price without a source is a price nobody
can check, and the check of the table holds the rule of the domain as a
defense in depth.

**The unit of the reference is the unit of the book.** The research may name
the unit the way a person says it — "kilos" — and the domain is what turns
the wording into the unit of the book (`g`, `kg`, `paquete`, `unidad`). The
check of the table holds the same rule as a defense in depth
(`price_references_unit_check`).

**The ingredient of a reference is the ingredient of the catalog.** The
column carries a foreign key to `ingredients`, and the reference never
carries a second copy of the flour.

### `productions`

The production of the bakery: what came out of the oven on one day.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that made it                          |
| `recipe_id`           | `bigint`     | Not null, the recipe it was made of                           |
| `day`                 | `date`       | Not null: the day it was made                                |
| `grams_in`            | `double precision` | Not null, positive: the grams that went into it        |
| `money`               | `bigint`     | Not null, never negative: the money of the layers it consumed, in whole pesos |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The production is derived, never typed.** The grams that went in and the
money of the layers it consumed are read from the stock in the order of
the FIFO, and the units it made are the yield of the recipe: nobody types
a production.

**The production is written with the word of the owner.** The check of the
table holds the rule of the domain as a defense in depth
(`productions_confirmed_with_not_empty_check`), and the grams and the money
are positive (`productions_grams_in_positive_check`,
`productions_money_not_negative_check`).

### `production_layers`

The layers the production consumed: the trace of the FIFO.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `production_id`       | `bigint`     | Not null, the production it belongs to                        |
| `position`            | `integer`    | Not null, positive, unique inside the production: the order the layers left the stock in |
| `layer_id`            | `bigint`     | Not null, the layer of the stock it took from                 |
| `purchase_id`         | `bigint`     | Not null, the purchase the layer comes from                   |
| `quantity`            | `double precision` | Not null, positive: what it took, in the unit of the layer |
| `unit`                | `text`       | Not null, one of `g`, `kg`, `paquete`, `unidad`: the unit of that quantity |
| `grams`               | `double precision` | Not null, positive: what it took, in grams             |
| `money`               | `bigint`     | Not null, never negative: the money of what it took, in whole pesos |

**The quantity and its unit travel together.** The unit is the one of the
layer, and the check of the table holds the rule of the domain as a defense
in depth (`production_layers_unit_check`).

**The position is the order of the FIFO.** The oldest layer leaves the
stock first, and the unique index of the table holds the rule as a defense
in depth (`production_layers_one_position`).

### `production_units`

The units the production made: what came out of the oven.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `production_id`       | `bigint`     | Not null, the production it came out of                       |
| `position`            | `integer`    | Not null, positive, unique inside the production              |
| `grams`               | `double precision` | Not null, positive: the weight of the unit             |
| `expires_on`          | `date`       | Not null: the day it expires                                  |

**Every unit carries the day it expires.** The day is the day of the
production plus the shelf life of the recipe, and it is approximate until
the owner corrects it.

**The position is the order the units were made in**, and the unique index
of the table holds the rule as a defense in depth
(`production_units_one_position`).

### `wastes`

The waste of the bakery: what she lost, and what it was worth.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns it                          |
| `recipe_id`           | `bigint`     | Not null, the product it lost                                |
| `day`                 | `date`       | Not null: the day the waste happened                         |
| `quantity`            | `double precision` | Not null, positive and whole: the units it lost         |
| `unit`                | `text`       | Not null, `unidad`: the waste counts the units it lost       |
| `reason`              | `text`       | Not null, one of `burnt`, `spoiled`, `dropped`, `returned`: what a later analysis reads |
| `money`               | `bigint`     | Not null, never negative: the cost the waste releases, in whole pesos |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The reason is a column of its own.** It is what a later analysis reads,
and it is not buried inside a sentence: the check of the table holds the
four reasons of the book as a defense in depth (`wastes_reason_check`).

**The waste counts the units it lost.** The quantity is positive and whole
(`wastes_quantity_positive_check`, `wastes_quantity_whole_check`), and the
unit is the unit of a piece (`wastes_unit_check`): the book never converts
a weight into a number of units, and it never splits a unit.

**The money is derived, never typed.** It is the money of the units that
left the book: the money of the production they came out of, divided by
the units that production made. The check of the table holds the rule of
the domain as a defense in depth (`wastes_money_not_negative_check`).

### `waste_units` (gone)

The units that left the book as a waste were rows of a table of their own.
The sale is the second exit of the book, and the rule of the book is that a
unit leaves it once, whichever exit takes it: the table of the exits
(`unit_exits`) owns that rule today, and the migration `00020_sales.sql`
moved the rows of the waste into it.

### `unit_exits`

The exits of the book: the units that left it, and the exit that took each
one.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `unit_id`             | `bigint`     | Primary key, not null, the unit that left the book           |
| `exit_kind`           | `text`       | Not null, one of `waste`, `sale`, `consumption`, `donation`: the exit that took the unit |
| `exit_id`             | `bigint`     | Not null, the row of that exit                               |
| `production_id`       | `bigint`     | Not null, the production it came out of                       |
| `position`            | `integer`    | Not null, positive: the position the unit had in the production it came out of |
| `grams`               | `double precision` | Not null, positive: the weight of the unit              |
| `expires_on`          | `date`       | Not null: the day it expired on                               |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**One table owns the exits of the book.** The waste, the sale, the
consumption and the donation are four exits, and the rule of the book is
that a unit leaves it once, whichever exit takes it: the primary key over
the unit holds that rule, and a unit that already left is never taken
again — not as a sale, not as a waste, not as a consumption and not as a
donation.

**The position is the position the unit had in its production**, and the
unique index of the table holds it as a defense in depth
(`unit_exits_one_position`).

**The kind of the exit is one of the ones the book knows** (`waste`,
`sale`, `consumption`, `donation`), and the check of the table holds the
rule of the domain as a defense in depth (`unit_exits_kind_check`). The
migration `00021_exits.sql` opened the check to the two exits that came
last: the consumption of the owner and the donation.

### `sales`

The sale of the bakery: what she sold, and what it cost her.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns it                          |
| `recipe_id`           | `bigint`     | Not null, the product it sold                                |
| `day`                 | `date`       | Not null: the day the sale happened                          |
| `quantity`            | `double precision` | Not null, positive and whole: the units it sold         |
| `unit`                | `text`       | Not null, `unidad`: the sale counts the units it sold        |
| `money`               | `bigint`     | Not null, positive: the money the client paid, in whole pesos |
| `cost`                | `bigint`     | Not null, never negative: the cost the sale releases, in whole pesos |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The money of the sale is what the client paid.** It is positive
(`sales_money_positive_check`), and the price list of the products is a
later work: the book never guesses the money of a sale.

**The cost is derived, never typed.** It is the money of the units that
left the book: the money of the production they came out of, divided by
the units that production made. The check of the table holds the rule of
the domain as a defense in depth (`sales_cost_not_negative_check`).

**The sale counts the units it sold.** The quantity is positive and whole
(`sales_quantity_positive_check`, `sales_quantity_whole_check`), and the
unit is the unit of a piece (`sales_unit_check`): the book never converts
a weight into a number of units, and it never splits a unit.

**The sale is written with the word of the owner**, and the check of the
table holds the rule of the domain as a defense in depth
(`sales_confirmed_with_not_empty_check`).

### `exits`

The exit of a unit of the book: the consumption of the owner and the
donation.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns it                          |
| `recipe_id`           | `bigint`     | Not null, the product it took                                |
| `kind`                | `text`       | Not null, one of `consumption`, `donation`: the kind of the exit |
| `day`                 | `date`       | Not null: the day the exit happened                          |
| `quantity`            | `double precision` | Not null, positive and whole: the units it took         |
| `unit`                | `text`       | Not null, `unidad`: the exit counts the units it took         |
| `cost`                | `bigint`     | Not null, never negative: the cost the exit releases, in whole pesos |
| `tax_treatment`       | `text`       | Not null, `pending_the_professional`: the treatment the exit waits for |
| `tax_question`        | `text`       | Not null, never empty: the question of the taxes that waits   |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The exit moves no money IN.** There is no column of money that came in:
nobody paid for the bread the owner consumed or for the bread the bakery
gave away. What the exit releases is the cost of the units that left the
book, and the cost is derived — the money of the production they came out
of, divided by the units that production made. The check of the table holds the rule of the
domain as a defense in depth (`exits_cost_not_negative_check`).

**The tax treatment is never decided here.** The column holds the treatment
the exit waits for, and the check of the table refuses any other value
(`exits_tax_treatment_check`): the question of the taxes waits for the
professional of the taxes of the owner, and the table keeps it
(`exits_tax_question_not_empty_check`).

**The exit counts the units it took.** The quantity is positive and whole
(`exits_quantity_positive_check`, `exits_quantity_whole_check`), and the
unit is the unit of a piece (`exits_unit_check`): the book never converts
a weight into a number of units, and it never splits a unit.

**The exit is written with the word of the owner**, and the check of the
table holds the rule of the domain as a defense in depth
(`exits_confirmed_with_not_empty_check`).

### `recipe_shelf_life_corrections`

The correction of the shelf life of a recipe: the days what it makes keeps,
the way the owner corrected it.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the recipe                   |
| `recipe_id`           | `bigint`     | Not null, the recipe it corrects                              |
| `shelf_life_days`     | `integer`    | Not null, positive: the days what it makes keeps             |
| `day`                 | `date`       | Not null: the day the correction was declared                 |
| `utterance`           | `text`       | Not null, default `''`: the sentence of the owner, verbatim  |
| `confirmed_with`      | `text`       | Not null, never empty: the word she used to confirm it        |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The correction is kept.** Every correction is a row of its own, and the
last one the owner declared is the shelf life that counts: the recipe keeps
the shelf life it declared, and the corrections are the history that says
which one is the truth today.

**The correction is written with the word of the owner**, and the check of
the table holds the rule of the domain as a defense in depth
(`recipe_shelf_life_corrections_confirmed_with_not_empty_check`), and the
number of days is positive
(`recipe_shelf_life_corrections_days_positive_check`).

### `parties`

The parties of the business: the person, the store and the farm that buy
from the bakery, or that the bakery buys from.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that knows the party and owns it       |
| `kind`                | `text`       | Not null: `person`, `store` or `farm`                        |
| `name`                | `text`       | Not null, never empty, NOT unique inside the business         |
| `roles`               | `jsonb`      | Not null, default `[]`, an array of `client` and `provider`   |
| `phone`               | `text`       | Null when it is not set, digits only                           |
| `email`               | `text`       | Null when it is not set, the shape of an address               |
| `address`             | `text`       | Null when it is not set                                       |
| `legal_human_name`    | `text`       | Null when the party has none: the human that represents it     |
| `legal_figure`        | `text`       | Null when the human has no legal figure                       |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |
| `tax_id`              | `text`       | Null when the party carries no tax identity                    |

**The tax identity of the party is an OPTIONAL fact of it.** It is the NIT
of a person, of a store or of a farm, and the book never asks for it and
never assumes one. The book does not know the format of a tax identity
either: the column holds it the way the owner said it, and the check
`parties_tax_id_not_empty_check` holds that it is never an empty string.

**The legal human that represents a store or a farm lives in the party.**
It is a name, and the legal figure it has when it has one. It is never a
second person: it is not a row of the users, it is not a party of its own,
and it never becomes a row of the people of PACA. The check
`parties_legal_human_check` holds the rule of the kind as a defense in
depth: a person carries no legal human, a store carries the one that
represents it, and a farm carries it only when it has one. The check
`parties_legal_figure_needs_human_check`, added by the migration
`00025_parties_checks.sql`, holds the other half of the rule: a legal
figure without the name of the human it belongs to is a figure of nobody,
and the table refuses it.

**The party holds at least one role.** The roles travel as JSONB, like the
aliases of an ingredient, and the checks `parties_roles_array_check` and
`parties_roles_check` hold that they are a list and that the party holds the
client, the provider, or both of them. The check
`parties_roles_check`, tightened by the migration `00025_parties_checks.sql`,
holds that EVERY role of the array is one of the two: an array that holds
the client and another role is refused.

**The name of a party is NOT unique inside the business.** There is no index
of the name on purpose: the book never refuses two parties that share a
name, and the reading of the parties holds all of them.

**The optional columns are NULL when the value is not set**, as in the
table of the people, and the checks of the table hold the rule of the
domain: the phone holds digits only, the email holds the shape of an
address, and no optional column holds an empty string.

## The repository of the people

The adapter writes the people through the port of the domain
(`user.Repository`):

```go
Create(ctx, id, registration) (user.User, error)
UpdateDetails(ctx, id, details) (user.User, error)
ChangeStatus(ctx, id, status, observation) (user.User, error)
```

There is no delete and no list: a person that must go is `blocked`, and
nothing consumes a list yet. There is no read either: the port holds the
methods the use cases call, and none of them asks for a person by its
document. The adapter reads the row it needs by itself, when the move of a
state needs it.

**The writes are direct.** The adapter does not read a row to find out
whether it may write it: the insert tells the truth with its primary key,
and the update tells the truth with the row it touched. A document that is
already registered is `ErrAlreadyExists`, an email that belongs to somebody
else is `ErrEmailTaken`, and a document that does not exist is
`ErrNotFound`.

**A field that is empty is a field that does not change.** The details of a
person carry the values that are set; the statement writes them with
`COALESCE`, and the ones that are absent keep the value they had.

**The change of the status is the only place the adapter reads before it
writes.** The map of the states lives in the domain, and the map needs to
know the state the user is in. The statement also requires that state, so a
user that moved at the same time is reported instead of written over.

### `ledger_entries`

The movements of the book of a business.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the book                     |
| `debit_account_id`    | `bigint`     | Not null, the account where the money lands                   |
| `credit_account_id`   | `bigint`     | Not null, the account the money comes from                    |
| `amount`              | `bigint`     | Not null, positive                                            |
| `occurred_on`         | `date`       | Not null, the day of the movement                             |
| `description`         | `text`       | Not null, never empty                                         |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The amount is an integer and never a float.** The arithmetic of money does
not forgive a rounding error, and the column counts the pesos of the book.

**The two sides of a movement are not the same account,** and the
description is not empty: the CHECKs of the table hold the rules of the
domain as a defense in depth.

**The index of the book is the one the person reads it by:** the business,
the day and the identity of the movement.

## The repositories of the book

The adapter writes the book of a person through the ports of the domain
(`business.Repository` and `account.Repository`):

```go
Create(ctx, owner, business) (business.Business, error)
Find(ctx, id) (business.Business, error)
FindPersonalByOwner(ctx, owner) (business.Business, error)

Create(ctx, account) (account.Account, error)
CreateMany(ctx, accounts) ([]account.Account, error)
Find(ctx, id) (account.Account, error)
List(ctx, businessID) ([]account.Account, error)
```

The writes are direct, like the writes of the people: the identity of the
business tells the truth with its primary key, the partial unique index
tells the truth about the second personal business, and the unique index of
the name tells the truth about the account that is taken.

**The set of accounts is written in one transaction.** A book that starts
with half of its accounts is a book that lies about what the person has.

```go
Record(ctx, entry) (ledger.Entry, error)
List(ctx, businessID, from, to) ([]ledger.Entry, error)
Balances(ctx, businessID, asOf) ([]ledger.Balance, error)
Summary(ctx, businessID, from, to) (ledger.Summary, error)
```

**The balance of the accounts is a sum, and the database does it.** The money
of a person is read from the rows of the book, and the rows are the ones the
library wrote. The account that never moved is in the answer with a balance
of zero, because an account that is missing reads like an account that does
not exist.

**The money of a period is a sum of two sides, and the database does it too.**
The income is the money that landed in the accounts of an income, and the
expenses are the money that landed in the accounts of an expense. The two
sides of a movement are read apart and never joined into one row, so a
movement between two accounts of the same kind is counted once and not
twice. A period without movements reads as an empty one: zero income, zero
expenses.

## The repositories of the utterance

The adapter writes the utterance of the owner through the ports of the domain
(`purchase.Repository`, `purchase.Recorder` and `ingredient.Repository`):

```go
Save(ctx, draft) (purchase.Draft, error)
Find(ctx, id) (purchase.Draft, error)
Confirm(ctx, id, revision) (purchase.Draft, error)

Record(ctx, purchase) (purchase.Purchase, error)

List(ctx, businessID) ([]ingredient.Ingredient, error)
Create(ctx, ingredient) (ingredient.Ingredient, error)
Update(ctx, ingredient) (ingredient.Ingredient, error)
```

**The lines of a draft travel as JSONB.** The adapter encodes them and
decodes them, and the domain reads them back through its own constructors:
the row never decides what a line is.

**The confirmation of a draft is guarded by the revision the owner saw.**
The row is only confirmed when it still carries that revision, and the
storage refuses the confirmation of a draft that changed under her eyes.

**The completion of an ingredient is one update.** What the owner declared
about the thing the business buys — the unit, the weight of a package, the
allergens, the certification with its evidence, the aliases and the
conversion — is written in one statement, and the row keeps its identity
and its name: the completion of an ingredient never makes it another one.

**The confirmation writes the purchase, its lines and the catalog in one
transaction.** The unit of work opens it, and every repository called
inside it joins the transaction it finds: the purchase, the lines and the
new names of the catalog are written whole or not at all.

**The name of an ingredient is unique inside its business.** The index of
the table tells the truth about the name that is taken, and the adapter
says it with `ingredient.ErrDuplicateName`.

**The layers of the stock are written with the purchase they come from.**
The adapter writes them through the port of the domain
(`purchase.LayerRepository`):

```go
Record(ctx, layers) ([]purchase.Layer, error)
List(ctx, businessID) ([]purchase.Layer, error)
```

The layers of a business are read in the order they leave the stock: the
oldest layer is the first. The row never decides what a layer is: the
domain reads it back through `purchase.Layer.Checked`, and a row that
another writer broke is refused. The layer is asked before it is written,
too: a layer that claims another business than its purchase never reaches
the table.

**The money of the layer is the money of the line of its purchase.** The
line is read with the layer, and the domain asks whether the layer is the
layer of that line (`purchase.Layer.CheckedAgainstTheLine`): a row broken
with another money, or with another quantity, is refused with
`purchase.ErrLayerNotTied`, and it is never read as a layer of the stock.

## The repository of the recipes

The adapter writes the recipe of the bakery through the port of the domain
(`recipe.Repository`):

```go
Record(ctx, recipe) (recipe.Recipe, error)
```

And it reads the recipes back through the other port of the domain
(`recipe.Reader`):

```go
Find(ctx, recipeID) (recipe.Recipe, error)
List(ctx, businessID) ([]recipe.Recipe, error)
```

**The recipe, its ingredients and its steps are written in one
transaction.** The repository opens it when the caller did not, and it
joins the transaction it finds when the caller did: a recipe whose
ingredient is not in the catalog leaves nothing behind, because a guide
without its ingredients teaches nothing.

**The recipes are read back with their rows.** The recipe is read, and its
ingredients and its steps are read with it: a recipe without them is not a
recipe, and the model is built by the domain, through its own
constructors.

**The row never decides what a recipe is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The ingredient that is not in the catalog is `recipe.ErrNotFound`.** The
foreign key of the table tells the truth about it, and so does the business
that is not there.

## The repository of the reference prices

The adapter writes the reference prices through the port of the domain
(`purchase.ReferenceRepository`):

```go
Record(ctx, reference) (purchase.Reference, error)
List(ctx, businessID, ingredientID) ([]purchase.Reference, error)
```

**The list is read for one ingredient or for every one of them.** The two
queries are the two answers, and the adapter chooses the one the caller
asked for: the zero identity answers every reference of the business.

**The row never decides what a reference is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `purchase.ErrInvalidValue`.

**The ingredient that is not in the catalog is `purchase.ErrNotFound`.** The
foreign key of the table tells the truth about it, and so does the business
that is not there.

## The repository of the labour

The adapter writes the labour of the owner through the port of the domain
(`recipe.LaborRepository`):

```go
Record(ctx, labor) (recipe.Labor, error)
```

And it reads the labour back through the other port of the domain
(`recipe.LaborReader`):

```go
List(ctx, businessID) ([]recipe.Labor, error)
```

**The list is read in the order of the day the labour was worked.** The
value of the hour the business declared is the last one it declared, and
the order of the list is what says which is the last one.

**The row never decides what a labour is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The business that is not there is `recipe.ErrNotFound`.** The foreign key
of the table tells the truth about it.

## The repository of the overhead

The adapter writes the overhead of the house through the port of the domain
(`recipe.OverheadRepository`):

```go
Record(ctx, overhead) (recipe.Overhead, error)
```

And it reads the overhead back through the other port of the domain
(`recipe.OverheadReader`):

```go
List(ctx, businessID) ([]recipe.Overhead, error)
```

**The list is read in the order of the day the overhead was declared.** The
rate of the house the business declared is the last one it declared, and
the order of the list is what says which is the last one.

**The row never decides what an overhead is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The business that is not there is `recipe.ErrNotFound`.** The foreign key
of the table tells the truth about it.

## The repository of the productions

The adapter writes the production of the bakery through the port of the
domain (`recipe.ProductionRepository`):

```go
Record(ctx, production) (recipe.Production, error)
```

And it writes the quantity that is left of the layers the production
consumed through the other port of the domain
(`purchase.LayerRepository`):

```go
WriteRemaining(ctx, remaining) error
```

**The production, the layers it consumed and the units it made are written
in one transaction.** The repository opens it when the caller did not, and
it joins the transaction it finds when the caller did: a production whose
layers did not leave the stock leaves nothing behind, because a production
that consumed no stock is not a production.

**The quantity that is left of a layer is written with the unit of the
layer.** The row is written only when the layer still holds what the
production read: the quantity and the unit travel together, and a layer
that changed under the production is refused.

**The row never decides what a production is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The business, the recipe or the layer that is not there is
`recipe.ErrNotFound`.** The foreign key of the table tells the truth about
it.

## The repository of the expiry

The adapter writes the correction of the shelf life through the port of the
domain (`recipe.ShelfLifeRepository`):

```go
Record(ctx, correction) (recipe.ShelfLifeCorrection, error)
```

And it reads the corrections back through the other port of the domain
(`recipe.ShelfLifeReader`):

```go
List(ctx, businessID) ([]recipe.ShelfLifeCorrection, error)
```

**The corrections are read in the order of the day they were declared.** The
shelf life that counts is the last one the owner declared, and the order of
the list is what says which is the last one.

The adapter reads the units of the productions through the port of the
domain (`recipe.UnitRepository`):

```go
List(ctx, businessID) ([]recipe.ExpiringUnit, error)
WriteExpiry(ctx, units) error
```

**The units are read with the production and the recipe they belong to.**
The expiry is read for one business, and the day of the expiry of a unit is
a day of that business: the query joins the production and the recipe, and
the units of another business are never read as units of it.

**The units that left the book are not offered.** A unit that was wasted,
sold, consumed or given away is not in the answer of the reader of the
expiry: the query reads the table of the exits, and a unit that left the
book is never offered again.
The correction of the shelf life moves the units that are still in the
book, and a unit that left keeps the day it expired on.

**The day of the expiry is written only when the unit still expires on the
day the correction read.** A unit that changed under the correction is
refused, and the correction leaves the book as it was.

**The row never decides what a correction or a unit is.** The model is
built by the domain, through its own constructors: a row that breaks a rule
of it is refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The business or the recipe that is not there is `recipe.ErrNotFound`.**
The foreign key of the table tells the truth about it.

## The repository of the waste

The adapter writes the waste of the bakery through the port of the domain
(`recipe.WasteRepository`):

```go
Record(ctx, waste) (recipe.Waste, error)
```

And it reads the productions back through the port of the domain
(`recipe.ProductionReader`):

```go
List(ctx, businessID) ([]recipe.Production, error)
```

**The waste and the units that left the book are written in one
transaction.** The repository opens it when the caller did not, and it
joins the transaction it finds when the caller did: a waste whose units did
not leave the book leaves nothing behind, because a waste of units the book
does not hold is not a waste.

**A unit leaves the book only once.** The primary key over the unit of the
table of the exits tells the truth about it, and the adapter says it with
`recipe.ErrInvalidValue`.

**The row never decides what a waste is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The business, the recipe or the unit that is not there is
`recipe.ErrNotFound`.** The foreign key of the table tells the truth about
it.

## The repository of the sale

The adapter writes the sale of the bakery through the port of the domain
(`recipe.SaleRepository`):

```go
Record(ctx, sale) (recipe.Sale, error)
```

**The sale and the units that left the book are written in one
transaction.** The repository opens it when the caller did not, and it
joins the transaction it finds when the caller did: a sale whose units did
not leave the book leaves nothing behind, because a sale of units the book
does not hold is not a sale.

**A unit leaves the book only once, whichever exit takes it.** The primary
key over the unit of the table of the exits tells the truth about it, and
the adapter says it with `recipe.ErrInvalidValue`: a unit that left as a
waste is never sold, and a unit that left as a sale is never wasted.

**The row never decides what a sale is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `recipe.ErrInvalidValue`.

**The business, the recipe or the unit that is not there is
`recipe.ErrNotFound`.** The foreign key of the table tells the truth about
it.

## The repository of the parties

The adapter writes the parties through the port of the domain
(`party.Repository`):

```go
Create(ctx, party) (party.Party, error)
List(ctx, businessID) ([]party.Party, error)
```

**The roles of the party travel as JSONB.** The adapter encodes them, and
the domain reads them back through its own constructors: the row never
decides what a role is.

**The legal human that represents a store or a farm travels in the party.**
It is a fact of the party and never a second person: it is not a row of the
users, and the adapter never writes it anywhere else.

**The row never decides what a party is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `party.ErrInvalidValue`.

**The business of the party that is not there is `party.ErrNotFound`.** The
foreign key of the table tells the truth about it.

### `invoices`

The invoices: the fact that comes with the purchase.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that bought and that holds the invoice |
| `provider_id`         | `bigint`     | Not null, the party that issued the invoice                   |
| `purchase_id`         | `bigint`     | Not null, the purchase the invoice came from                  |
| `number`              | `text`       | Not null, never empty, unique inside the business             |
| `day`                 | `date`       | Not null, the day the invoice was issued                      |
| `base`                | `bigint`     | Not null, never negative: the money the tax applies to        |
| `rate`                | `bigint`     | Not null, the rate in hundredths of a percent                 |
| `tax`                 | `bigint`     | Not null, never negative: the tax of the base at the rate     |
| `total`               | `bigint`     | Not null, never negative: the base plus the tax               |
| `reference`           | `text`       | Null when the invoice carries none                            |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The tax of an invoice is NEVER ASSUMED: it comes from the invoice.** The
invoice states the rate it was issued with and the base that rate applies
to, and the tax is derived from the two of them. The rate travels in
hundredths of a percent, because the rate of the tax authority of the
owner may not be a whole percent, and a rate that is rounded away is a rate
nobody charged.

**An invoice that does not add up is never read as an invoice.** The checks
`invoices_tax_check` and `invoices_total_check` hold the rule of the
domain as a defense in depth: the tax is the tax of the base at the rate,
rounded to the nearest whole peso with a half peso going up, and the total
is the base plus the tax.

**The money of an invoice enters the book in whole pesos, and the table
never rounds it.** The rate travels in hundredths of a percent, and a rate
with more than two decimals is refused by the domain. The rounding of the
CENTAVOS of an invoice is NOT decided here: it is an OPEN QUESTION for the
owner and for the professional of the taxes.

**The number of an invoice is unique inside the business.** The index
`invoices_one_number_per_business` refuses a second invoice with the same
number inside the same business: the number is what identifies the invoice.

**A purchase carries ONE invoice.** The index `invoices_one_per_purchase`
refuses a second invoice of the same purchase, whatever number it carries:
the invoice that came with the purchase is the one the purchase points at.

**The invoice is attached only to the purchase it names.** The update that
points the purchase at the invoice carries the purchase the invoice names,
and the row that is not the one the invoice names is not touched.

**The electronic invoice is NEVER assumed.** There is no integration with
the tax authority and no column that pretends to be one: the reference is a
plain fact of the document the owner shows, and it may be NULL.

### The repository of the invoices

The adapter writes the invoices through the port of the domain
(`purchase.InvoiceRepository`):

```go
Record(ctx, invoice) (purchase.Invoice, error)
Attach(ctx, invoice, purchase) error
List(ctx, businessID) ([]purchase.Invoice, error)
```

**The row never decides what an invoice is.** The model is built by the
domain, through its own constructors: the rate is read back as a rate and
an invoice that does not add up is refused, and the adapter says it with
`purchase.ErrInvalidValue`.

**The number the business already holds is
`purchase.ErrDuplicateInvoiceNumber`.** The index of the table tells the
truth about it, and the domain refuses it before the row is written.

**The purchase that already carries an invoice is
`purchase.ErrPurchaseAlreadyHasInvoice`.** The index of the table tells
the truth about it, and the domain refuses it before the row is written.

**The purchase that is not the one the invoice names is
`purchase.ErrInvoiceOfAnotherPurchase`.** The update of the table tells
the truth about it.

**The business, the provider or the purchase of the invoice that is not
there is `purchase.ErrNotFound`.** The foreign key of the table tells the
truth about it.

### `products`

The products of the bakery: the thing it sells.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the product and its price list |
| `recipe_id`           | `bigint`     | Not null, the recipe the product comes from                   |
| `name`                | `text`       | Not null, never empty, unique inside the business             |
| `tax_treatment`       | `text`       | Null when the owner has not decided it                        |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The product of the bakery IS the thing it sells.** The column carries the
identity of the recipe it comes from and never a second copy of it: the
bakery sells what it bakes.

**The tax treatment is an OPTIONAL fact of the product.** The column is
NULL when the owner has not decided it, and the book never derives it from
the price and never assumes a rate. The check of the table holds the rule
of the domain as a defense in depth
(`products_tax_treatment_not_empty_check`).

**The name of a product is taken inside the business**, and the index
`products_one_name_per_business` refuses a second product with that name.
The name is compared the way a person reads it — trimmed and without case —
so the index is over the LOWER CASE OF THE TRIMMED NAME:
`lower(btrim(name))`.

**One recipe backs one product inside the business**, and the index
`products_one_recipe_per_business` refuses a second one.

**The product is backed by a recipe of ITS OWN business.** The composite
foreign key `products_recipe_of_its_business_fkey` refuses the recipe of
another business: the two independent foreign keys cannot hold the rule,
and the composite one carries the business of the pair. The unique index
`recipes_business_and_id` holds the columns it points at.

### `prices`

The price list: the money of the product in a channel, with the day it was
set.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the price list               |
| `product_id`          | `bigint`     | Not null, the product the price is of                         |
| `channel`             | `text`       | Not null: `retail` or `wholesale`                             |
| `amount`              | `bigint`     | Not null, positive: the money in whole pesos                  |
| `basis`               | `text`       | Not null: the way the owner charges                            |
| `day`                 | `date`       | Not null: the day the price was set                           |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The channel is already in the price from the first row.** Every price
carries the channel it was set for, so that the discount of the wholesale
never has to rebuild the history of the prices. The check of the table
holds the two channels of the book as a defense in depth
(`prices_channel_check`).

**The money of a price is a positive number of whole pesos**, and the check
`prices_amount_positive_check` holds the rule of the domain as a defense in
depth.

**The price carries THE WAY THE OWNER CHARGES.** The money of a price is
only a price when the book knows whether it is per unit, per gram or per
kilogram, and the check `prices_basis_check` holds the three ways of
charging of the book as a defense in depth.

**The price is of a product of ITS OWN business.** The composite foreign
key `prices_product_of_its_business_fkey` refuses the product of another
business, and the unique index `products_business_and_id` holds the columns
it points at.

**The price list is a HISTORY.** The rows are kept and nothing is rewritten
when a price changes: the price that holds today for a product and a
channel is the last one set on or before the day that is asked for, and two
prices of the same day are both kept. The index
`prices_of_the_business` reads the list of a business, a product and a
channel.

### The repository of the price list

The adapter writes the products through the port of the domain
(`product.Repository`):

```go
Create(ctx, product) (product.Product, error)
List(ctx, businessID) ([]product.Product, error)
State(ctx, product) (product.Product, error)
```

And it writes the prices through the other port of the domain
(`product.PriceRepository`):

```go
Set(ctx, price) (product.Price, error)
List(ctx, businessID) ([]product.Price, error)
```

**The product and the price are written in one transaction.** The use case
opens the unit of work, and every repository called inside it joins the
transaction it finds: a price of a product that is not there leaves nothing
behind.

**The row never decides what a product or a price is.** The model is built
by the domain, through its own constructors: a row that breaks a rule of it
is refused, and the adapter says it with `product.ErrInvalidValue`.

**The name that is taken is `product.ErrDuplicateName`, and the recipe
that backs another product is `product.ErrDuplicateRecipe`.** The indexes
of the table tell the truth about them.

**The recipe of another business and the product of another business are
`product.ErrOfAnotherBusiness`.** The composite foreign keys of the table
tell the truth about them.

**The business or the product that is not there is `product.ErrNotFound`.**
The foreign key of the table tells the truth about it.

### `orders`

The orders of the bakery: what it owes to a client, and the day it is due.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the order and owes it       |
| `party_id`            | `bigint`     | Not null, the client that ordered                             |
| `party_name`          | `text`       | Not null, never empty: the name the owner reads               |
| `due_on`              | `date`       | Not null: the day the order is due on                         |
| `state`               | `text`       | Not null, default `open`: `open` or `delivered`               |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The order is written OPEN.** An order that was not delivered stays open,
and the delivery of an order is written with the route of the round: the
route is what the agent computes when it comes back from the round, and the
increment that writes the route writes the delivery with it. The check
`orders_state_check` holds the two states of the book as a defense in
depth.

**The party of the order is a party of ITS OWN business.** The composite
foreign key `orders_party_of_its_business_fkey` refuses the party of
another business, and the unique index `parties_business_and_id` holds the
columns it points at.

**The name of the party is the name the owner reads**, and the check
`orders_party_name_not_empty_check` holds that it is never empty.

### `order_lines`

The lines of an order: the products with the quantities of them and the
way the owner charges them.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `order_id`            | `bigint`     | Not null, the order the line belongs to                       |
| `business_id`         | `bigint`     | Not null, the business that owns the order                    |
| `product_id`          | `bigint`     | Not null, the product of the price list the line is of        |
| `position`            | `integer`    | Not null, positive: the order the owner said the line in      |
| `product_name`        | `text`       | Not null, never empty: the name the owner reads               |
| `quantity`            | `double precision` | Not null, positive: the amount of pieces or of weight   |
| `unit`                | `text`       | Not null: `unit`, `gram` or `kilogram`                       |

**The unit of a line is THE WAY THE OWNER CHARGES.** The book knows three
of them — the piece (`unit`), the gram (`gram`) and the kilogram
(`kilogram`) — and the check `order_lines_unit_check` holds them as a
defense in depth: the table refuses the row that carries another one. The
way the owner said it is read by the domain, and the table holds the way
the book knows it, the way the basis of a price is held.

**The quantity of a line travels with the way she charges.** When she
charges by the PIECE the quantity is a whole number of pieces — the book
never splits a piece — and the check `order_lines_piece_quantity_check`
holds that rule as a defense in depth. When she charges by the weight the
quantity is any positive amount of weight: the book never converts a
weight into a number of pieces, and the column of the quantity is a double
precision for that reason. The check `order_lines_quantity_check` holds
that the quantity is positive.

**The migration of the orders is not edited.** The rules of the units of
the book arrived with the migration `00031_orders_units_of_the_book.sql`,
and the migration of the orders (`00030_orders.sql`) is already applied: a
migration that is applied is a history that is not rewritten.

**The order of the line and the product of the line are of the business
that owns the order.** The composite foreign keys
`order_lines_order_of_its_business_fkey` and
`order_lines_product_of_its_business_fkey` refuse the line of a foreign
order and the line of a foreign product, and the unique indexes
`orders_business_and_id` and `products_business_and_id` hold the columns
they point at.

**The position of a line is the order the owner said it in**, and the index
`order_lines_one_position_per_order` refuses a second line in the same
place.

### `routes`

The route of the round of a day: the stops in the order the agent will
visit the clients in, and what each stop costs.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the route                   |
| `day`                 | `date`       | Not null: the day of the round the route is of               |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The route of a business and a day is written ONCE.** The unique index
`routes_one_route_per_business_and_day` refuses a second route of the same
day, and the domain refuses it too (`order.ErrRouteAlreadyWritten`): the
round was already planned.

**The total of the route is NOT a column.** The total is derived from the
stops by the domain (`order.Route.Total`), and the storage never holds it.

### `route_stops`

The stops of a route: the orders the business still owes, in the order the
agent will visit the clients in, and what each visit costs.

The table can prove that a stop is an order of the business that owns the
route, and it cannot prove that the order is due on the day of the route
or that it is still open: those two rules are held by the domain
(`order.TheRoute`), and they say so here. A writer that bypassed the
domain could stop on an order of another day, and the delivery of the
orders — which moves only the open ones — is what refuses the reopened
order.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `route_id`            | `bigint`     | Not null, the route the stop belongs to                       |
| `business_id`         | `bigint`     | Not null, the business that owns the route                    |
| `order_id`            | `bigint`     | Not null, the order the stop visits                           |
| `position`            | `integer`    | Not null, positive: the order the agent visits it in          |
| `cost`                | `bigint`     | Not null: the money of the transport to that client, in pesos  |

**The cost of a stop is NEVER NEGATIVE.** The money of the book counts
whole pesos, and a stop the owner walks to costs nothing: the check
`route_stops_cost_check` holds that the cost is never negative, and the
domain refuses it too (`order.ErrInvalidCost`).

**The stop of an order of ANOTHER business is refused.** The composite
foreign key `route_stops_order_of_its_business_fkey` refuses it, and the
domain refuses it too (`order.ErrStopNotAnOrder`).

**The stop of a route of another business is refused.** The composite
foreign key `route_stops_route_of_its_business_fkey` refuses it, and the
unique indexes `routes_business_and_id` and `orders_business_and_id` hold
the columns they point at.

**The position of a stop is the order the agent visits it in**, and the
index `route_stops_one_position_per_route` refuses a second stop in the
same place.

**The route visits every order ONCE.** The unique index
`route_stops_one_stop_per_order` refuses the second stop of the same order
in the same route, because the total of the route is derived from the stops
and would count the visit twice. The domain refuses it first
(`order.ErrDuplicateStop`), and the index arrived with the migration
`00033_route_stops_one_order_each.sql`.

### The repository of the orders

The adapter writes the orders through the port of the domain
(`order.Repository`):

```go
Create(ctx, order) (order.Order, error)
List(ctx, businessID) ([]order.Order, error)
Deliver(ctx, orders) error
```

**The DELIVERY of the orders is written with the ROUTE of the round.** The
delivery of an order moves its state to the delivered one, and the state
that is written is the one the domain decided: the adapter never decides
the state of an order by itself. The deliveries are written in one
transaction, so an order that was delivered and an order that was not is a
round that was made and never happened.

**The delivery moves an order that is OPEN, and only one.** The query
`DeliverOrder` writes the delivered state over the open one and of no other
state, so an order that was delivered is never reopened: a caller that
bypasses the domain is refused (`order.ErrAlreadyDelivered`), and the row
it tried to reopen is left as it was. The order that is not there is
`order.ErrNotFound`.

### The repository of the routes

The adapter writes the routes through the port of the domain
(`order.RouteRepository`):

```go
Create(ctx, route) (order.Route, error)
List(ctx, businessID) ([]order.Route, error)
```

**The route and its stops are written in one transaction.** The set joins
the transaction of the caller when the caller opened one, and opens its own
when it did not: a route whose stop is refused leaves nothing behind.

**The row never decides what a route is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `order.ErrInvalidValue`.

**The second route of a day is `order.ErrRouteAlreadyWritten`, and the
stop of an order of another business is `order.ErrStopNotAnOrder`.** The
unique index and the composite foreign key of the tables tell the truth
about them.

**The order and its lines are written in one transaction.** The set joins
the transaction of the caller when the caller opened one, and opens its own
when it did not: an order whose line is refused leaves nothing behind.

**The row never decides what an order is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `order.ErrInvalidValue`.

**The party of another business is `order.ErrPartyOfAnotherBusiness`, and
the product of another business is `order.ErrProductOfAnotherBusiness`.**
The composite foreign keys of the tables tell the truth about them.

**The business, the party or the product that is not there is
`order.ErrNotFound`.** The foreign key of the tables tells the truth about
it.

### `commitments`

The promises of the bakery: what is due, in the words of the owner, and the
day it is due — and the day it was met, when it was met.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owes the promise                 |
| `what`                | `text`       | Not null, never empty: what is due, in the words of the owner |
| `due_on`              | `date`       | Not null: the day the promise is due on                       |
| `met_on`              | `date`       | NULL while the promise is open, the day it was met after      |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The table carries NO state column.** The day of meeting IS the fact: a row
with no `met_on` is an open promise and a row that carries it is a promise
that was kept. A state column would be a second truth for the same fact, and
the two truths drift.

**The table carries NO money and NO party.** The promise is not an order:
the order owns the client delivery — the party, the products, the money and
the round — and the promise never copies it. The "what" names what is due in
the words of the owner, and it may name the party in her own words.

**A promise is NEVER DELETED.** The promise that was met stays in the book
with the day it was kept, and there is no query of this table that deletes a
row.

**The promise says what is due.** The check `commitments_what_check` refuses
a "what" that is empty or that holds spaces only, as a defense in depth: the
domain refuses it first (`commitment.ErrMissingWhat`).

**The promise belongs to a business that is there.** The foreign key
`commitments_business_id_fkey` refuses the promise of a business that does
not exist, and the adapter says it with
`commitment.ErrBusinessNotFound`.

**What the table CANNOT prove: that a promise was met once.** There is no
constraint of the schema that refuses a second day of meeting, because the
day of meeting is the state of the row and a constraint cannot compare a row
with the row it replaces. The rule is held by the write: the meeting is
written with `WHERE ... AND met_on IS NULL`, so a second meeting changes no
row and the adapter answers the error of the domain
(`commitment.ErrAlreadyMet`). A writer that bypassed the library could
overwrite the day of meeting with a raw `UPDATE`, and the domain refuses that
row when it is read back.

**The meeting carries its day.** The promise that was kept stays in the book
WITH the day it was kept, so the write of the meeting also asks for the day:
`WHERE ... AND $1::date IS NOT NULL`. A meeting with no day matches no row,
and the adapter answers the rule of the domain
(`commitment.ErrMissingMetDay`). The domain refuses it first, and the table
holds it as its defense in depth: a caller that goes around the domain and
the use case is never told that a promise was kept when nothing was kept.

### The repository of the commitments

The adapter writes the promises through the port of the domain
(`commitment.Repository`):

```go
Create(ctx, commitment) (commitment.Commitment, error)
Meet(ctx, commitment) (commitment.Commitment, error)
List(ctx, businessID) ([]commitment.Commitment, error)
```

**The promise is written OPEN.** The insert names the business, what is due
and the day it is due: the day of meeting is not a column of the insert, and
the book never writes one at the creation.

**The MEETING is written over an open promise, and of no other one.** The
query `MeetCommitment` writes the day of meeting where no day of meeting is
held, so a promise that was met already is never met again and the day the
book holds is never hidden. The same write asks for the day of the meeting
(`AND $1::date IS NOT NULL`), so a meeting that carries no day matches no row
either.

The adapter reads the row back only when the write matched no promise, and
the day the row carries answers the three outcomes the port states: the
promise is not there (`commitment.ErrNotFound`), the promise was met already
(`commitment.ErrAlreadyMet`, with the day the book holds), or the promise is
open and the meeting carried no day (`commitment.ErrMissingMetDay`). The
adapter answers those three errors of the domain and invents no fourth one.

**The row never decides what a promise is.** The model is built by the
domain, through its own constructors: a row that breaks a rule of it is
refused, and the adapter says it with `commitment.ErrInvalidValue`.

**The reading holds the promises of the business and of no other one**, in
the order of their identity: the answer is the same every time it is read,
and the promises that were kept are in it.

### `learnings`

The learning log of the bakery: the memory of the errors and the wins of the
owner. An entry says what happened, in her words, and what was learned from
it, about a day and about the batch the business bakes.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the memory                   |
| `day`                 | `date`       | Not null: the day the entry is about                          |
| `recipe_id`           | `bigint`     | Not null, the batch the entry is about                        |
| `production_id`       | `bigint`     | NULL when the entry is about the batch alone, the run of it after |
| `happened`            | `text`       | Not null, never empty: what happened, in the words of the owner |
| `learned`             | `text`       | Not null, never empty: what was learned, in the words of the owner |
| `answers_id`          | `bigint`     | NULL when the entry answers no other one                      |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The table carries NO money.** The log is not a sale and it is not a
movement of the book: it never moves a peso, and the entry never carries an
amount, a price or a currency.

**The log is NEVER REWRITTEN.** There is no query of this table that updates
the words of an entry and no query that deletes a row: a correction is a NEW
entry that answers the old one, and the entry that was answered keeps its own
words for ever.

**The answer carries the business with it.** The composite foreign key
`learnings_answer_of_its_business_fkey`, over `(answers_id, business_id)`
against `(id, business_id)`, refuses the answer of an entry of another
business, and the adapter says it with
`learning.ErrEntryOfAnotherBusiness`. The key needs the unique index
`learnings_identity_and_business`, which this migration creates.

**An entry is answered ONCE.** The partial unique index
`learnings_answered_once`, over `answers_id` where it is not null, refuses the
second answer of the same entry, and the adapter says it with
`learning.ErrAlreadyAnswered`. The log is a chain and not a tree, and two
answers of the same entry would leave the owner with two truths.

**The batch and the production are of the business that owns the entry.** The
composite foreign keys `learnings_batch_of_its_business_fkey` and
`learnings_production_of_its_business_fkey` carry the business of the pair,
and the adapter says them with `learning.ErrRecipeOfAnotherBusiness` and
`learning.ErrProductionOfAnotherBusiness`. They need the unique indexes
`recipes_business_and_id` of `00027_products_of_their_own_business.sql` and
`productions_business_and_id`, which this migration creates.

**The production is a run of the NAMED batch.** The composite foreign key
`learnings_production_of_its_batch_fkey`, over `(production_id, recipe_id)`
against `(id, recipe_id)`, refuses the entry about a production that is not a
run of the batch it names, and the adapter says it with
`learning.ErrAnotherBatch`. It needs the unique index
`productions_identity_and_recipe`, which this migration creates.

**An entry says what happened and what was learned.** The checks
`learnings_happened_check` and `learnings_learned_check` refuse a "what
happened" and a "learned" that are empty or that hold spaces only, as a
defense in depth: the domain refuses them first
(`learning.ErrMissingHappened` and `learning.ErrMissingLearned`).

**The entry belongs to a business that is there.** The foreign key
`learnings_business_id_fkey` refuses the entry of a business that does not
exist, and the adapter says it with `learning.ErrBusinessNotFound`.

**What the table CANNOT prove: that the day of the entry is a day of the
book.** The column refuses what is not a date, and the domain is the one that
says whether a day is there at all (`learning.ErrMissingDay`). The table also
cannot tell WHICH entry a writer meant to answer: the composite foreign key
refuses the pair of another business, and an entry that is not there violates
the same key, so the two are the same refusal of the table — the domain is the
one that tells them apart (`Log.Answering`). And a writer that read the chain
before another one wrote its answer is refused by the partial unique index
when it writes, not when it reads: the adapter answers that refusal with
`learning.ErrAlreadyAnswered`.

### The repository of the learning log

The adapter writes and reads the log through the port of the domain
(`learning.Repository`):

```go
Create(ctx, entry) (learning.Entry, error)
List(ctx, businessID) ([]learning.Entry, error)
```

**The production and the answer are written only when the entry names them.**
The insert carries NULL in `production_id` when the entry is about the batch
alone, and in `answers_id` when the entry answers no other one: the columns
keep the rule of the absence.

**There is no update and no delete.** The port has no method that rewrites an
entry, and the table has no query that changes the words of one: the log of a
business is written once and read whole. A correction is a row of its own
that answers the row it corrects.

**The row never decides what an entry is.** The model is built by the domain,
through its own constructors: a row that breaks a rule of it is refused, and
the adapter says it with `learning.ErrInvalidValue`.

**The reading holds the entries of the business and of no other one**, in the
order they were written, each one with the name of the batch it names — read
from the table of the recipes, which is the owner of it — and the ones that
were answered are in it: the log never hides the past.

## The transaction of the storage

The unit of work is the port of a transaction of the storage
(`store.UnitOfWork`):

```go
Do(ctx, work) error
```

The work that the caller hands over runs inside one transaction: the rows
are written whole or not at all. The caller never sees the transaction and
never commits by itself.

**The transaction travels in the context.** The repositories of the library
are bound to the pool and not to a transaction: the unit of work opens the
transaction, and every repository called inside it joins the transaction it
finds. A repository called outside one writes by itself.

## The migrations

The schema has **one writer**: the `db-admin` service (`apps/db-admin/`). It
applies the pending migrations when it starts, and nothing else writes the
schema — not a service, not a command.

The other services never migrate on boot: a service that migrates on boot
races with the other replicas of itself. The writer is the exception, and
the lock of the database is what keeps it safe.

```sh
make stack-up          # the writer applies the schema when it starts
make stack-schema      # assert that the schema was written by its owner
```

The migrator takes the lock of the database while it applies a migration,
so two migrators that run at the same time take turns instead of writing
the same table twice.

The migration files are embedded in the binary: the file that creates a
table is the file the generator reads, so the schema and the row model
cannot drift apart.

## The generated code

The schema is the source of truth and the SQL is written by hand. `sqlc`
reads the migrations, reads the queries and writes the row model and the
queries themselves.

```sh
make -C packages/database generate          # write the generated code
make -C packages/database generate-check    # fail when it is not what the schema describes
```

The gate of the module regenerates the code and fails when it differs from
the committed code, so a change of the schema never reaches the repository
without the code that goes with it.

The generator needs one query to write the code of the connection: the
`Ping` of the health check. It touches no table, and it is not a query of
the domain. The queries of the people live in `queries/users.sql`, and they
are one statement per operation.

## The tracing of the statements

The library knows the **API** of the tracing and nothing else: it builds no
provider, exports no span and reads no environment variable. The
composition root of the service owns the provider and hands the tracer over
when it opens the pool:

```go
pool, err := postgres.Open(ctx, poolCfg, postgres.WithTracer(tracer))
```

A pool without a tracer traces nothing, and the behaviour of the library is
the same. The option is additive: no caller of the library breaks.

Every statement leaves a span of its own, named `pg.query`, child of the
span of the work that ran it:

```
open_business                    the call of the tool
└── pg.query                     the statement
```

The span carries `db.system`, `db.namespace` and `db.statement`. The
statement is the SQL of the generated code, and the generated code always
binds the values: the span carries the shape of the query and never a value
of it. The type of a failure is recorded when a statement fails, and never
its message: the messages of the database carry the values they refuse.

The seam is the one the generator owns (`DBTX`), so one decorator covers the
statements of the repositories that exist and the ones that will be written
tomorrow.

The migrator of the schema traces its work the same way: `schema.migrate`,
`schema.lock`, `schema.version` and `schema.pending` are the spans of the
work of the writer of the schema, and the tracer reaches it through the
same kind of option. The migrator never opens a **root** span: it enriches
a trace that is already open, so the health check of the service — the
runtime asking whether the service is alive — leaves nothing behind in the
store.

## The configuration of a service

The library reads no environment variable. A service maps its own environment
onto the configuration of the library, so the same library serves every
service with its own settings.

The services that name their database with the credentials of the
PostgreSQL stack build the connection string in one place, in
`packages/database/config`. The package is pure: it reads no variable, no
file and no clock, it takes the credentials and returns the string. Which
variable wins, which default applies and which value is invalid stay in
the service, because every service states them in its own words and in its
own tests.

| Variable                       | Default | What it is                                          |
|--------------------------------|---------|-----------------------------------------------------|
| `PACA_DATABASE_URL`            | *empty* | The connection string of the database.               |
| `PACA_DATABASE_MAX_CONNS`      | `4`     | The number of connections the pool may open.         |
| `PACA_DATABASE_CONNECT_TIMEOUT`| `5s`    | The bound of the attempt to open a connection.       |

The database is optional: a service that does not name one starts without
it. The pool is lazy, so a database that is down does not stop a service
from starting — the health check is what asks the database whether it
answers.

## Running it

```sh
make -C packages/database all               # the full quality gate
make -C packages/database test-integration  # the container tests (Docker)
```

## The quality gate

| Target            | What it does                                                     |
|-------------------|------------------------------------------------------------------|
| `make deps`       | Downloads and verifies the dependencies.                          |
| `make tidy-check` | Fails when `go.mod` is not tidy.                                 |
| `make fmt-check`  | Fails when a source is not formatted.                             |
| `make vet`        | Runs the analyzers of the toolchain.                              |
| `make lint`       | Runs `golangci-lint`, with `gosec` among the linters.             |
| `make build`      | Builds every package of the module.                               |
| `make test`       | Runs the test suite.                                              |
| `make test-integration` | Starts a PostgreSQL with the schema and asserts the rules of the tables against it. |
| `make cover`      | Measures the coverage and **fails below 80%**.                    |
| `make race`       | Runs the test suite with the race detector.                      |
| `make bench`      | Runs the benchmarks.                                              |
| `make perf`       | Checks the allocation budgets, then runs the benchmarks.          |
| `make vuln`       | Runs `govulncheck` over the dependencies.                         |
| `make generate-check` | Fails when the generated code is not what the schema describes. |

The tools are pinned in the repository (`tools/go.mod`), so every machine
and every container uses the same versions.

### The coverage, in two numbers

The gate of the module measures the packages whose rules can be asserted
without a database: the adapter. The parts that talk to the database — the
migrator and the file of the repository — are covered by the container
tests, which report their own coverage. The generated code is not measured
at all: it is written by the generator, never by hand.

## The tests

The unit tests live with the code they cover. The container tests live in
`test/integration` and need Docker:

```sh
make -C packages/database test-integration
```

They start a PostgreSQL, apply the migrations through the single writer, and
assert every rule of the tables against the real database: the primary key
that refuses a duplicate, the identifier that the database never invents,
the status that the database checks, the optional columns that hold NULL,
the timestamps that the system fills, the email and the phone that belong
to one person only, and the observation that belongs to the closed states.

They run in the workflow of the continuous integration, next to the gate of
the applications.

## The versions

| What             | Version  | Why                                                     |
|------------------|----------|---------------------------------------------------------|
| Go               | 1.27.4   | The latest security release of the 1.27 line.            |
