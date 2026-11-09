# The MCP server of the PACA monorepository.

The server speaks the [Model Context Protocol](https://modelcontextprotocol.io)
over the official Go SDK, answers a health endpoint, and exposes **three
tools**: the ones that take care of the people who use PACA.

The service is **private**. It publishes no port: the only things that reach it
are the other containers of the monorepository, through the `paca_internal`
network. The host and the internet do not.

The image of the server is built from the root of the monorepository, because
the server imports the shared libraries of the repository, and they are
modules of their own next to it.

---

## The architecture

The server is a hexagonal application. The rules live in the middle, the
protocol lives at the edge, and the composition root is the only place that
knows every layer at once.

```
cmd/paca-mcp-server/          the composition root: it wires the layers
internal/
  application/                 the use cases: the rules of the service
  ports/                       the contracts between the layers
  adapters/
    inbound/http/              the HTTP adapter: the router of the service
    inbound/mcpserver/         the MCP adapter: the official Go SDK
    outbound/serving/          the checker of the MCP transport
    outbound/database/         the checker of the database
  platform/config/             the configuration of this service: the only
                               infrastructure the service owns
  bootstrap/                   the wiring of every layer
test/integration/              the container tests
```

The models and the infrastructure live in the shared modules: the person, the book of
their money and the health in [packages/domain](../../packages/domain), the health itself
— the use case that reports it and the endpoint that answers it — in
[packages/health](../../packages/health), and the clock, the logs, the telemetry and the
server in [packages/platform](../../packages/platform). The service owns only the models
that belong to it alone, and the configuration that is its own.

The dependency rule holds: the domain knows nothing, the application knows
the domain and the ports, the adapters know the ports, and only the
composition root knows the adapters. Every layer is exercised through its
public interface, so a rule can be tested without a server.

## The database

The server imports the [database library](../../packages/database) and the
[domain](../../packages/domain) of the monorepository, and it reads and writes
the people through them: the pool, the driver and the queries belong to the
database library, the rules of the person belong to the domain, and the
server never names them.

The database is **still optional**. The server starts without one, the three
tools are still there, and every call answers that the database is not
configured. When the configuration names one, the health report carries a
`database` check, and a database that does not answer makes the service
unhealthy.

The pool is lazy: a database that is down does not stop the server from
starting. The health check is what asks the database whether it answers.

**The stack points the server at the database.** The credentials live with
the stack that owns them — `db/postgres/.env` — and the compose of the
server loads that file with `env_file`, so they are written once and only
once. The server assembles the connection string itself, because Docker
Compose does not interpolate `${VAR}` from the `env_file` of a service into
its `environment` block.

The connection string is resolved in this order:

1. `PACA_DATABASE_URL` — the whole connection string. When it is set, it
   wins, and nothing else is read.
2. Otherwise the connection string is assembled from the credentials of the
   PostgreSQL stack — `POSTGRES_USER`, `POSTGRES_PASSWORD` and
   `POSTGRES_DB` — with `PACA_DATABASE_HOST` (default `postgres`) and
   `PACA_DATABASE_PORT` (default `5432`):
   `postgres://<user>:<password>@<host>:<port>/<database>?sslmode=disable`.
   The user and the password are escaped the way a URL carries them, so a
   password with an `@` or a `/` in it does not break the connection string.
3. When neither of them is there, the server runs **without a database**, and
   its health report says nothing about it. Credentials that are half there
   are ignored the same way: the server runs without a database and says so
   in its log. A variable that is there and holds something invalid is
   still an error, never a silent fallback.

The URL names the container, so it is the same URL the host does not reach:
the database publishes no port, and only the containers of the internal
network do. To point the server at a database by hand, name it whole:

```sh
export PACA_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
docker compose up -d --wait
```

## The tools

The server exposes forty-one tools: the way PACA onboards a person, the four
moments of the journey of a person inside PACA, the two tools of the
businesses of a person, the three tools of the book of their money, the
four tools of their debts and of the closing of their month, the four tools
of the utterance — the purchase the owner just said, the purchase she
confirmed, and the two that drop and read back the pending drafts — the
three tools of the catalog: the name she said matched against the names
the business buys, the catalog read with what is known of every ingredient,
and the completion of an ingredient with what the owner declared about it,
the tool of the recipes: the guide she just confirmed, the two tools of
the reference prices: the price of the market that PACA researched, and
the reading of the ones the business holds, and the tool of the stock: what
the business holds and what it cost, the tool of the labour she just
confirmed, the tool of the overhead of the house, the tool of the report
of the real price, the tool of the margin, the tool of the production and
the two tools of the expiry: the correction of the shelf life the owner
just declared, and the read of what expires, the tool of the waste the
bakery lost, the tool of the sale the bakery made, the tool of the exit
of a unit: the consumption of the owner and the donation, the tool of
the rotation: the order in which the units leave the book, and the two
tools of the parties: the person, the store or the farm that buys from the
bakery or that the bakery buys from, and the reading of the parties the
business knows, and the two tools of the invoices: the invoice that came
with the purchase, and the reading of the invoices the business holds.

| Tool                      | What it does                                                    |
|---------------------------|-----------------------------------------------------------------|
| `complete_onboarding`     | Registers a person, opens their Personal Business with its accounts, and moves them to the active status. One call, one transaction. |
| `get_user`                | Reads the data of a person who is already registered.            |
| `list_users`              | Reads the people PACA knows, as many of them as the agent asks. |
| `change_user_status`      | Moves a user to another status, and says why when it needs it.  |
| `update_user_contact_data`| Writes the name, the phone and the email of a user.              |
| `open_business`           | Opens a business of a person, with the book of accounts it starts with. One call, one transaction. |
| `read_businesses`         | Reads the businesses of a person: the Personal Business and the others they own. |
| `record_movement`         | Writes a movement of money in the book: it goes from one account to another one. |
| `read_movements`          | Reads the movements of a period: the libro diario.              |
| `read_balances`           | Reads the balance of every account and the money of the business: the libro mayor. |
| `record_debt`             | Opens a debt with its counterparty and its due date, and writes the money that opened it. One call, one transaction. |
| `list_debts`              | Reads the debts the book still holds: what the person owes, to whom, and the day it falls due. |
| `record_debt_payment`     | Writes the payment of a debt, and refuses a payment that passes what the debt still holds. |
| `read_month_summary`      | Closes a month: the income, the expenses, the net, and the debts that were open at the end of it. |
| `draft_purchase`           | Proposes the purchase the owner just said and keeps it as a draft of the business, with what it understood, what it still needs, what it was not sure about and what it filled. |
| `confirm_purchase`         | Writes the purchase of the draft the owner confirmed and moves the money of the payment from the account that pays to the inventory. One call, one transaction. |
| `cancel_draft`             | Drops the draft the owner does not want: it moves to the state `cancelled` and it never waits again. It writes nothing in the book. |
| `read_drafts`              | Reads the pending drafts of a business, each one with what it still needs, what it was not sure about and what PACA filled, so that a lost conversation can be recovered. It writes nothing. |
| `match_items`             | Matches the name the owner just said against the catalog of the names the business buys: the ingredient it names, the options it offers, or the news that the catalog does not know it. It writes nothing. |
| `read_ingredients`        | Reads the catalog of the names the business buys: every ingredient with its unit, the weight of a package, its allergens, its certification with its evidence and its conversion, and what each one is still missing. It writes nothing. |
| `complete_ingredient`     | Writes what the owner just declared about an ingredient of the catalog: the unit it is bought in, the weight of a package, the allergens, the certification with the evidence that proves it, the names she uses for it and the conversion of its raw grams into its prepared ones. |
| `record_recipe`           | Writes the recipe the owner just confirmed: the guide to learn and to teach, with its ingredients of the catalog and their grams, its steps, its yield and its shelf life. One call, one transaction. |
| `record_reference`        | Writes the price of the market that PACA researched as a REFERENCE of the business: the ingredient, the price, its unit, the market, the source and the day it was seen, with the confirmation of the owner. A reference is never the price of the ingredient. |
| `read_references`         | Reads the reference prices of a business, each one with its source and the day it was seen, for one ingredient or for every one of them. It writes nothing. |
| `read_stock`              | Reads the stock of a business: what it holds of every ingredient, the layers that are still open — the oldest one first, because the valuation is FIFO — and the money the stock cost. The money of a layer that is partly consumed is the money of what is left of it. It writes nothing. |
| `read_batch_cost`         | Reads the cost of a batch of a recipe: what the ingredients it needs cost today, taken from the layers of the stock in the order of FIFO, with the money per unit, per gram and per kilogram. The DIRECT cost, the LABOUR and the OVERHEAD of the batch travel apart: no layer hides inside another one, and the rates are the rates of the DIRECT cost, without the labour and without the overhead. It writes nothing. |
| `record_labor`            | Writes the labour the owner just confirmed: the hours she worked — the hours of her hands, the hours of the oven and the hours of the waiting — and the value of the hour she declared. At least one of the three kinds of the hours carries hours, and the other two are optional. The money of the labour is never inferred, and a labour whose value of the hour PACA filled by itself is refused, and the payroll of the employees is out of scope for now. |
| `record_overhead`         | Writes the overhead of the house the owner just confirmed: the electricity she paid, the hours the oven ran and the share of that electricity that belongs to the oven. The money of the overhead is the electricity times the allocation, it is never inferred, and an overhead whose electricity PACA filled by itself is refused. The overhead of a batch is derived from the hours of the oven of the steps of the recipe. |
| `read_real_price`         | Reads the report of the real price of a product: the DIRECT cost, the LABOUR and the OVERHEAD of a batch, each one with its money and its rates per unit, per gram and per kilogram, their sum shown apart, and the price the owner charges beside them. The money of the layers is the money of the whole batch, and the money of the price of the owner is the money in her basis. The price of the owner is hers and it is never inferred: when she declared none, the answer says that it is not known. It writes nothing. |
| `read_margin`             | Reads the margin of a product: what the owner charges above the DIRECT cost of a batch, and what the MARKET said about the same ingredients. Every reference of the market is shown with the place it was read in, the day it was seen and its age, and the ones older than the owner expects are marked as old. A reference is never the price of the ingredient, and it never moves the money of the book. The money of the layers is the money of the whole batch, and the money of the price of the owner is the money in her basis. It writes nothing. |
| `record_production`       | Writes the production of the bakery the owner just confirmed: the recipe she made, the day she made it, the grams that went in, the money of the layers the production consumed and the units that came out, each one with the day it expires. The grams and the units are DERIVED by the book: the grams come from the layers of the stock in the order of FIFO — the oldest layer leaves first — and the units are the yield of the recipe. A production the stock cannot cover is refused. One call, one transaction: the production and the layers it consumed are written whole or not at all. |
| `correct_shelf_life`      | Writes the correction of the shelf life the owner just confirmed: the recipe she corrected, the number of days what it makes keeps, the day she corrected it and her word. The shelf life of a recipe is an APPROXIMATION until she corrects it, and the correction is KEPT: the last one she declared is the shelf life that counts. The correction moves the units that were ALREADY made: every unit of that recipe expires again, the day it was made plus the days she corrected. The days are never invented by PACA. One call, one transaction: the correction and the days it moves are written whole or not at all. |
| `record_waste`            | Writes the waste of the bakery the owner just confirmed: the product she lost, the quantity of units, the unit, the reason — burnt, spoiled, dropped or returned — and the day it happened. The units that leave the book are the units of the product that EXPIRE FIRST, and the money of the waste is the money of those units: the cost it releases. The money is never lost from the book — it leaves the inventory and it lands in the account of the waste. The reason is what a later analysis reads, and it is never invented by PACA. One call, one transaction: the waste and the movement of its money are written whole or not at all. |
| `read_expiring`           | Reads what expires, and when: every unit of the business with the day it expires, the days that are left of it and whether it is ALREADY expired. The units are ordered by the day they expire — the one that expires first is the first one. The units that already left the book are not offered any more: the ones that were sold and the ones that were wasted are gone. A unit is expired on the day it expires and on every day after it, and a unit that expired is NEVER sold. It writes nothing. |
| `record_sale`             | Writes the sale of the bakery the owner just confirmed: the product she sold, the quantity of units, the unit, the MONEY the client paid and the day it happened. The money is what the client paid and it is never inferred by PACA: the price list of the products is a later work. The units that leave the book are the units of the product that EXPIRE FIRST, and the sale of a unit that EXPIRED is refused: the expired unit has to be written as a waste. The sale moves the money IN — it lands in the cash of the business, or in the account the owner names — and it RELEASES the cost of the units it sells: the cost leaves the inventory and it lands in the account of the cost of the sales. One call, one transaction: the sale and the two movements of its money are written whole or not at all. |
| `record_exit`             | Writes the exit of a unit of the book the owner just confirmed: the KIND of the exit — `consumption` (the owner ate the bread) or `donation` (the bakery gave it away) — the product, the quantity of units, the unit and the day it happened. The exit moves NO money: nobody paid for the bread the owner consumed or for the bread the bakery gave away. What it does is RELEASE the cost of the units that left the book — the units of the product that EXPIRE FIRST — and the cost leaves the inventory and lands in the account of the expense of the exit: the cost of the sales for the consumption, and the account of the donation for the donation, which is a social expense of the business. The TAX TREATMENT is never decided by PACA: it waits for the professional of the taxes of the owner, and the answer carries the question that waits. One call, one transaction: the exit and the movement of its cost are written whole or not at all. |
| `read_rotation`           | Reads the rotation of the book: the ORDER in which the units the bakery still holds leave it, and when each one expires. The rotation follows the EARLIEST EXPIRY: the unit that expires first is the first one that leaves, and the units that expire on the same day keep a stable order — the production first, and the position of the unit in it. The answer carries the name of the product, the day each unit was made, the PLACE it takes in that order, the DAY it expires and the DAYS that are left of it. The product is optional: the whole book of the business, or one product of it. The units that already left the book are not offered any more: the ones that were sold, wasted, consumed or given away are gone. The answer carries NO money: the valuation of the stock follows the FIFO, and it is another rule. It writes nothing. |
| `open_party`              | Opens a party of the business: the PERSON, the STORE or the FARM that buys from the bakery, or that the bakery buys from. The business is the business of its own — the one with the clients and the providers of the bakery — and the Personal Business of a person is refused. The kind is required, and the name is required: the name of a party is NOT unique inside the business, and two parties may share it. The roles are required: the CLIENT, the PROVIDER, or both of them, and a party that holds neither is refused. The legal human that represents the party is a FACT OF THE PARTY and never a second person: a store REQUIRES it, a farm CARRIES it only when it has one, and a person REFUSES it. The phone, the email and the address are optional facts of the party, and PACA never invents them. |
| `read_parties`            | Reads the parties of the business: every person, store and farm it knows, in the order of their identity, with the roles it holds, the way of reaching it, its address and the legal human that represents it. The name of a party is not unique inside the business, and the answer carries the ones that share it. It writes nothing. |
| `record_invoice`          | Writes the invoice that came with a purchase: the provider that issued it, the number it carries, the day it was issued, the base the tax applies to, the RATE the invoice states, the tax that travels with it and the total of the invoice. The tax is NEVER assumed: there is no default of nineteen percent and no default of zero, and the tax is derived from the rate and the base the invoice states, rounded to the nearest whole peso with a half peso going up. An invoice that does not add up is refused, and so is a provider that does not hold the role of the provider. The invoice moves NO money: the purchase already moved the money of the payment. The reference of the document the owner shows is optional, and PACA never assumes an electronic invoice. One call, one transaction: the invoice and the purchase that points at it are written whole or not at all. |
| `read_invoices`           | Reads the invoices of the business: every invoice it holds, in the order of their identity, with the provider that issued it, the purchase it came from, its number, its day, its base, the rate the invoice states, the tax that travels with it and its total. The tax of an invoice is never assumed: it is the one the invoice states. It writes nothing. |

**There is one tool for the registration and the onboarding.** The tool of
the onboarding registers the person and opens their book in one call, and a
tool that registers a person and stops there is a tool the agent can call
halfway: it leaves a person in the onboarding with no business, and the
agent has to remember to come back.

**The description of a tool is not a comment.** It is the instruction the
agent reads before it calls the tool, so it says what the tool needs, what it
does with what it receives, and what it must not do. The description of
`complete_onboarding` tells the agent that the onboarding is ONE call and
ONE transaction, that the document ID and the name are required and that
PACA never invents the first, that the phone and the email are a favour to
ask for kindly, that the phone must carry its country code, that the email
must be a valid address, that the book of the person starts empty, that the
call is safe to repeat, and it names the accounts the book starts with — a
list read from the domain, never written a second time. The description of `change_user_status` tells the agent the moves the
map of the states allows and the rule of the observation. The description of
`update_user_contact_data` tells the agent that an empty field is a field
that stays as it is. The description of `get_user` tells the agent that the
tool answers with what PACA knows about the person, and the description of
`list_users` tells the agent to ask for the people it needs and to ask again
when the answer carries more people than it asked for.

**The business of a person has a book of its own.** The tool that opens a
business writes the business and the accounts of its book in one
transaction: a book that starts with half of its accounts lies about what
the owner has. The name of the business is required, the currency is
optional and it means the peso when the agent names none, and the kind of
the business is always `business` — the Personal Business of a person is
opened by the onboarding. The name of a business is taken inside the
businesses of its owner, and PACA refuses the duplicate. The description of
`open_business` names the accounts the book starts with and the kinds of a
business, and the description of `read_businesses` says that the tool
changes nothing: both lists are read from the domain and never written a
second time.

**The money goes from one account to another one.** The tool of the movement
takes the account the money comes from and the account it lands in, the
way the person says it: a salary lands in the cash and comes from the
salary. The amount is positive and it is never a fraction, the day is the
day the money moved — the day of today when the agent names none — and the
description is what the person reads in their book.

**The utterance is a draft, and a draft waits for the yes.** The tool
`draft_purchase` takes the business and the sentence of the owner verbatim,
with the lines the agent read from it: the name as she said it, the
quantity, the unit and the money. It writes a draft and nothing else — the
book of the business does not move until the owner says yes — and it answers
with the three parts of the answer of Paca: what it understood, what it did
not understand and what it still needs. The money is never inferred: a
price the agent filled is refused, and a line the owner did not price is
what the draft still needs. The quantity and its unit travel together: the
unit of a line is the unit the owner named, and the unit of the ingredient
of the catalog when she named none, and a line that ends without a unit is
refused. The unit is stored the way the BOOK knows it: Paca reads the way
she says it ("kilos", "gramos") and saves "kg", "g", "paquete", "unidad". And when the owner named both the money that was paid and the
price of the unit, the two must agree. The day is one of the three things
Paca may fill by herself, so a draft that names no day is the day of today
and the answer says so. A draft expires in seven days if the owner says
nothing.

**The confirmation writes the purchase and moves the money.** The tool
`confirm_purchase` takes the business, the draft, the revision the owner
saw and the word she used to confirm it. The account that pays is optional:
the money comes from the cash of the business, and from the account the
owner names when she paid with another one. The purchase, its lines and the
movement of the money are written together or not at all: a book that holds
a purchase and no movement of its money lies about what the owner bought.
The confirmation is refused when the draft expired, when the draft changed
under the eyes of the owner, and when the draft still needs the money of one
of its lines: the money is never inferred. A refused confirmation leaves
nothing behind — the draft stays as it was and no purchase is written. The
answer carries the purchase that was written and the three parts of the
answer of Paca: what it recorded, what it did not understand and what it
still needs. The answer names the names the catalog of the business did not
know: a name the catalog does not know is not an error, it is recorded as
the owner said it, and the agent tells her that the name is new.

**The pending drafts can be dropped and read back.** The tool `cancel_draft`
takes the business and the draft, and it drops the draft the owner does not
want: the draft moves to the state `cancelled` and it never waits again. A
draft that is confirmed already and a draft that is cancelled already are
refused — they do not wait any more — and so is a draft that expired: the
price of a market does not wait. The tool `read_drafts` takes the business
and answers with the drafts that still wait for the yes of the owner, each
one with what it still needs, what it was not sure about and what PACA
filled by herself: a lost conversation is recovered from the book of the
business and not from the memory of the model. The day is optional and it
means the day of today when it is empty: the day decides which drafts are
still pending, because the expiry is computed when the draft is read. Both
tools change nothing in the book.

**The catalog of the names begins empty.** The tool `match_items` takes the
business and the name as the owner said it, and it answers with the status
of that name: `exact` when one ingredient answers it, `ambiguous` when
several answer it — with the options, so the owner chooses with one word —
and `unknown` when none of them does. A name the catalog does not know is
not an error: the answer says that it is new, and the confirmation of the
purchase records it as she said it. The tool writes nothing: the catalog
grows with the confirmation of a purchase, never with a read.

**The catalog is read, and the ingredient is completed.** The tool
`read_ingredients` reads the catalog of a business and answers with what is
known of every ingredient — the unit it is bought in, the weight of a
package, the allergens it carries, the certification it has and the
conversion of its raw grams into its prepared ones, with the weight of its
unit in grams — and with what each one is still missing, so the agent asks
the owner for exactly what she has not said yet. The tool `complete_ingredient` writes what the owner declared
about one of them: the unit, the weight of a package, the allergens, the
certification with the document that proves it, the names she uses for it
and the conversion. A claim carries its evidence: PACA refuses an
"organic" without the document that proves it, and the two fats are two
ingredients with their own certification. The units of the book are the
gram, the kilogram, the package and the piece, and the weight of a package
is never assumed.

**The recipe is a guide, and it is written with the yes of the owner.** The
tool `record_recipe` takes the business, the name the owner reads, the
ingredients of the catalog with their grams, the steps in the order she
said them, the yield, the shelf life of what the recipe makes and the word
she used to confirm it. A recipe is not a list of ingredients: it is a
guide to learn and to teach, and the steps are part of it — with their
times, their temperatures and their cues, when she named them. The grams of
an ingredient are never inferred, and the yield is a declared constant of
the recipe, never measured per bake. The ingredient of a recipe is the
ingredient of the catalog: PACA refuses an ingredient the catalog of the
business does not hold, and the answer tells the agent to match the name
first. The recipe, its ingredients and its steps are written together or not
at all: a recipe without its ingredients and its steps is not a guide, and a
refused recipe leaves nothing behind.

**A reference is never the price of the ingredient.** The price the owner
pays exists only in the market of her town and only in her mouth on the
day: it comes from her own purchases. The tool `record_reference` writes
what the research of PACA found — the ingredient, the price, the unit of
that price, the market where it was seen, the place it was read and the day
it was read — with the confirmation of the owner, the way the purchase is
written. The answer carries the kind of what it is (`reference`), so that
nothing may read it as the price of the ingredient. The source is never
empty: a price without a source is a price nobody can check, and PACA
refuses it. The tool `read_references` reads the references of the
business — of one ingredient, or of every one of them — with their source
and their day, and it writes nothing.

**The stock is the sum of the layers that are still open.** The purchase is
the fact of what the business bought, and the layer is derived from it: the
ingredient, the quantity with its unit, the money it cost, the day it was
bought and the purchase it comes from. The tool `read_stock` reads what the
business holds — the quantity of every ingredient, the layers that are
still open and the money the stock cost — and it writes nothing. The
business is required, and the tool refuses the business that is not there.
The valuation is FIFO: the oldest layer is the first one that leaves, and the
answer carries the layers in that order. The order of the output by expiry
is a different rule, and it does not live here. The quantity and its unit
travel together: the layers bought in different units are held apart, and
the book never adds a gram to a kilogram. The layer carries the unit the
book knows (`g`, `kg`, `paquete`, `unidad`), and so does the purchase it
comes from: the way the owner said it is read by the domain and never
stored. The money of a layer that is partly consumed is the money of what is
left of it: the money the layer bought, pro rata the quantity that is left,
and the layers of a holding sum to the money of the holding.

**The report of the real price shows the three layers apart.** The tool
`read_real_price` reads the report of what a product really costs: the
direct cost of the ingredients, the labour of the hours of the recipe and
the overhead of the hours of the oven — each one with its money and its
rates per unit, per gram and per kilogram — and their sum, shown apart, so
that no layer of the cost hides inside another one. The price the owner
charges travels beside them, with the way she charges it (`unit`, `gram` or
`kilogram`), and the book never moves it: the money of what she sells is
never inferred, and a report of a price nobody declared says that it is not
known. The two monies travel apart: the money of the layers is the money of
the whole batch, and the money of the price of the owner is the money in her
basis — the price of one unit, of one gram or of one kilogram. The tool
writes nothing, and the labour and the overhead move no money of the ledger
yet: their accounts exist in the seed of the book, and the movement belongs
to a later increment.

**The margin compares the price of the owner with the cost and with the
market.** The tool `read_margin` reads what a product earns: the profit
above the DIRECT cost of the batch — per unit, per gram and per kilogram,
and the percent it is of the direct cost — and, beside it, what the MARKET
said about the same ingredients: what the owner paid for them, what the
market says they cost, and the difference between the two. The reference is
a reference: it never becomes the price of the ingredient, and it never
moves the money of the book — the direct cost comes from the layers of the
stock. Every reference is shown with the place it was read in, the day it
was seen and its age, and the newest one of every ingredient is the one
that compares, because the market of today is the last one that was seen.
When the owner expects the references to be fresh, the agent sends
`max_age_days`, and the ones older than that are marked as old, with their
age. The expectation of the age is a number of days: a negative one is
refused, and a refused read reads nothing. The money of the price of the
owner is the money in her basis, and the money of the layers is the money of
the whole batch. The tool writes nothing.

**The production is derived, never typed.** The tool `record_production`
writes what the bakery made on a day: the recipe she made, the day she
made it, the grams that went in, the money of the layers the production
consumed and the units that came out, each one with the day it expires.
The grams and the units are never asked to the owner: the grams come from
the layers of the stock in the order of FIFO — the oldest layer is the
first one that leaves — and the units are the yield of the recipe, each
one with the day of the production plus the shelf life of the recipe. The
quantity that is left of each layer is the quantity it held minus what the
production took from it, and the unit travels with it: the book never
subtracts a gram from a kilogram. A production the stock cannot cover is
refused, and the answer says that the purchase of the missing ingredient
comes first. The production and the layers it consumed are written in one
transaction: a production whose layers did not leave the stock leaves
nothing behind. The recipe is named by her name or by its identity, and a
name that more than one recipe carries is refused: the book never guesses
which recipe she made. The word she used to confirm it is kept with the
record, and her sentence is kept verbatim when she said one.

**The expiry is corrected, and it is read.** The shelf life of a recipe is
an approximation until the owner corrects it, and the tool
`correct_shelf_life` writes that correction with her word: the number of
days is what she said, and PACA never invents it. The correction is KEPT:
every correction she declared is written down, and the last one is the
shelf life that counts — so the units that come out of the oven from now
on expire the day of the production plus the days she corrected, and the
units she already made expire again. The correction and the days it moves
are written in one transaction, and a unit that changed under the
correction is refused: the book is never left half-corrected. The tool
`read_expiring` answers what expires, and when: every unit of the business
with the day it expires, the days that are left of it and whether it is
already expired, the unit that expires first first. The units that already
left the book are not offered any more: the ones that were sold and the ones
that were wasted are gone, and the answer carries the units the bakery still
holds. A unit is expired on the day it expires and on every day after it,
and a unit that expired is never sold. The two tools write nothing in the book of the money: the
expiry moves the day a unit expires, and never a peso.

**The waste is a movement, and it is written with the word of the owner.**
The tool `record_waste` writes what the bakery lost: the product she lost,
the quantity of units, the unit, the reason — burnt, spoiled, dropped or
returned — and the day it happened. The reason is what a later analysis
reads, and it is never invented: PACA asks her why she lost them. The waste
counts the UNITS it lost: the book never converts a weight into a number of
units, and it never splits a unit. The units that leave the book are the
units of the product that EXPIRE FIRST — the rotation of the output, a rule
apart from the FIFO of the valuation — and the money of the waste is the
money of those units: the cost it releases, the money of the production
they came out of divided by the units that production made. The money is
never lost from the book: it leaves the inventory and it lands in the
account of the waste, and the waste and the movement of its money are
written in one transaction. The word she used to confirm it is kept with
the record, and her sentence is kept verbatim when she said one.

**The sale is a movement, and it moves the money in and the cost out.**
The tool `record_sale` writes what the bakery sold: the product she sold,
the quantity of units, the unit, the money the client paid and the day it
happened. The money is what the client paid, and it is never inferred: PACA
asks her what the client paid. The sale counts the UNITS it sold: the book
never converts a weight into a number of units, and it never splits a unit.
The units that leave the book are the units of the product that EXPIRE
FIRST — the rotation of the output, a rule apart from the FIFO of the
valuation — and a unit that EXPIRED is never sold: the sale is refused, and
the expired unit has to be written as a waste. The cost of the sale is
derived, never typed: the money of the production the units came out of,
divided by the units that production made. The money of the sale lands in
the account that receives it — the cash of the business, or the one the
owner names — and it comes from the account of the sales; the cost leaves
the inventory and it lands in the account of the cost of the sales. The
sale and the two movements of its money are written in one transaction. The
word she used to confirm it is kept with the record, and her sentence is
kept verbatim when she said one.

**The exit is a movement, and it moves no money at all.**
The tool `record_exit` writes the exit of a unit of the book: the KIND of it
— `consumption` when the owner ate the bread, `donation` when the bakery
gave it away — the product, the quantity of units, the unit and the day it
happened. The exit moves NO money: nobody paid for the bread the owner
consumed or for the bread the bakery gave away. What the exit does is
RELEASE the cost of the units that left the book: the units of the product
that EXPIRE FIRST, and the cost is derived, never typed — the money of the
production they came out of, divided by the units that production made.
The cost leaves the inventory and it lands in the account of the expense of
the exit: the cost of the sales for the consumption, and the account of the
donation for the donation, which is a social expense of the business. The
TAX TREATMENT is never decided by PACA: it waits for the professional of the
taxes of the owner, and the answer carries the question that waits. The
exit and the movement of its cost are written in one transaction. The word
she used to confirm it is kept with the record, and her sentence is kept
verbatim when she said one.

**The rotation is the order in which the units leave the book.**
The tool `read_rotation` answers that order: the units the bakery still
holds, the one that expires first first, with the place it takes in that
order, the day each one expires and the days that are left of it. The rotation follows the EARLIEST EXPIRY, and
the units that expire on the same day keep a stable order: the production
first, and the position of the unit in it. The answer holds only the units
that are still in the book: the ones that were sold, wasted, consumed or
given away are gone. The rotation may be asked for the whole book of the
business or for one product of it, and the agent names the product when the
owner asks about one product. The answer carries NO money: the FIFO is the
rule of the valuation of the stock, and the rotation is the rule of the
output — the two rules are apart, and the model keeps them apart. The tool
writes nothing.

**The party is the other side of the bread.** The tool `open_party` writes
the person, the store or the farm that buys from the bakery or that the
bakery buys from, and the tool `read_parties` answers the parties the
business knows. The kind of the party is one of the three of the owner, and
the roles it holds are the CLIENT and the PROVIDER: it may hold both, and
it may hold neither only when it holds none — the bakery neither sells to
it nor buys from it, and it is not a party of the book. The party belongs
to a business of its own: the Personal Business of a person holds the money
of that person and not the clients and the providers of the bakery, and the
tool refuses a party of it.

**The legal human that represents a store or a farm is a fact of the party,
and never a second person.** A store REQUIRES the human that represents
it, a farm CARRIES it and its legal figure only when it has one — some
farms have a legal figure and some do not — and a person REFUSES it. The
human is never a user of PACA, never a party of its own, and never a row of
the people of PACA: it travels inside the party it belongs to, and nowhere
else. The name of a party is NOT unique inside the business: the book never
refuses two parties that share a name, and the reading of the parties holds
all of them.

**The invoice is a fact that comes with the purchase, and it moves no
money.** The tool `record_invoice` writes the invoice against the purchase
it came from, and the purchase points at it. The purchase already moved
the money of the payment when the owner bought, and the invoice arrives
later: the book never moves money twice for the same purchase. The tax of
an invoice is NEVER ASSUMED: it comes from the invoice, which states the
rate it was issued with and the base that rate applies to. The tax is
derived from the two of them and rounded to the nearest whole peso, with a
half peso going up, and an invoice that does not add up is refused. The
provider of an invoice is a PARTY that holds the role of the provider, and
the number of the invoice is unique inside the business. The electronic
invoice is never assumed either: there is no integration with the tax
authority, and the reference of the document the owner shows is a plain
fact of the invoice.

**The failures are answered in words the agent can act on.** A document that
is not a number, a phone without a country code, an address that is not an
address, a document that is already registered, an email that belongs to
somebody else, a move the map does not allow, an observation that does
not fit the status, a name that is missing, a business that is already
there, an account whose name is taken inside its business, a recipe without
a name or without a yield, an ingredient of a recipe without its grams and
a step without its instruction, a reference without its source, without its
day or without the ingredient of the catalog: each one says what happened
and what to do about it.

The rules themselves live in the [domain](../../packages/domain): the cleaning of
the phone, the check of the email, the map of the states and the rule of the
observation, the rule of the complete onboarding, the rule of the personal
business and the rule of the debt. The tools only carry them to the agent.

## The endpoints

| Path       | Method            | What it answers                                          |
|------------|-------------------|----------------------------------------------------------|
| `/healthz` | `GET`             | The health of the service: 200 while it serves, 503 when it does not. |
| `/mcp`     | `POST` (`GET`, `DELETE`) | The MCP transport, over the streamable HTTP transport. |

The health endpoint answers a JSON body:

```json
{
  "status": "ok",
  "service": { "name": "paca-mcp-server", "version": "0.1.0" },
  "uptime_seconds": 42.5,
  "checked_at": "2026-10-27T10:00:00Z",
  "checks": [ { "name": "mcp-transport", "status": "ok", "latency_ms": 0 } ]
}
```

## The configuration

Everything comes from the environment. An invalid value is an error, never a
silent fallback.

| Variable                    | Default     | What it is                                            |
|-----------------------------|-------------|-------------------------------------------------------|
| `PACA_MCP_HOST`             | `127.0.0.1` | The interface the server binds. The container overrides it with `0.0.0.0`. |
| `PACA_MCP_PORT`             | `8081`      | The port reserved for the MCP server.                 |
| `PACA_MCP_SHUTDOWN_TIMEOUT` | `10s`       | The grace period of the shutdown.                     |
| `PACA_MCP_CHECK_TIMEOUT`    | `2s`        | The bound of a single health check.                   |
| `PACA_MCP_MAX_REQUEST_BYTES`| `4MiB`      | The largest body the MCP transport accepts.           |
| `PACA_MCP_ALLOW_NON_LOOPBACK` | `false`  | The server may bind an address that is not the loopback interface. The container sets it: it is the deployment where the other containers of the stack reach the server. |
| `PACA_MCP_ALLOWED_HOSTS`    | `localhost,127.0.0.1,[::1]` | The hosts the server answers for. The `Host` header of a request that names another one is refused. The container names itself. |
| `PACA_MCP_TRUSTED_ORIGINS`  | *empty*     | The origins the server trusts. The rule of the standard library — the same origin as the `Host` header — is the rule when the list is empty. |
| `PACA_LOG_LEVEL`            | `info`      | The verbosity of the logs: `debug`, `info`, `warn`, `error`. |
| `PACA_SERVICE_VERSION`      | the version of the binary | The version the clients see.             |
| `PACA_DATABASE_URL`         | *empty*     | The connection string of the database. When it is set, it wins and nothing else is read. Empty means the server runs without one. |
| `PACA_DATABASE_MAX_CONNS`   | `4`         | The number of connections the pool of the database may open. |
| `PACA_DATABASE_CONNECT_TIMEOUT` | `5s`    | The bound of the attempt to open a connection. |
| `PACA_DATABASE_HOST`        | `postgres`  | The host the database answers on. It is only read when the connection string is assembled from the credentials. |
| `PACA_DATABASE_PORT`        | `5432`      | The port the database answers on. It is only read when the connection string is assembled from the credentials. |
| `POSTGRES_USER`             | *empty*     | The role that owns the application database. It travels from `db/postgres/.env` with the `env_file` of the compose. |
| `POSTGRES_PASSWORD`         | *empty*     | The password of the role. It travels from `db/postgres/.env` with the `env_file` of the compose. |
| `POSTGRES_DB`               | *empty*     | The application database. It travels from `db/postgres/.env` with the `env_file` of the compose. |

Without `PACA_DATABASE_URL` and without the credentials of the PostgreSQL
stack, the server runs without a database. Credentials that are half there
are ignored: the server runs without a database and says so in its log.

### The tracing

The service traces its requests with [OpenTelemetry](https://opentelemetry.io)
and sends the traces to a collector over OTLP. The settings are the ones of
the OpenTelemetry specification, so an operator who knows the specification
knows how to turn the tracing on and off.

| Variable                          | Default     | What it is                                            |
|-----------------------------------|-------------|-------------------------------------------------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT`     | *empty*     | The address of the collector: `host:port`. The tracing is **off** without it: there is nowhere to send the traces. |
| `OTEL_EXPORTER_OTLP_PROTOCOL`     | `grpc`      | The protocol of the exporter: `grpc` or `http/protobuf`. |
| `OTEL_EXPORTER_OTLP_INSECURE`     | `false`     | Whether the exporter speaks to the collector without TLS. |
| `OTEL_SDK_DISABLED`               | `false`     | Turns the tracing off.                                 |
| `OTEL_TRACES_SAMPLER`             | `parentbased_always_on` | The sampler: `always_on`, `always_off`, `traceidratio`, `parentbased_always_on`, `parentbased_traceidratio`. |
| `OTEL_TRACES_SAMPLER_ARG`         | *empty*     | The ratio of the traces that are kept, for the samplers that take one. |
| `OTEL_SERVICE_NAME`               | `paca-mcp-server` | The name the telemetry carries.                  |
| `OTEL_RESOURCE_ATTRIBUTES`        | *empty*     | The attributes of the deployment, as `key=value` pairs. |

An invalid value is an error, never a silent fallback: a server that starts
with a collector it cannot reach, or with a sampler it cannot read, is a
server that nobody asked for.

### The logs and the traces

The logs go to the standard output, as JSON. Every record carries the
`trace_id` and the `span_id` of the trace it belongs to, when the request
that wrote it was inside a trace. That identifier is the join between the
two: paste it into the search of the trace backend, and the whole request
the line belongs to shows up.

A record written outside of a request carries no identifier: there is no
trace to point at.

```json
{"time":"2026-10-27T10:00:00Z","level":"INFO","msg":"request","method":"POST",
 "path":"/mcp","status":200,"duration":1200000,
 "trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}
```

The traces themselves live in the [observability stack](../../observability),
which keeps them for two days.

## Running it

```sh
make -C apps/mcp-server build          # build the binary into bin/
make -C apps/mcp-server run            # run the server on 127.0.0.1:8081
```

Or as a container, with the stack of the monorepository:

```sh
make -C apps/mcp-server compose-up     # start the stack and wait until it is healthy
make -C apps/mcp-server smoke          # ask the running server whether it is alive
make -C apps/mcp-server compose-down   # stop the stack
```

The stack of the server loads the credentials of the PostgreSQL stack, so it
expects a database to answer at `postgres:5432`: the database of the
monorepository, on the network the stacks share. Start the database first
(`make db-up`) or start the whole monorepository (`docker compose up
--wait`). Without a database the server still serves, and its health report
says that the database does not answer.

## The quality gate

```sh
make -C apps/mcp-server all
```

The gate runs, in this order: the format check, the analyzers of the
toolchain, the linter, the build, the test suite, the race detector, the
coverage gate and the vulnerability scan.

| Target            | What it does                                                     |
|-------------------|------------------------------------------------------------------|
| `make deps`       | Downloads and verifies the dependencies.                          |
| `make tidy-check` | Fails when `go.mod` is not tidy.                                 |
| `make fmt-check`  | Fails when a source is not formatted.                             |
| `make vet`        | Runs the analyzers of the toolchain.                              |
| `make lint`       | Runs `golangci-lint`, with `gosec` among the linters.             |
| `make build`      | Builds the binary into `bin/`.                                    |
| `make test`       | Runs the test suite.                                              |
| `make test-integration` | Builds the image, starts the container and talks to it.     |
| `make cover`      | Measures the coverage and **fails below 80%**.                    |
| `make race`       | Runs the test suite with the race detector.                      |
| `make bench`      | Runs the benchmarks.                                              |
| `make perf`       | Checks the allocation budgets, then runs the benchmarks.          |
| `make vuln`       | Runs `govulncheck` over the dependencies.                         |

The tools are pinned in the repository (`tools/go.mod`), so `go tool` runs the
same version everywhere. A contributor needs nothing but the Go toolchain.

## The tests

The unit tests live with the code they cover. The container tests live in
`test/integration` and need Docker:

```sh
make -C apps/mcp-server test-integration
```

They build the image of the server from its Dockerfile, start it, and check
that it comes up, that it answers its health, that it speaks the MCP
protocol, and that it keeps its transport to itself.

## The security of the slice

- The server is private: no published port, only the `paca_internal`
  network, which has no outbound access.
- The image runs as a non-root user, on a read-only file system, without any
  capability, with `no-new-privileges`, on a distroless base that has no
  shell and no package manager.
- The DNS-rebinding protection of the MCP SDK stays on: a request that
  arrives through the loopback interface with a foreign `Host` header is
  rejected.
- The server **refuses a bind that is not the loopback interface**, unless
  the operator allows it with `PACA_MCP_ALLOW_NON_LOOPBACK`. The tools
  carry the data of the people, and a bind that the network of the machine
  reaches is a bind that anybody on the network reaches. The container is
  the deployment where the other containers of the stack reach the server,
  over a network that has no route to the internet, and the compose file of
  the service says so.
- The server **answers only for the hosts of its allowlist**
  (`PACA_MCP_ALLOWED_HOSTS`): the `Host` header of a request that names
  another host is refused with a `403`, and the refusal never reaches the
  tools. The default is the loopback, and the container names itself.
- The cross-origin protection of the standard library stays on, and its rule
  is **stated and not left to the default**: the service trusts the origins
  of `PACA_MCP_TRUSTED_ORIGINS`, and the rule of the standard library — the
  same origin as the `Host` header — is the rule when the list is empty. A
  cross-origin request from a browser is rejected, and the requests of a
  client that is not a browser are let through.
- The body of a request is bounded, the headers are bounded, and the header
  read is timed out.
- There is no authentication yet: the network is the boundary. The token and
  the TLS come with the first tool that needs them.
- The five tools carry **personal data**: the document, the name, the phone
  and the email of a person. The boundary has not moved — it is still the
  network — and the hardening that was proposed for it (refusing a bind that
  is not loopback, a Host allowlist and the explicit Origin rule) is closed.
  There is still no authentication: the token and the TLS come with the
  first tool that needs them.

## The versions

| What                | Version  | Why                                                     |
|---------------------|----------|---------------------------------------------------------|
| Go                  | 1.27.3   | The latest security release of the 1.27 line (2026-10-19). The local toolchain of the repository is 1.26.3, which is below the security release 1.26.10. |
| MCP Go SDK          | v1.8.0   | The official SDK, with the protocol revision `2026-07-28`. |
| golangci-lint       | v2.15.0  | Pinned in `tools/go.mod`.                               |
| govulncheck         | v1.8.0   | Pinned in `tools/go.mod`.                               |
| testcontainers-go   | v0.44.0  | The container tests.                                    |
| OpenTelemetry Go    | v1.47.0  | The tracing: the SDK, the OTLP exporter and the instrumentation of the transport. |
| otelhttp            | v0.72.0  | The instrumentation of the HTTP transport.               |
