# The domain of the PACA monorepository.

The domain holds what the services of the monorepository share: the model
of a *person*, the rules a person carries, the port of the storage that keeps
them, the state of the *schema* of the database — the version it is at, the
migrations it carries and what is pending — and the model of the *health* of
a service: the states it is in, the checks it runs and the report it answers.

A model has one owner. The person lives here because more than one service
needs it, the state of the schema because two sides need it — the service
that writes the schema and the library that applies the migrations — and the
health because every service with a health endpoint speaks it. Both sides
speak the same record, and the adapter between them maps nothing.

It is a **library**. It publishes no port on the network, it runs no
container, and it starts no process.

---

## The layout

```
packages/domain/
  domain.go         the package comment: the models and the rules, nothing else
  user/             the model of the person: the identifier, the states, the
                    details, the registration and the port of the storage
  business/         the model of the business that owns a book: the identity
                    the database generates, the kinds and the port of the
                    storage
  account/          the model of the account of a book: the five kinds of the
                    accounting, the debt and the seed of the personal book
  ledger/           the model of the movements of the book: the entry, the
                    amount, the day, the balance of an account, the money of
                    the business, the period of a reading and the debt of an
                    account
  purchase/         the model of the draft of a purchase and of the purchase
                    itself: the candidate the agent proposes, the lines, the
                    measure, the money, what the draft still needs, the
                    confirmation, the reference price of the market, the
                    price of the owner with the way she charges, the
                    layers of the stock and the port of the storage
  ingredient/       the model of the catalog of the names the business buys:
                    the name the owner reads, the names she uses for it, the
                    matching of a name she said, the units of the book, the
                    certification with its evidence, the conversion of the
                    raw into the prepared and the port of the storage
  recipe/           the model of the recipe of the bakery: the guide to
                    learn and to teach, the ingredients with their grams, the
                    steps with their times, their temperatures and their
                    cues, the yield, the shelf life, the cost of a batch,
                    the labour of the owner, the overhead of the house,
                    the report of the real price, the production of the day
                    with the units it made and the day they expire, the
                    correction of the shelf life, the read of what
                    expires, the waste of the bakery, the sale of the
                    bakery, the exit of a unit - the consumption of the
                    owner and the donation - and the port of the
                    storage
  store/            the port of a transaction of the storage: the unit of work
  party/            the model of the party of the business: the person, the
                    store and the farm that buy from it or that it buys
                    from, the legal human that represents a store or a
                    farm, the roles of the client and of the provider, and
                    the port of the storage
  product/          the model of the product of the bakery: the thing it
                    sells, the channel that buys it — the retail and the
                    wholesale — the price with the day it was set, and the
                    port of the storage
  order/            the model of the order of the bakery: what it owes to a
                    client, the lines with the products and the quantities
                    of them, the state of the order, the round of a day and
                    the port of the storage
  schema/           the state of the schema of the database: the version, the
                    migrations and what is pending
  health/           the model of the health of a service: the states, the
                    checks, the report and the rule that one failing
                    component makes the whole service unhealthy
```

The dependency rule of the hexagonal architecture holds here, and the
toolchain enforces it: **the module imports nothing but the standard
library.** A model that reached for a driver, for a table or for a
connection string would break the build of every service of the
monorepository — the compiler says so before a reviewer has to.

That is the reason this module is a module of its own, and not a package of
the database library: the boundary between the two is a `go.mod`, and the
visibility rule of `internal` does not cross it. A package of the same
module would organize the code, but nothing would forbid the domain from
importing the generated code of the queries.

## The model of the person

`user.User` is the person that PACA knows. It carries the identifier, the
details, the state and the timestamps of the person.

**The identifier is the national document ID of the person.** The caller
supplies it, and nothing in the domain generates one: the number is not
sequential because it is not a counter, it is the number the person already
carries. A person writes `1.234.567` and `1234567`, and both are the same
document.

## The rules of the person

The domain package holds the rules that a table cannot.

**The phone is cleaned the way a person writes it.** The plus of the
country code, the spaces, the dashes, the dots and the parentheses go away,
and what is left are the digits: `+57 321-253-9984` is stored as
`573212539984`. A number that carries no country code is refused: ten
digits are exactly the length of a Colombian number, and that is the number
that must be written with its code.

**The email is checked** and stored in its canonical form: an address that
does not look like an address is refused, and the canonical form of an
address is its lower case. Two addresses that differ only in the case of the
name are the same mailbox, and the mailbox belongs to one person, so the
domain stores one value for them and never two.

The table holds the same rules as a defense in depth: its unique index
refuses two rows that carry the same mailbox, and its constraint refuses an
address that is not in its canonical form. The table guards against the
writer that does not know the domain; the domain is still the one that
refuses.

**The states move through a map.** The map of the transitions lives in the
domain, and it is the only place that knows it:

```
onboarding ──▶ active      active     ──▶ inactive   inactive   ──▶ active
onboarding ──▶ inactive    active     ──▶ blocked    inactive   ──▶ blocked
onboarding ──▶ blocked     blocked    ──▶ active     blocked    ──▶ inactive
```

No move leads back to the onboarding: that state is the entry of the
machine, the state of a person that has just been created, and it is never
a destination.

**The map is published.** `user.AllStatuses()` returns the four states and
`user.AllTargets()` the states a person may reach, and `user.AllowedTargets()`
the states one state may move to. The surfaces that show the states of the
person — the schema of a tool, the instruction an agent reads — read them from
here, so they never drift from the rule.

**The observation belongs to the status.** The two closed states —
`inactive` and `blocked` — carry the reason why the person is not using the
product, and the two open states carry none.

## The repository of the people

The domain publishes the port of the storage of the people:

```go
Create(ctx, id, details) (user.User, error)
Find(ctx, id) (user.User, error)
UpdateDetails(ctx, id, details) (user.User, error)
ChangeStatus(ctx, id, status, observation) (user.User, error)
```

There is no delete and no list: a person that must go is `blocked`, and
nothing consumes a list yet.

The read is there because a use case asks for a person by its document: the
tool that reads the data of a person needs it, and a port holds the methods
the use cases call. The adapter reads the row it needs by itself, when the
move of a state needs it, and that read is the same one.

The port is what a use case needs, not the shape of a table. It lives here,
in the module of the models, because both sides of it need it: the service
that writes the people, and the adapter that reads the table. An adapter of
another storage — a gRPC client of a data service, tomorrow — implements
the same port, and no service changes.

## The model of the book

The domain of PACA is the person, and the book of that person is the second
thing it carries: the **business** that owns the money and the **accounts**
that hold it.

### The business

A business is the thing that owns a book: the Personal Business of a person,
and the businesses of their own — the bakery of that same person.

**The identity of a business is generated by the database, and the document
ID of the owner is its owner and never its key.** The owner is found through
`(owner_id, kind = 'personal')`, and a person may own more than one
business: the key space is the businesses, never the people. The port of the
storage offers `ListByOwner`, which reads every business of a person in the
order of their identity.

**A person has exactly one personal business.** The domain states the rule
in `FindPersonalByOwner`, and the storage holds it as a defense in depth: the
partial unique index of the table refuses a second personal business of the
same owner, and it leaves the other kinds free.

The kinds are `personal` and `business`. The second one is in the vocabulary
of the domain, and the use case of the registration opens one: the name of a
business is taken inside the businesses of its owner, and the domain holds
that rule.

### The account

An account is a bucket of money inside the book of a business. The five kinds
are the five of the accounting, and the kind says what the account means: an
asset is what the person has, a liability is what the person owes, an income
is what the person earns, an expense is what the person spends, and an
equity is what the person put into the book of their own business.

**The name of an account is unique inside its business**, because the person
reads the accounts of their book by name.

**A debt is an account, and never a table of its own.** A loan with a friend
is a liability account with a counterparty and a due date, and the domain
refuses the pair of values that breaks that rule: an account of any other
kind carries neither.

**The domain owns the seeds of the books.** `DefaultPersonalAccounts`
returns the accounts that the Personal Business of a person starts with, and
`DefaultBusinessAccounts` returns the accounts that a business of the kind
`business` starts with: the cash, the bank, the inventory, the sales, the
cost of the sales, the labor, the overhead, the waste, the donation and the
capital. The
contract the agent reads is derived from them — the list of the accounts
and the kinds of an account are read from the domain and never written a
second time.

## The movements of the book

The `ledger` package holds the money of a person: the movements that fill the
book and the balance that comes out of it.

**Every movement is money that goes from the credit account to the debit
account.** The money lands where it is debited and it comes from where it is
credited. The amount is positive, the two sides are not the same account, and
the description is not empty: a movement the person cannot read is a
movement they cannot recognise.

**The amount is an integer and never a float.** The arithmetic of money does
not forgive a rounding error, and the person reads the numbers that the book
carries. It counts the units of the currency of the book, and the currency of
PACA is the Colombian peso.

**The day of a movement is a day and not a moment.** The time of the day is
not a fact of the money, and the column of the table holds no time.

**The balance of an account is the difference between its debits and its
credits, with the sign that the kind of the account asks for.** An asset and
an expense count as they are: what the person has and what they spend are
positive when they grow. A liability, an income and an equity count the
other way round: what the person owes, what they earn and what they put into
the book are positive when they grow. The result reads "what it means to
the person" in every kind.

**The money of the business is read from those balances:** what it has, what
it owes, what it earned, what it spent, and what it is worth.

### The period of the book

A reading of the book covers a **period**: the stretch of days that the
answer takes. `ledger.Period` holds the two ends of it, and it includes
both of them — the movement of the first day and the movement of the last
one are inside the period. `NewPeriod` refuses the ends that are in the
wrong order, and `Month` returns the period of a month of the calendar, with
the first day and the last day of the month that the calendar really has.

**The money of a period is what the person earned and what they spent.** The
two numbers of `ledger.Summary` are positive magnitudes — what came in and
what went out — and the net is what is left of the income once the expenses
are taken out of it. The net is negative when the person spent more than
they earned, and that is a fact of the month and not an error of the book.
The port of the book carries the reading of that money: `Summary` asks for
the money of a period and answers it.

### The debt

A debt is an account and never a table of its own. `account.IsDebt` says
when an account carries one: its kind owes or is owed, and it names the
other side of the money.

**The debt of the book is the account and what it still holds.**
`ledger.Debt` is the pair, and `ledger.Debts` reads the debts of a book out
of the accounts and their balances: the amount is the balance of the account
read the way the person reads it — what the person still owes for a
liability, what the person is still owed for an asset — and only the open
debts are in the answer.

**The debts come first by the day they fall due, and the debt without a day
comes last.** Two debts of the same day come by name, so the answer is the
same every time it is read.

**A payment may not pass what the debt holds.** `Debt.AcceptsPayment` asks
whether a payment may be paid against the debt: the payment must be
positive, and it may not be bigger than the amount the debt still holds.

## The model of the draft

`purchase.Draft` is the candidate that the agent proposes from what the
owner said. It is a proposal and not a fact: it is kept by the business
that owns it, with its state, its revision and the sentence it came from,
and it waits for the yes of the owner before anything is written in the
book. The transport of the MCP server is stateless, and a candidate that
lives only in the context of the model is a candidate that can be
paraphrased, lost or silently changed between the proposal and the yes.

**The draft carries the three parts of the answer of Paca.** What it
understood are its lines, what it still needs is `Missing`, what it was not
sure about is `Ambiguous` — and an ambiguity always carries the options
Paca offers, because Paca never asks an open question when it can name the
options — and what Paca filled by herself is `Inferred`.

**The money is never inferred.** `purchase.NewLine` refuses a price the
agent filled (`ErrInferredPrice`), and a line the owner did not price is
what the draft still needs. The quantity and its unit travel together:
`purchase.NewMeasure` refuses a quantity without a unit, and it reads the
unit the owner said it and keeps the unit the book knows. When the owner
named both the money that was paid and the price of the unit, the two must
agree, and the domain refuses the pair that does not.

**The day is one of the three things Paca may fill by herself.** A draft
that names no day is the day of today, and the answer names it in
`Inferred`. The day, the currency and the business are the three, and the
money is never one of them.

**A draft expires seven days after the day it was proposed.** The price of
a market does not wait. An amendment grows the revision of the draft and
keeps the sentence the owner said the first time.

**A draft never returns to `proposed`.** The state machine of the draft is
in the domain: a draft starts in `proposed` and it ends in `confirmed`,
`cancelled` or `expired`, and `purchase.States` is the vocabulary of it.
`purchase.Cancel` drops the draft the owner does not want and refuses the
draft that does not wait any more (`ErrNotCancellable`), and the amendment
of a draft that ended is refused (`ErrNotProposed`).

**The expiry is computed when the draft is read.** `purchase.Draft.ReadsAs`
answers the state a draft reads as on a day: a draft that was neither
confirmed nor cancelled seven days after the day it was proposed reads as
`expired`, and the storage needs no job to expire anything.
`purchase.Pending` answers the drafts of a business that still wait for the
yes of the owner.

## The model of the purchase

`purchase.Purchase` is the fact of what the business bought: the business,
the day, the market, the invoice, its lines, the sentence of the owner and
the word she used to confirm it. A purchase never changes.

**The confirmation is a rule of the domain.** `purchase.Confirm` turns a
draft into the purchase it describes, and it refuses what the domain
forbids: a confirmation without the word of the owner
(`ErrMissingConfirmation`), a draft that does not wait any more
(`ErrNotProposed`), a draft that changed under the eyes of the owner
(`ErrStaleRevision`), a draft that expired (`ErrExpiredDraft`) and a draft
that still needs the money of one of its lines (`ErrStillNeeds`).

**The money of the book counts whole pesos.** `purchase.Price.Amount`
answers the money of a line: the money that was paid, and the price of the
unit times the quantity rounded to the nearest peso when the owner named
only the price of the unit. The sentence of the owner keeps the exact
wording, and the money of the book is whole pesos.

**The confirmation answers with the three parts of Paca.** What it
recorded are the lines of the purchase, what it did not understand are the
ambiguities the draft still carried, and what it still needs is empty: a
purchase without the money of its lines is refused.

**The money of the payment moves from the account that pays to the
inventory.** `purchase.ThePayingAccount` answers the account the money
comes from — the cash of the business, and the account the owner names
when she names another one — and `purchase.TheInventoryAccount` answers the
account of the inventory of the business. The role of an account belongs to
the seed of the book of a business: `account.OfRole` answers the account of
a role without writing its name down twice.

**A draft is confirmed once.** The state of a draft moves from `proposed`
to `confirmed`, and the storage refuses the confirmation of a draft that is
not the one the owner saw: the revision she saw is the revision the row
carries.

## The model of the invoice

`purchase.Invoice` is the fact that comes with the purchase: the provider
that issued it, its number, its day, its base, its rate, its tax, its total
and the purchase it came from.

**The invoice moves NO money.** The purchase already moved the money of the
payment when the owner bought, and the invoice arrives later. It is a fact
that is written against the purchase it came from.

**The tax of an invoice is NEVER ASSUMED: it comes from the invoice.** The
invoice states the rate it was issued with and the base that rate applies
to, and the book derives the tax from the two of them (`purchase.TheTax`).
There is no default of nineteen percent and no default of zero, and the
rate is never taken from the catalogue of the country: the rule of the tax
authority and the advice of the professional of the owner are not read
here.

**The rounding rule of the book.** The tax is rounded to the nearest whole
peso, and a half peso goes UP: a base of fifty pesos at nineteen percent is
nine pesos and a half, and the tax of that invoice is ten pesos. The rate
is counted in hundredths of a percent (`purchase.Rate`), because the rate of
the tax authority of the owner may not be a whole percent, and a rate that
is rounded away is a rate nobody charged. A rate with MORE THAN TWO
DECIMALS is refused (`ErrInvalidRate`): the rate of an invoice is the rate
the invoice states, or nothing, and the book never rounds one away. The
rate of an invoice is also a rate when it reaches the domain whole: a rate
outside of the hundred percent is refused by the invoice itself
(`ErrInvalidRate`).

**The money of an invoice enters the book in whole pesos, and the book never
rounds it.** The money of the book counts the pesos the person has, and the
book takes the money the invoice states. The rounding of the CENTAVOS of an
invoice is NOT decided here: it is an OPEN QUESTION for the owner and for
the professional of the taxes. The question stays open until the owner and
the professional of the taxes answer it, and PACA never fixes a total.

**An invoice that does not add up is never read as an invoice.** The total
of an invoice is the base plus the tax, and the domain refuses the invoice
that breaks the rule (`ErrInvoiceDoesNotAddUp`).

**The provider of an invoice is a PARTY.** The party holds the role of the
provider, and a party that holds only the role of the client is refused
(`ErrNotAProvider`). The provider is a party of the business that bought,
and the invoice of one business is never attached to the purchase of
another one (`ErrInvoiceOfAnotherBusiness`).

**The purchase points at the invoice it came from.** The invoice names the
purchase (`ErrMissingPurchase`), the purchase carries the identity of the
invoice, and the two belong to the same business. The invoice is also
attached ONLY to the purchase it names (`ErrInvoiceOfAnotherPurchase`): the
two sides of the link must agree, and an invoice of the purchase 2 attached
to the purchase 3 would leave the book with two truths about the same
purchase.

**A purchase carries ONE invoice.** The invoice that came with the purchase
is the one the purchase points at, and the second invoice of the same
purchase is refused (`ErrPurchaseAlreadyHasInvoice`). The table holds the
same rule as a defense in depth: the unique index of the purchase.

**The word of the invoice and the fact of it are two different facts.** The
purchase carries the WORD the owner said at the time of the purchase
(`Purchase.Invoice`) and the FACT of the link to the invoice
(`Purchase.InvoiceID`). The owner may read "FV-001" while the invoice the
purchase points at carries another number, and the book NEVER COMPARES
THEM: the word is what she said and the fact is where the invoice is.

**The number of an invoice is unique inside the business that holds it**
(`ErrDuplicateInvoiceNumber`): the number is what identifies the invoice,
and the table holds the same rule as a defense in depth.

**The electronic invoice is NEVER assumed.** There is no integration with
the tax authority and no field that pretends to be one: the reference of the
invoice is a plain fact of the document the owner shows, and it may be
absent.

## The stock and the layers of the FIFO

`purchase.Layer` is a layer of the stock: the ingredient it is of, the
quantity that was bought with its unit, the money it cost, the day it was
bought and the purchase it comes from.

**The purchase is the fact and the layer is derived from it.**
`purchase.LayersOf` derives the layers of a purchase from its lines, and a
line the catalog does not know has no layer (`ErrLayerWithoutIngredient`):
the layer would name nothing at all. Nobody types a layer, and the layer
never becomes a second truth about what was bought.

**The stock of an ingredient is the sum of the layers that are still
open.** `purchase.StockOf` answers what the business holds: the quantity it
holds, the money it cost and the layers that are still open. A layer that is
closed — nothing is left of it — is not in the answer.

**The valuation is FIFO.** The layers of every holding are read in the
order they leave the stock: the oldest layer is the first one that leaves.
Two layers bought the same day are read by the identity of their purchase,
so the answer is the same every time it is read. The order of the output by
expiry is a different rule, and it does not live here.

**The quantity and its unit travel together.** The layers of an ingredient
that were bought in different units are held apart: the book never adds a
gram to a kilogram.

**The layer carries the unit the book knows.** The layer is derived, and the
unit it carries is the one of the book (`g`, `kg`, `paquete`, `unidad`): the
purchase stores the unit of the book too, and the layer of that purchase is
in kilograms. A line whose unit is not one of the units of
the book has no layer, and a layer that carries another unit is refused
(`ErrInvalidUnit`, `ingredient.ErrUnknownUnit`).

**A layer that is not whole is refused.** `purchase.Layer.Checked` refuses
the layer that names nothing of the catalog, that bought no quantity, that
carries a unit the book does not know, that holds more than it bought, that
comes from no line of its purchase and whose money is negative
(`ErrInvalidLayer`, `ErrInvalidPosition`). The storage reads the layer
through it, and a row that another writer broke is never read as a layer.

**A layer belongs to the business of its purchase.** The layer carries the
business of the purchase it comes from, and a layer that claims another
business is refused (`ErrLayerOfAnotherBusiness`). The stock of a business
never holds the layer of a purchase of another one.

**A layer comes from one line of its purchase.** The position of the layer
is the position of the line it comes from: it is the order the owner said
the line in, and it is unique inside the purchase.

**The stock belongs to a business.** `purchase.TheBusinessOfTheStock` answers
the business whose stock is read, and it refuses the business that is not
there (`ErrMissingBusiness`): there is no stock of nobody.

## The catalog of the names

`ingredient.Ingredient` is the thing the business buys: the name the owner
reads, the names she uses for it — its aliases — and the business that owns
it. `ingredient.Catalog` is the set of the ingredients of a business.

**The catalog begins empty and grows with what the owner buys.** The first
purchase creates the first name, and a name the catalog does not know is
not an error: it is recorded as the owner said it, and the answer tells
her it is new.

**The matching is a rule of the domain.** `ingredient.Catalog.Match`
answers the name the owner said: `exact` when one ingredient answers it,
`ambiguous` when several answer it — with the options it offers — and
`unknown` when none of them does. The options are in the order of the
identity of the ingredients, so the answer is the same every time.

**The name is compared the way a person reads it.** Trimmed, and without
case: `Harina` and `harina ` are the same name, and the alias `harina` is
the same ingredient as the name `Harina panadera`.

**Two ingredients of the same business never carry the same name.**
`ingredient.NewCatalog` refuses the catalog that holds two of them
(`ErrDuplicateName`), and the table holds the same rule as a defense in
depth. An ingredient of another business is not in this catalog
(`ErrWrongBusiness`).

**The line of a purchase carries the ingredient it names.**
`purchase.Line.WithIngredient` answers the line with the identity of the
ingredient of the catalog, and the identity is empty when the catalog does
not know the name: the line keeps the name as the owner said it. The
catalog grows with the confirmation of a purchase: a name the catalog does
not know becomes an ingredient of the business.

**The unit of a line is the unit of the ingredient.** The quantity and its
unit travel together, and the unit of a line the owner did not name is the
unit of the ingredient of the catalog: the use case asks the catalog and
fills it, and the domain still refuses a line that ends without a unit
(`ErrMissingUnit`).

## The units and the completion of an ingredient

`ingredient.Unit` is the unit an ingredient is bought in. The units of the
book are the four the bakery knows: the gram (`g`), the kilogram (`kg`),
the package (`paquete`) and the piece (`unidad`). The vocabulary belongs
to the domain (`ingredient.Units`), and the table holds the same rule as a
defense in depth.

**The conversion of a unit into the gram is declared, never assumed.** The
gram is one and the kilogram is a thousand (`ingredient.GramsPerKilogram`).
The package is the weight the owner declares
(`ErrMissingPackageWeight` when she declared none), and the piece is one
unit of the ingredient: its weight is never assumed, and it is never
converted into grams (`ErrUnknownConversion`).

**What the owner declares of an ingredient is a declaration.**
`ingredient.Declaration` carries the unit, the weight of a package, the
allergens, the certification with its evidence, the aliases and the
conversion, and `ingredient.Complete` applies it to the ingredient of the
catalog: it fills what the owner declared and keeps the rest. A field the
owner did not declare is not a field the ingredient loses.

**"Organic" is a claim that carries its evidence.**
`ingredient.NewCertification` refuses a certification without the document
that proves it (`ErrMissingEvidence`), and the two fats are two ingredients
with their own certification: the domain never merges them.

**The conversion of the ingredient is declared, never assumed.** The owner
decided that, for now, the grams of a recipe are the raw ones: the
conversion is one to one (`ingredient.RawConversion`), and the mechanism
carries any other factor she declares later (`ErrInvalidConversion` when
the factor is not positive).

**What is missing of an ingredient is a rule of the domain.**
`ingredient.Missing` answers what the owner has not declared yet: the unit
the ingredient is bought in, and the weight of a package when the unit is
the package. The alias of an ingredient is written by the completion: the
names the owner uses for it.

## The model of the recipe

`recipe.Recipe` is the guide of the bakery: what it makes and how. It
carries the name the owner reads, the ingredients with their grams, the
steps with their times, their temperatures and their cues, the yield of what
it makes and the shelf life of it.

**The recipe is not a list of ingredients.** It is a guide to learn and to
teach, and the steps are part of it: a recipe without steps teaches nothing,
and `recipe.New` refuses it (`ErrNoSteps`). A step without its instruction
is not a step either (`ErrMissingInstruction`).

**The ingredient of a recipe is the ingredient of the catalog.**
`recipe.Ingredient` carries the identity of the ingredient and the grams of
it, and nothing else: the recipe never carries a second copy of the flour.
An ingredient of the catalog that the business does not hold is refused
(`ErrUnknownIngredient`).

**The yield is a declared constant.** `recipe.Yield` is the number of units
the recipe makes and the weight of each one, and it is positive
(`ErrInvalidYield`). It is never measured per bake: the production of a day
is a fact of its own.

**The steps keep their order, and a step may be named by its cue.** The cue
is the sign the owner looks for — the dough that holds the shape. A step
carries its time and its temperature when the owner named them, and neither
when she did not: the zero of the time and of the temperature is the
absence of them, because a number PACA invented is a lie in the guide. A
temperature below the absolute zero is not a temperature
(`ErrInvalidTemperature`).

**The recipe is written with the word of the owner.** The name is not empty
(`ErrMissingName`), the shelf life of what it makes is positive
(`ErrMissingShelfLife`), and the word she used to confirm it is kept with
the record (`ErrMissingConfirmation`). The sentence of the owner is
optional: the recipe is the record, and the sentence is kept with it when
she said one.

## The cost of a batch

`recipe.Cost` is what a batch of a recipe costs today: the direct cost, the
money of the layers of the stock that the recipe consumes. It is **derived,
never typed**: `recipe.CostOf` reads the layers of the stock and takes what
the recipe needs, and the price of the owner never reaches it.

**The recipe consumes the layers in the order of the FIFO.** The oldest
layer leaves first, and `purchase.Consume` is where that order lives: the
consumption of a layer is taken, and the layer that is left keeps its place
in the queue.

**The money of what is left of a layer is the money of what it bought, pro
rata the quantity that is left.** `purchase.Layer.MoneyLeft` states the
rule, and the book rounds it to the nearest peso because the money of the
book counts whole pesos. The stock carries the money of what it still holds,
never the money of what it bought.

**The layer is tied to the line of its purchase.** `purchase.Layer.CheckedAgainstTheLine`
re-derives the layer from the line and refuses it when the money, the
quantity or the ingredient differ (`ErrLayerNotTied`): a row another writer
broke is never read as the stock of the business.

**The cost answers per unit, per gram and per kilogram**, and it says what
it does not know instead of guessing: an ingredient the stock holds in
pieces or in a package without a declared weight has no rate per gram, and
the answer names it.

**The cost is direct.** The overhead (T14) is carried beside it, in
`recipe.Cost.Overhead`, and the labour of the batch is carried beside it
too, in `recipe.Cost.Labor`. The three layers are never added into one
number: the report of the real price shows them apart.

## The labour of the bakery

`recipe.Labor` is the labour of the owner, as she declared it: the hours she
worked, in the three kinds of the labour — the hands, the oven and the
waiting — the value of the hour, the day it was worked and the word she used
to confirm it. It lives in the package of the recipe, next to the cost of
the batch it is a layer of.

**The labour is measured in hours, in three kinds.** `recipe.Hours` holds
the hours of the hands, of the oven and of the waiting apart, and
`recipe.NewHours` refuses the hours that are negative
(`ErrInvalidHours`): a negative number of hours is a number nobody worked.
The kinds are the vocabulary of the domain: `recipe.LaborKinds` names the
three of them and `recipe.ParseLaborKind` reads the way the owner says them
(`hands_on`, `oven`, `waiting`), and a kind the book does not know is
refused (`ErrUnknownLaborKind`).

**The value of the hour is declared by the owner.** It is positive
(`ErrMissingHourValue`), and the money of the work of a person is never a
number PACA invents. The money of the labour is the hours of it times the
value of the hour, and the book rounds it to the nearest peso because the
money of the book counts whole pesos.

**The labour of a batch is derived from the hours of its steps.**
`recipe.LaborOf` adds the time of every step of the recipe to the kind of
labour the step is, and values it with the value of the hour. A step the
owner did not qualify is a step she did with her hands, and the answer that
does not know the value of the hour says so instead of guessing it.

**The labour is of the owner alone.** The payroll of the employees is out of
scope for now, and the model of the labour of an employee does not exist
yet: `recipe.Labor` names no employee, because it has none.

## The overhead of the house

`recipe.Overhead` is the overhead of the house of the bakery, as the owner
declared it: the electricity the house used, the hours the oven ran, the
allocation she declared, the day it was declared and the word she used to
confirm it. It lives in the package of the recipe, next to the cost of the
batch it is a layer of.

**The overhead is the electricity of the house and the hours of the oven.**
The electricity is positive (`ErrMissingElectricity`) and it is declared by
the owner: the money of the house is not a number PACA invents. The hours
of the oven are positive (`ErrMissingOvenHours`): without them the money of
the oven has no rate, and a rate nobody declared is a rate PACA invented.

**The allocation is declared by the owner.** It is the share of the
electricity of the house that belongs to the oven, and it is a share: it is
greater than zero and not greater than one (`ErrInvalidAllocation`). The
money of the overhead is the electricity of the house times the allocation,
and the book rounds it to the nearest peso because the money of the book
counts whole pesos.

**The overhead of a batch is derived, never typed.** `recipe.OverheadOf`
reads the steps of the recipe and counts the hours of the oven of it — the
steps the owner said are the oven — and values them with the rate of the
house. The steps of the hands and of the waiting are the labour of the
batch, not the overhead of it.

**The rate of the house is derived from what the owner declared.**
`recipe.TheOverheadRateOf` takes the last overhead the business declared —
the rate of a house does not wait — and answers the money of one hour of
the oven. A business that declared none has no rate, and the batch says
that it does not know it: the money of the house is never inferred.

**The overhead is a fact of the business.** It moves no money of the ledger
yet: the account of the overhead exists in the seed of the business, and
the report of the real price writes none — a report reads and writes
nothing. The decision of whether the overhead writes a movement stays open,
and it belongs to the increment that writes the movement.

## The report of the real price

`recipe.RealPrice` is the report the owner reads when she asks what her
product really costs: the three layers of the cost of a batch — the direct
cost, the labour and the overhead — per unit, per gram and per kilogram,
and the price she charges beside them. It lives in the package of the
recipe, next to the cost of the batch it is a report of.

**The three layers travel apart, and so does their sum.** `recipe.RealPriceOf`
builds the report out of the cost of the batch: every layer keeps its own
money and its own rates in `recipe.Rate`, and `RealPrice.Total` carries the
sum of the three beside them. No layer of the cost is ever hidden inside
another one, and the sum is a number of its own, not a layer.

**Every layer answers per unit, per gram and per kilogram.** The bakery
sells by the unit and by the weight, and a comparison between two products
that knows only one of them is a lie.

**The price of the owner is hers, and the book never moves it.**
`purchase.OwnerPrice` is the money she charges with the way she charges it:
`purchase.PricePerUnit`, `purchase.PricePerGram` or
`purchase.PricePerKilogram`. The money is positive
(`ErrInvalidOwnerPrice`) and the way she charges is one of the three the
book knows (`ErrUnknownPriceBasis`); `purchase.ParsePriceBasis` reads the
way a person says it (`unidad`, `gramos`, `kilos`), and
`purchase.PriceBases` names the three for the description of the tool. The
report carries the money she declared and the rates it weighs against, and
the price of the unit and the price of the gram are two different truths.

**The money of what she sells is never inferred.** A report of a price
nobody declared says that it is not known, and the three layers of the cost
still answer.

**The sum is the sum of a batch that can be baked.** A layer that is not
known — the labour of a business that declared no value of the hour — or a
stock that does not hold everything the recipe needs leaves the total
unknown, and the answer says why.

## The margin of a product

`recipe.Margin` is the margin the owner reads when she asks what her
product earns: what her price leaves above the direct cost of the batch,
and what the market said about the same ingredients. It lives in the
package of the recipe, next to the cost of the batch and the report of the
real price it is built upon: the margin needs the yield of the recipe and
the cost of the batch, and the direction of the dependencies forbids the
package of the purchase from reaching for them. The age of a reference is
the rule of the purchase (`purchase.Reference.AgeOn` and
`purchase.Reference.IsOlderThan`), because a reference is a price fact of
the business.

**The margin compares the price of the owner with the direct cost, and
with the reference of the market.** `recipe.MarginOf` builds the margin
out of the cost of the batch, the price of the owner, the references of the
market and the catalog of the names. `Margin.Profit` is what the price
leaves above the direct cost — per unit, per gram and per kilogram — and
`Margin.Percent` is that profit over the direct cost. `Margin.Market` is
what the market said: the money it says the ingredients of the batch cost,
the money the owner paid for them, the difference between the two, and one
line for every ingredient. `Margin.MarketProfit` is what the price leaves
above the market, and it is a number of its own: neither profit hides the
other.

**A reference is a reference, and it never moves the money of the book.**
The direct cost comes from the layers of the stock, and the reference is
read beside it: what the owner paid and what the market says are two
truths that never mix.

**Every reference carries its source and its day, and its age.** The
newest reference of an ingredient is the one that compares — the market of
today is the last one that was seen — and the older ones travel with it as
the history they are. `recipe.ReferenceExpectation` carries what the owner
expects: a reference older than she expects is shown with its age and
marked as old, and every reference shows its age anyway.

**Only the gram and the kilogram are converted into the grams of the
recipe.** The package is converted only when the owner declared the weight
of it (`ingredient.PackageGrams`), and the piece is never converted: the
weight of a piece is not known until she says it.

**An answer that does not know says so.** The profit is unknown when the
price of the owner is unknown, and when the stock does not hold everything
the recipe needs. The money of the market is unknown when one of the
ingredients has no reference, when the reference is in a unit the book
never converts and when the weight of the package is not declared: a sum
that hides a missing part is a lie, and the answer says what is missing.

## The model of the production

`recipe.Production` is what the bakery made on a day: the recipe it was
made of, the day it was made, the grams that went in, the money of the
layers it consumed and the units that came out. It lives in the package of
the recipe, next to the recipe it is made of: the recipe declares the yield
of the units and the shelf life of them, and the production is the day
those units came out.

**The production is derived, never typed.** `recipe.ProductionOf` reads the
recipe, the layers of the stock and the catalog of the names, and it builds
the production out of them. The grams that went in are the grams the
layers gave, and the units that came out are the yield of the recipe: the
book never asks the owner for a number it can read.

**The production consumes the layers in the order of the FIFO.** The oldest
layer leaves first, and `purchase.Consume` is where that order lives. The
production carries the layers it consumed in `Takes`, in the order they
left the stock, and what is left of each of them in `Remaining`.

**The quantity that is left of a layer is the quantity it held minus what
the production took from it, and the unit travels with it.**
`purchase.Layer.Consumed` states the rule and refuses a take whose unit is
not the unit of the layer (`ErrInvalidUnit`): the book never subtracts a
gram from a kilogram. The storage writes what is left of the layer with the
unit of the layer, and it writes it only when the layer still holds what
the production read.

**The money of the production is the money of the layers it consumed, pro
rata the quantity that was taken.** The money of the book counts whole
pesos, and it is never inferred: the money of what was taken is the money
of what the layer held minus the money of what is left of it.

**Every unit carries the day it expires.** The day of the expiry is the day
of the production plus the shelf life of the recipe, and it is approximate
until the owner corrects it. A production that made no unit is not a
production (`ErrMissingProductionUnits`), and a unit that expires before
the production is refused.

**The production is written with the word of the owner.** The day is there
(`ErrMissingProductionDay`), the recipe is there (`ErrMissingRecipe`), and
the word she used to confirm it is kept with the record
(`ErrMissingProductionConfirmation`). The sentence of the owner is
optional, and it is kept verbatim when she said one.

**A production the stock cannot cover is refused.** The stock that does not
hold what the recipe needs is refused (`ErrMissingStock`), and so is the
stock that holds an ingredient in a unit the book cannot weigh
(`ErrUnknownStock`): the weight of a piece is never assumed.

**A production names ONE recipe.** The name of a recipe is not unique
inside the business, and `recipe.TheRecipeNamed` refuses a name that more
than one recipe carries (`ErrAmbiguousRecipe`): the book never guesses
which recipe she made. A name that no recipe of the business carries is
refused too (`ErrUnknownRecipe`).

## The model of the expiry

`recipe.ShelfLifeCorrection` is the correction the owner declared of the
shelf life of a recipe: the recipe she corrected, the number of days what
it makes keeps, the day she corrected it and the word she used to confirm
it. `recipe.ExpiringUnit` is a unit of the book read for the day it
expires: the unit of the production, with the production it came out of
and the recipe it was made of.

**The shelf life is approximate until the owner corrects it.** The recipe
declares one, and the correction is the truth she gave later: the number
of days is what she said, and the book never invents it.

**The correction is KEPT.** Every correction is a row of its own, and
`recipe.ShelfLifeOf` answers the shelf life that counts: the last
correction the owner declared, and the declared one of the recipe when she
corrected nothing. The corrections of another recipe are not hers: the
shelf life of a bread does not move the expiry of the cake.

**The correction moves the units that were already made.**
`recipe.CorrectedUnits` answers the units of the recipe — and of no other
recipe — with the day they expire on now: the day of the production plus
the days she corrected. The storage writes that day only when the unit
still expires on the day the correction read: a unit that changed under
the correction is refused.

**Every unit carries the day it expires, and the unit keeps the days of
its shelf life.** The shelf life of three days is the day of the
production plus three days, and the unit expires on the first day it is no
longer good: `recipe.ProductionUnit.Expired` says it, and
`recipe.ProductionUnit.DaysLeft` counts the days that are left of it.

**A unit that expired is never sold.** `recipe.ProductionUnit.CheckSellable`
refuses it (`ErrExpiredUnit`), and a sale of no day is refused too
(`ErrMissingSaleDay`): the day is what says whether the unit expired.

**The answer of the expiry is ordered by the day the units expire.**
`recipe.ExpiringUnits` orders them, and the unit that expires first is
the first one. The rotation — the order in which the units leave the book
— is another rule, and it lives in its own section.

## The model of the rotation

`recipe.Rotation` is the order in which the units of the book leave it: the
day the answer was read on, the product it is of, and the units in the
order they leave. `recipe.RotationUnit` is one unit of that order: the unit
of the production, the product it was made of, the day it expires and the
days that are left of it.

**The rotation follows the earliest expiry.** `recipe.RotationOf` builds the
answer, and the unit that expires first is the first one that leaves the
book. The units that expire on the same day keep a stable order: the
production first, and the position of the unit in it. Every unit carries the
place it takes in that order: the first one that leaves is the first one.

**The unit that leaves a production is the one that expires first.** The
waste, the sale and the exit take the units of the product in that order,
and the rule lives in `recipe/rotation.go`: its holders are
`theUnitsOfTheProduct`, that answers the units of the product that are
still in the book, and `orderedByTheRotation`, that answers them in the
order they leave it. `recipe.RotationOf` builds the answer of the rotation
with them, and it holds no other order.

**The rotation follows the earliest expiry and the valuation follows the
FIFO: the two rules are apart.** The FIFO is the rule of the layers of the
stock, and the rotation is the rule of the output. The answer of the
rotation carries **no money at all**: the money of a unit is the money of
the production it came out of, and it belongs to the movement that takes
the unit out of the book.

**The rotation may be of the whole book or of one product.** The zero
identity is the whole book of the business, and the identity of a product
is that product alone.

**The rotation holds only the units that are still in the book.** The
caller reads them through the reader of the book, and a unit that left it
— the one that was sold, wasted, consumed or given away — is never offered
again.

## The model of the waste

`recipe.Waste` is what the bakery lost: the product, the quantity, the unit
and the reason, the day it happened, the word of the owner and the money of
what it was worth. `recipe.WastedUnit` is a unit of a production that left
the book. It lives in the package of the recipe, next to the production it
is a waste of: the waste is of a product, and the product is a recipe.

**The waste is a movement of the book.** It carries the product, the
quantity, the unit and the reason — burnt, spoiled, dropped or returned —
and it is written with the confirmation of the owner, the way the
production is written (`ErrMissingWasteConfirmation`). The sentence of the
owner is optional, and it is kept verbatim when she said one.

**The reason is what a later analysis reads, and it is never guessed.** The
reasons are the four of the book, and `recipe.ParseWasteReason` reads the
name the owner says the way a person reads it: a reason the book does not
know is refused (`ErrUnknownWasteReason`).

**The waste counts the units it lost.** The quantity is a whole number of
units (`ErrInvalidWasteQuantity`), and the unit is the unit of a piece
(`ErrInvalidWasteUnit`): the book never converts a weight into a number of
units, and it never splits a unit. When the owner names a weight, the agent
asks her how many units she lost.

**The units that leave the book are the units that expire first.** The
rotation of the output follows the earliest expiry, and the valuation of
the stock by the FIFO is another rule: the model keeps the two apart. A
waste of units the book does not hold is refused (`ErrNotEnoughUnits`).

**The waste moves no money IN and releases the cost of what it was
worth.** Nobody paid for the bread the bakery lost, and what the waste does
is release the cost of the units that left the book. The money of the waste
is the money of the units that left: the money of the production they came
out of, divided by the units that production made. The money is derived,
never typed, and it counts whole pesos.

**The money of the waste is never lost from the book.**
`recipe.Waste.Movement` moves it from the inventory to the account of the
waste — the account the book of a business opens for the product that
never reached a client. The description of the movement is the sentence of
the owner, and the words of the book when she said none.

## The model of the sale

`recipe.Sale` is what the bakery sold: the product, the quantity, the
unit, the money the client paid, the day it happened, the word of the
owner and the cost the sale released. `recipe.SoldUnit` is a unit of a
production that left the book. It lives in the package of the recipe, next
to the production it is a sale of: the sale is of a product, and the
product is a recipe.

**The sale is a movement of the book.** It carries the product, the
quantity, the unit and the money the client paid, and it is written with
the confirmation of the owner, the way the production and the waste are
written (`ErrMissingSaleConfirmation`). The sentence of the owner is
optional, and it is kept verbatim when she said one.

**The money of the sale is what the client paid, and it is never
inferred.** It is positive (`ErrInvalidSaleMoney`) and it counts whole
pesos: the price list of the products is a later work, and the book never
guesses the money of a sale.

**The sale counts the units it sold.** The quantity is a whole number of
units (`ErrInvalidSaleQuantity`), and the unit is the unit of a piece
(`ErrInvalidSaleUnit`): the book never converts a weight into a number of
units, and it never splits a unit.

**The units that leave the book are the units that expire first.** The
rotation of the output follows the earliest expiry, and the valuation of
the stock by the FIFO is another rule: the model keeps the two apart. A
sale of units the book does not hold is refused (`ErrNotEnoughUnits`).

**A unit that expired is never sold.** The sale is refused
(`ErrExpiredUnit`), and it does not skip the expired unit to take a
fresher one: the expired unit leaves the book as a waste, and the book
says so.

**The sale releases the cost of what it sold.** The cost of the sale is
the money of the units that left: the money of the production they came
out of, divided by the units that production made. The FIFO of the layers
was already taken when the production consumed them, so the money of the
production is already FIFO-valued, and the sale reads that money: it never
recomputes a valuation of its own. The cost is derived, never typed, and
it counts whole pesos (`ErrInvalidSaleCost`).

**The sale moves the money in and it releases the cost.**
`recipe.Sale.MoneyMovement` moves the money of the sale from the account
of the sales to the account that receives it — the cash of the business, or
the one the owner names (`recipe.TheReceivingAccount`) — and
`recipe.Sale.CostMovement` moves the cost of the units that left the book
from the inventory to the account of the cost of the sales. The money of
the sale is never lost from the book, and neither is the cost of it.

## The model of the exit

`recipe.Exit` is what left the book without money in it: the consumption of
the owner and the donation. `recipe.ExitedUnit` is a unit of a production
that left the book. It lives in the package of the recipe, next to the
production it is an exit of: the exit is of a product, and the product is
a recipe.

**The exit is a movement of the book, and it moves no money IN.** Nobody
paid for the bread the owner consumed or for the bread the bakery gave
away, and what the exit does is release the cost of the units that left the
book. It carries
the kind of the exit — the consumption of the owner or the donation — the
product, the quantity, the unit, the day it happened and the word of the
owner, and it is written with her confirmation, the way the production, the
waste and the sale are written (`ErrMissingExitConfirmation`). The kind is
never guessed (`ErrUnknownExitKind`), and `recipe.ParseExitKind` reads the
name the owner says the way a person reads it.

**The exit counts the units it took.** The quantity is a whole number of
units (`ErrInvalidExitQuantity`), and the unit is the unit of a piece
(`ErrInvalidExitUnit`): the book never converts a weight into a number of
units, and it never splits a unit.

**The units that leave the book are the units that expire first.** The
rotation of the output follows the earliest expiry, and the valuation of
the stock by the FIFO is another rule: the model keeps the two apart. An
exit of units the book does not hold is refused (`ErrNotEnoughUnitsToExit`).
The sale is the only exit that refuses the unit that expired: the bread
that expired is still bread for the owner and for the one she gives it to.

**The exit releases the cost of what it took.** The cost of the exit is the
money of the units that left: the money of the production they came out
of, divided by the units that production made. The derivation is the one of
the waste and of the sale, and it is never written a second time. The cost
is derived, never typed, and it counts whole pesos (`ErrInvalidExitCost`).

**The exit moves no money IN and releases the cost.**
`recipe.Exit.CostMovement` moves the cost of the units that left the book
from the inventory to the account of the expense of the exit, and it is the
only movement of the exit: nobody paid for the bread the owner consumed or
for the bread the bakery gave away. The account is the one the domain
names: `recipe.Exit.TheExpenseRole` answers the cost of the sales for the
consumption of the owner — the consumption releases the cost like a sale
does — and the account of the donation for the donation, which is a social
expense of the business (`account.RoleDonation`). The description of the
movement is the sentence of the owner, and the words of the book when she
said none (`recipe.Exit.Description`).

**The tax treatment of the exit waits for the professional of the taxes.**
The book decides no tax treatment of its own: the exit carries the
treatment it waits for (`recipe.TaxPendingTheProfessional`) and the question
that waits with it, and an exit that carries none is refused
(`ErrMissingTaxTreatment`). There is no rate that the book assumes and no
default that it hides.

## The model of the reference price

`purchase.Reference` is what the market was seen at: the ingredient it is
about, the price that was seen, the unit of that price, the market where
it was seen, the place it was read and the day it was read. It lives in
the package of the purchase, next to the price it is a reference of: it is
a price fact of the business, and not a field of the ingredient.

**A reference is never the price of the ingredient.** The price the owner
pays exists only in the market of her town and only in her mouth on the
day: it comes from her own purchases. The reference is what Paca
researched, and the domain labels it as one — `purchase.ReferenceKind` and
`purchase.Reference.Label` — so that no later analysis may read it as the
price of the ingredient.

**The reference is whole.** It names an ingredient of the catalog
(`ErrMissingIngredient`), its price is positive
(`ErrInvalidReferencePrice`), the price carries its unit
(`ErrMissingReferenceUnit`), it carries the source where the price was
read (`ErrMissingSource`) and it carries the day it was seen
(`ErrMissingDay`). The source is never empty: a price without a source is
a price nobody can check.

**The reference is written with the word of the owner.** It is confirmed
the way the purchase is confirmed, and the word she used is kept with the
record (`ErrMissingConfirmation`). The market is optional: it is there when
the research named one, and absent when it named none.

**The age of a reference is a rule of the domain.** `Reference.AgeOn` counts
the days between the day the price was seen and the day of the reading, and
`Reference.IsOlderThan` says whether it is older than what the owner
expects. A reference older than she expects is shown with its age, and the
age is never negative: a reference of the day of the reading is not old.

## The model of the party

`party.Party` is the other side of the bread: the person, the store or the
farm that buys from the bakery, or that the bakery buys from. The three are
one model, because the owner buys from a farm and sells to a person, and
neither of them is a second business.

**The kinds of a party are the three of the owner.** `party.Kind` names the
person, the store and the farm, and the domain reads them from the name the
storage holds (`ParseKind`). The kind decides what the party carries.

**The legal human that represents a store or a farm is a FACT OF THE PARTY.**
It is a name, and the legal figure it has when it has one. It is never a
second person: it is not a `user`, it is not a party of its own, and it
never becomes a row of the people of PACA. The rules are the ones of the
kind: a store **requires** the legal human that represents it
(`ErrMissingLegalHuman`), a farm **carries** it only when it has one — some
farms have a legal figure and some do not — and a person **refuses** it
(`ErrUnexpectedLegalHuman`).

**The domain builds the legal human from the raw facts.**
`party.LegalHumanOf` takes the kind of the party, the name of the human and
the legal figure, and answers the legal human that the party of that kind
carries: the person refuses one, the store requires one, and the farm
carries one only when it has one. A legal figure without the name of the
human it belongs to is a figure of nobody, and the domain refuses it. The
use case hands the raw facts to the domain and never decides.

**The roles of the party are the client and the provider.** A party holds
the role of the client, the role of the provider, or both of them, and the
domain refuses the party that holds none of them (`ErrNoRole`): the bakery
neither sells to it nor buys from it, and it is not a party of the book. A
role that is said twice is one role.

**The way of reaching the party and its address are facts of the party.**
The rules of the phone and of the email are the rules of the person
(`user.CleanPhone` and `user.CleanEmail`), and the package reuses them: the
party never writes a second cleaning rule.

**The party belongs to a business of its own.** `party.OwnedBy` states the
rule: the parties belong to a business of kind `business`, and the Personal
Business of a person holds the money of that person and not the clients and
the providers of the bakery (`ErrWrongBusiness`). `party.BusinessNotFound`
states the other half of the rule: a business of a party that is not there
is refused with `ErrNotFound` — the error of the party — and never with the
error of the personal business of a person, because the caller of the party
is not the onboarding.

**The name of a party is NOT unique inside the business.** The domain never
refuses two parties that share a name, and the reading of the parties lists
them all. The table holds the same rule: there is no index of the name.

**The tax identity of the party is an OPTIONAL fact of it.** It is the NIT
of a person, of a store or of a farm, and the book never asks for it and
never assumes one (`party.WithTaxIdentity`). The book does not know the
format of a tax identity either: it is kept the way the owner said it, and
the rule of the tax authority and the advice of the professional of the
owner are not read here.

## The model of the product and the price list

`product.Product` is the thing the bakery sells: the product, the price
and the channel that buys it. `product.Price` is the money of the product
in a channel, with the day it was set.

**The product of the bakery IS the thing it sells.** It carries the recipe
it comes from — the bakery sells what it bakes — and the name the owner
reads. The product is never a second copy of the recipe: it carries the
identity of it and nothing else (`product.New`).

**The product comes into being the first time it is named.**
`product.TheProduct` answers the product the business already holds and
opens the one it does not know, and the name is never guessed: a product
without a name is refused (`ErrInvalidName`), the name of a product is
taken inside the business (`ErrDuplicateName`), and one recipe backs one
product inside the business (`ErrDuplicateRecipe`). The name is compared
the way a person reads it: trimmed and without case.

**The product is backed by a recipe of ITS OWN business.** The recipe
belongs to the business that owns it, and the product of the bakery is
never backed by the recipe of another one (`ErrOfAnotherBusiness`). The
recipe and the product belong to the business that owns them, and the
price of the list is of a product of that business too. The row of the
storage is built by the domain as well (`product.NewFromFacts`), and the
table holds the same rule as a defense in depth.

**The channels are the two of the owner.** `product.Channel` names the
retail and the wholesale, and the domain reads the two ways the owner says
them: `ParseChannel` reads `retail` and `minorista`, and `wholesale` and
`mayorista` too. There is no third channel and no default.

**The channel is ALREADY IN THE PRICE from the first row.** Every price
carries the channel it was set for, so that the discount of the wholesale
never has to rebuild the history of the prices. There is no "the price
without a channel", and a price without a channel is refused
(`ErrUnknownChannel`).

**The money of a price is a positive number of whole pesos.** The money of
the book counts the pesos the person has, and a price of zero pesos is not
a price the owner charges (`ErrInvalidPrice`). The price carries the day it
was set (`ErrMissingDay`) and names the product it is of
(`ErrMissingProduct`).

**The price carries THE WAY SHE CHARGES.** Three thousand pesos is not a
price until the book knows whether it is per loaf, per gram or per
kilogram. The way of charging belongs to the domain of the purchase
(`purchase.PriceBasis`) and the price of the list REUSES it: there is no
second copy of the basis. A price without its basis is refused
(`purchase.ErrUnknownPriceBasis`), and the book never guesses which of the
two ways of selling she uses.

**The price list is a HISTORY.** The rows are kept and nothing is rewritten
when a price changes: `product.ThePriceOn` answers the price that holds
for a product and a channel on a day, and it is the LAST one set on or
before that day. Two prices of the same day are both kept, and the last one
that was set is the one that holds. `product.ThePriceList` answers the
whole list: the price that holds for every product and every channel of the
business.

**The price of a product is a fact of the BUSINESS.** It never becomes a
column of the inventory, of the recipe or of the stock: it lives in the
table of the prices, with a row of its own.

**The tax treatment of a product is an ATTRIBUTE OF THE PRODUCT, and it is
OPTIONAL.** An empty one means the owner has not decided it
(`product.WithTaxTreatment`), and the book never derives it from the price
and never assumes a rate. The treatment is the fact the owner states, and
nothing more: the rule of the tax authority and the advice of the
professional of the owner are not read here.

**The price list belongs to a business of its own.** `product.OwnedBy`
states the rule: the price list belongs to a business of kind `business`,
and the Personal Business of a person holds the money of that person and
not the price list of the bakery (`ErrWrongBusiness`).

## The model of the order and the round

`order.Order` is what the bakery owes: the client that ordered, the
products it ordered with the quantities of them, and the day it is due.
`order.Round` is the day and the orders it carries.

**The order NAMES the party and the products.** It never copies the client
and never copies the product: the models of the party and of the product
are imported and not rewritten (`order.New`). The name the owner reads of
the party and of the product travel with the order, the way the name of the
recipe travels with the sale.

**The party of the order is a CLIENT.** The bakery does not owe bread to a
provider (`order.ErrNotAClient`), and the party is a party of the business
that owns the order (`order.ErrPartyOfAnotherBusiness`). The products of
the lines are the products of the price list of that business
(`order.ErrProductOfAnotherBusiness`).

**The order is written OPEN, and the domain owns the state of it.** An
order that was not delivered stays open (`order.StateOpen`), and the
delivery of an order is written with the ROUTE of the round: the route is
what the agent computes when it comes back from the round, and the
increment that writes the route writes the delivery with it. The state of
the book is the two of them (`order.AllStates`), and the row never decides
which of them an order carries (`order.NewFromFacts`).

**The DELIVERY of an order is the domain's transition, and it happens
once.** `order.Order.Delivered` moves an open order to the delivered one,
and it refuses an order that was delivered already
(`order.ErrAlreadyDelivered`). The agent never writes the state of an order
by itself: the route that visits the client delivers the order.

**The unit of a line is THE WAY THE OWNER CHARGES.** The book knows three
ways of charging, and it is the model of the tree: `purchase.PriceBasis` —
the piece (`unit`), the gram (`gram`) and the kilogram (`kilogram`). The
way the owner said it is read in the vocabulary of the book
(`purchase.ParsePriceBasis`): the piece ("unidad", "unit", "piece"), the
gram ("g", "gramo", "gram") and the kilogram ("kg", "kilo", "kilogramo"). A
way of charging the book does not know is refused
(`order.ErrUnknownUnit`), and a line with no unit at all is refused
(`order.ErrMissingUnit`). The line keeps the way the book knows it, and
the storage and the protocol carry that name.

**The quantity of a line travels with the way she charges.** When she
charges by the PIECE the quantity is a whole number of pieces: the book
never splits a piece (`order.ErrSplitPiece`). When she charges by the gram
or by the kilogram the quantity is any positive amount of weight: the book
never converts a weight into a number of pieces. A quantity that is not
positive is refused in the two ways (`order.ErrInvalidQuantity`), and the
order carries the day it is due (`order.ErrMissingDay`).

**The ROUND of a day carries the orders that are DUE on it.** It is the
bread the bakery owes that day (`order.TheRound`). A round with no day
carries every order of the business: the book of the orders is read whole
when the owner names no day.

**The round answers WHAT IS STILL OWED: every open order of the business.**
The rule and the round are apart: the round is the day and the orders that
are due on it, and the owed is every open order of the business — the ones
that are due on the day and the ones that fell due on an earlier day and
were not delivered (`order.Round.Owed`). The order that was delivered is
not owed, and the order that was not delivered is, no matter the day it
fell due on.

## The model of the route

`order.Route` is the route of the round of a day: the stops in the order
the agent will visit the clients in, and what each stop costs. The agent
is the only component of PACA that has access to the outside, and it
computes the route there; the book only holds the fact of it.

**The stop carries the order it visits and the MONEY of the visit.** The
cost is the transport to that client, in whole pesos: it is never negative
(`order.ErrInvalidCost`), and it may be zero, because a stop the owner
walks to costs nothing. The money of the book is the money of the book
(`ledger.Amount`).

**The TOTAL of the route is derived, and it is never typed.** The route
carries no total of its own: `order.Route.Total` adds the money of every
stop, and the agent never writes it.

**The route of a business and a day is written ONCE.** A second route for
the same day is refused (`order.ErrRouteAlreadyWritten`), and the routes
already written are the ones the domain is asked about
(`order.TheRoute`).

**Every stop is an order that the business still OWES.** The stops are
EVERY open order of the business — the ones that are due on the day of the
route and the ones that fell due on an earlier day and were not delivered —
and the rule is the one the round already answers (`order.Round.Owed`): the
order that was not delivered is owed, no matter the day it fell due on. A stop
that names an order the business does not hold is refused
(`order.ErrStopNotAnOrder`), and a stop that names an order that was
delivered already is refused (`order.ErrAlreadyDelivered`). The leftover
order is delivered by the route of the next day that is still open.

**The route visits every order ONCE.** A route that names the same order in
two stops is refused (`order.ErrDuplicateStop`): the total of the route is
derived from the stops, so a route that visits an order twice would count
the money of that visit twice.

**A route without a stop visits nobody, and it is not a route.** The route
carries at least one stop (`order.ErrNoStops`), and it carries the day of
the round (`order.ErrMissingRouteDay`).

**Writing the route DELIVERS the orders of its stops.** The route the
agent writes back is the round it made, and the delivery travels with it
(`order.Route.TheDelivery`): the orders of the stops move to the delivered
one, in the same unit of work that writes the route.

**A ROUND WITHOUT A ROUTE IS A ROUND THAT WAS NEVER PLANNED.** The round
carries the route that was written for it (`order.Round.Route`), and
`order.Round.Planned` says whether there is one. The reading of the round
answers the route of the day with it (`order.TheRoundWithRoute`).

**The business of the orders that is not there has its own refusal.**
`order.ErrBusinessNotFound` names the business of the order, and the agent
is never told to open the personal business of a person for a business it
mistyped.

**The orders belong to a business of its own.** `order.OwnedBy` states the
rule: the orders belong to the business of the bakery, the one that sells
the bread, and the Personal Business of a person holds the money of that
person and not the orders of the bakery (`order.ErrWrongBusiness`).

**The money of an order is NOT part of the model.** The acceptance of the
increment does not state it, and the book never adds a rule nobody wrote:
whether the order carries the money of the price list of the day it was
made is a question of the owner, and it waits for her.

## The transaction of the storage

The unit of work is the port of a transaction of the storage, and it lives
in its own package because it is a capability of the storage and not of a
person or of a book:

```go
Do(ctx, work) error
```

The work runs inside one transaction: the rows are written whole or not at
all. The caller never sees the transaction, and never commits by itself. The
error of the work is the error of the call.

## The model of the health

`health.Status` is the state a service is in: `ok`, `degraded` or
`unhealthy`. `health.Check` is one component the service watches, with the
time it took to answer, and `health.Report` is the answer of the service:
its identity, its state, the checks it ran and the time it has been up.

**The state of the service is the worst state of its components.** The rule
lives here, in `Status.Worst`, and every service answers the same way: one
failing component makes the whole service unhealthy, one degraded component
makes it degraded, and a service with no checks at all is healthy.

**An unknown state is never trusted.** A status that the domain does not know
is `unhealthy`, never `ok`: the report of a service that nobody understands
must not be read as a promise.

## Running it

```sh
make -C packages/domain all    # the full quality gate
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
| `make cover`      | Measures the coverage and **fails below 80%**.                    |
| `make race`       | Runs the test suite with the race detector.                      |
| `make bench`      | Runs the benchmarks.                                              |
| `make perf`       | Checks the allocation budgets, then runs the benchmarks.          |
| `make vuln`       | Runs `govulncheck` over the dependencies.                         |

The tools are pinned in the repository (`tools/go.mod`), so every machine
and every container uses the same versions.

### The coverage, in one number

The module talks to no database, so the whole suite runs without one and
the gate measures it as a whole: the rules of the person are asserted
without a PostgreSQL, and the assertions are the tests.

## The tests

The unit tests live with the code they cover. There is no container test
here: the rules of the domain are proven without a database, and the rules
of the table are proven against a real one by the container tests of the
database library (`packages/database`).

## The versions

| What             | Version  | Why                                                     |
|------------------|----------|---------------------------------------------------------|
| Go               | 1.27.3   | The latest security release of the 1.27 line.            |
