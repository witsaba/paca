# PACA — the bakery: the second business of PACA, written from what you say

Feature document for the second business of PACA: the bakery the owner is about
to open in the community where she lives. The bakery has its own accounting,
its own inventory, its own recipes, its own costs and its own clients, and the
whole of it enters PACA the way it enters the owner: **she says it, and Paca
writes it down**. The work is divided the way a block of marble is divided:
many small cuts, each one a finished piece, and never the whole sculpture in
one session.

- **Project:** `packages/domain`, `packages/database`, `apps/mcp-server`, `hermes`.
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/bakery-business`, branch `feat/bakery-business`.
- **Scope:** the registration of the business, the utterance that writes the record, the units of measure, the events of the business (the purchase, the production, the exit), the cost of a batch, the real price of a product, the parties, the orders and the reminders.
- **Out of scope, for now:** the payroll of employees, the multi-currency, the suggestions of improvement (the data is recorded for them; the suggestions themselves are a later work), and every piece of data of the owner (see S19).

---

## The method

The maintainer asked for the work to be divided the way Michelangelo divided
the marble (S32): **granular but incremental, never the whole in one session**.
The method is the following, and it is a rule of this document, not a habit.

**The block is whole from the first day.** This document describes the whole
sculpture before the first cut is made: every movement, every increment, every
rule. Nothing is lost, and the shape of the work is never in doubt.

**The cuts are vertical, never horizontal.** An increment is not "the tables
of the ingredients" and then "the tools of the ingredients". An increment runs
from the rule of the domain to the tool the owner calls, and ends with a
capability she would recognize. There is no increment that lands as a layer
that nothing uses: that is debris, and the gate of the repository pins it.

**The order is decided by risk, not by tidiness.** The increments that would
kill the feature come first. The utterance (Movement II) is the feature itself
(S20): if Paca cannot turn a sentence into a record, nothing else matters. The
cost (Movement III) is the second risk: if the real price cannot be derived
from what was really paid, the whole accounting is a fiction. The tax work
(Movement V) is heavy but unsurprising, so it comes last. The registration of
the business comes before everything else (S36): a purchase without a business
is a purchase nowhere.

**The detail of a cut is written when the cut is made.** The map of every
increment is here; the spec of an increment is written at the start of its own
session, quoting the owner. A distant cut on unseasoned marble is how the
block breaks.

**The design of a cut may correct the map.** The map is a plan, not a promise:
when the design of a cut finds a hole, the hole is filled and the map is
corrected in the same breath. The design of the first cut (T1) did exactly
that: it found that the registration of the business was missing from the map
altogether, and it added the movement.

**The increment is cut again into steps.** Every increment below is further
divided into steps when its session starts; a step is one focused piece of
work with its own test. No session crosses from one movement into the next.

**The polish is a real pass.** After the movements work, a later pass removes
the tool marks of the roughing — the naming, the shape of the descriptions of
the tools, the messages — without adding behavior.

**The document carries the rules; the data comes at runtime.** This document
holds no recipe, no receipt and no price of the owner (S19). The evidence
gathered during the exploration is named in the Log with its sources, and its
numbers live in the conversation with the owner, which is where they belong.

---

## The delivery of the feature

The feature is too big for one review, and it is not one review: it is
thirty-one increments, each one a session and a commit. The delivery keeps
the review small.

**The branch is the whole block.** `feat/bakery-business` carries every
increment of the feature, in the order of the movements, and it is never
merged as one pull request.

**The review unit is the increment.** Each increment lands as its own commit,
with its tests and its documentation, and the commits of one increment
together stay inside a review of about sixty minutes.

**The pull requests are the slices.** When the pull requests open, the
feature is cut into slices of about four hundred changed lines: one
increment, or the two that belong to the same behavior, never a part of an
increment. The slices are stacked in the order of the movements, and each
one states its dependency on the previous, what it delivers and what it does
not.

**The size of an increment is the size of its proof.** An increment that is
vertical carries its own tests, and the tests are most of its lines: the
registration of the bakery was two thousand lines of which a thousand were
the proof. A pull request that carries such an increment asks for the
exception of the size, and it says why: the increment is one behavior with
its proof, and splitting it would put half of a behavior in a review. The
exception is the owner's to accept, and it is asked once, with the first
pull request.

**The chain strategy is the owner's decision.** The two shapes of the
chained pull requests are the stack to `main` — every slice lands in order
and `main` stays alive and green — and the tracker, where the whole chain
waits behind a draft that merges at the end. The first pull request asks,
and the answer is written here.

**The boundary of the first slice.** The two commits of the document are the
design, and they are not code. The first slice of code opens with T2, the
registration of the bakery, and it is the first thing a reviewer of Paca
will see that is not a document.

---

## Specs

Each spec is the owner's own wording, quoted verbatim, except the ones marked
*(decided)*, which record a choice she took during the discussion.

### The business

**S1 — The second business**

> My business is a bakery. I got in mind to open comming soon a new bakery in the comunity where I live, I need to have a way to have inventory, reciepes, and track costs and the real price of produce a bread.

**S2 — What the business needs around the book**

> I off course i need create the new business with its own contability, but have providers information, calendar reminders, and I detected too a distrubution system that allows me to make deliveries to clients can be persons or stores.

**S3 — The vocabulary already holds the second business** *(decided)*

The domain of PACA already carries the kind `business` and already states that
a person may own more than one business: the Personal Business is found
through its owner, and the business is found by its own identity
(`packages/domain/business/business.go`). The kind exists and **no use case
creates one yet**: the port of the storage offers no way to list the businesses
of an owner, and no tool of the MCP server creates or lists a business. The
first cut of this feature lands on that hole.

**S4 — The clients are persons, stores and farms**

> my clients can be so far persons, stores (with a legal human as representation), or farms which some have a legal figure but can be clients too.

### The cost

**S5 — The real price has three layers** *(decided)*

> Direct + labor + overhead (full PAC)

The three layers are never hidden inside a single number: the report shows the
direct cost, the labor and the overhead apart, and their sum apart.

**S6 — The full real inventory** *(decided)*

> Full real inventory (lots, expiry, valuation)

**S7 — No batches** *(decided)*

> No batches

The consequence is recorded in the Log: the yield of a recipe is a declared
constant, and the actual production is not measured per bake.

**S8 — The taxes, in full** *(decided)*

> Full: NIT + IVA + electronic invoicing

The tax treatment of a product is an attribute of the product, and the
implementation of the electronic invoice is never assumed by the repository:
it follows the rule of the tax authority and the advice of the professional
of the owner.

**S9 — The payroll, out of scope** *(decided)*

> Out of scope for now

The labor of the PAC is therefore the labor of the owner alone, and the model
of the labor of an employee does not exist yet.

**S10 — The units and the grams**

> yea, support both, I think the user now have notion about to see the price for gr. But when is unit, is not worry about the grams.

Both ways of selling are supported: by unit, and by weight. The cost answers
in both too, per unit and per gram, or the comparison between two products is
a lie.

**S11 — The unit of sale**

> i'll be working on making individuals of aproximated of 90 gr the unity before baking it.

**S12 — The weight of the batch**

> the final is baked 900 gr approximately.

**S13 — The two fats**

> I'll pretent to use Organic animal-based fat—specifically from pigs and cattle—will be the two primary types.

The two are two ingredients of the book, never one: they carry different
prices and different certifications.

**S14 — The conversion is not measured yet** *(decided)*

> assume the raw gr so far

**S15 — The starter is not detailed as a job** *(decided)*

> Do not detail the starter feeding as a job, assumme that exists.

> is the 100 gr the starter! that is my differential! i'm not using industrial levain.

### The prices

**S16 — The prices are not fixed** *(decided)*

> i'll be brining the information in the exact moment, example, the next saturday I'll say ' lets prepare the molde bread. I bought this flour and cost...' so that is that i need. not fixed prices due to the reality is different.

The price of an ingredient is not a field of the ingredient: it is a fact
about a purchase that happened, and the purchase is what the owner dictates.

**S17 — The purpose of the history**

> the important here is to later see what can I do for improve prices or the product, so paca can suggest the improvements when got data to analyze.

> the suggest improvements is for the future, i mentioned just for knowing where i'll be going on.

The requirement of this feature is therefore not the suggestion: it is that
every event is recorded in the shape a later analysis can use.

**S18 — The research is a reference, never the local price** *(decided)*

> I can for sure ask to paca to investigate prices but not all the companies here sell online.

The price the owner pays exists only in the market of her town and only in her
mouth on the day. The research of Paca returns public references with their
source and their date, and it is never stored as the price of an ingredient.

### The utterance

**S19 — The document carries no recipe, no receipt and no price of the owner**

> Do not write any recieps, the feature MUST to be able to get my reciept later, i mentioned as a example.

The examples the owner gave are examples of the flow, not content of this
document. The capability to capture them is the feature; the data itself is
the owner's and lives in the book, in the conversation, and never in this
document.

**S20 — The feature is the natural language**

> I need that Paca can create the reciept based on my natural language. That's is the feature really.

**S21 — Paca reduces the ambiguity**

> Paca MUST to reduce the ambigüety as much as posible, that is even the feature I need from paca.

An utterance yields a **candidate**, never a fact: Paca proposes the record
and the owner confirms it. Paca infers what can be inferred safely, asks
about what cannot, and says plainly what it did not understand and what it
still needs. The original sentence is kept with the record, so the entry can
be audited and derived again.

**S22 — The weekend conversation**

> I wanna for example startiing the weekend with the preparation and goes talking to Paca about the ingredients, the products i waste and the final product i got with grams.

### The book of the bread

**S23 — The production, the units and the expiry**

> This must to have a expiration date too (i dont know it yet but must to be and aproximate)

The shelf life of a product is approximate until the owner corrects it, and
every produced unit carries the day it expires.

**S24 — The exits of a unit**

> i sell one, and consume the second one, or even I donate it, so must to be that I'm assuming this cost, can be I pay for it or just discount as a lose.

Four exits, each with its own treatment in the book: the sale, the consumption
by the owner, the donation and the waste. The waste carries its reason, and
the reason is what a later analysis reads.

**S25 — The waste is a movement** *(decided)*

> Recorded as movements, from the start

**S26 — The valuation is FIFO** *(decided)*

> FIFO

The rotation of the perishable output follows the earliest expiry; the
valuation follows FIFO. The two are different rules and the model keeps them
apart.

### The business around the bread

**S27 — The price list, and the channel**

> One price list for the whole bakery

> I think for stores must to have a lower price, but for now, asumme all pay the same price.

**S28 — The orders and the route** *(decided)*

> Orders plus route optimization

**S29 — The reminders** *(decided)*

> Fact in PACA, alarm in the agent

**S30 — The places**

> for now use the concept of "Hidden kitchens"

**S31 — The currency** *(decided)*

> Colombian pesos only

### The method

**S32 — The division of the work**

> use a similar method of the micelian works. Divide, and divide in order to have granular but incremental tasks. I dont want you work entirely in the bakery business in a single session but divided.

**S33 — The skeleton first, and the flow of the utterance designed before any code**

> use the 3

The flow of the utterance — what Paca infers, what it asks, what it refuses —
is designed in a session of its own, and this document is extended with it
before the first line of the feature is written.

### The guide and the learning

**S34 — A recipe is a guide, not a list**

> a reciepe is not only a list of ingredients but too a guide to learn and teach later.

A recipe carries the process: the steps, their times, their temperatures and
their cues, and the shelf life of what it makes. The numbers of the process
live in the domain; the lessons live in the memory of Paca.

**S35 — Paca learns from the errors and the wins**

> Paca could learnd to about my errors and wins.

The learning log is where the owner records what happened, and where Paca
learns to tell her what she already knows: the batch that rose slower than
the recipe says, the oven that ran hot, the dough that was too stiff.

### The flow of the utterance

These are the specs of the design of T1, the first cut of the feature.

**S36 — The registration of the business comes first** *(decided)*

> ok I wanna add the capacity to register a new business

The bakery is registered before anything is bought: it is a business of its
own kind, with its own book and its own accounts, and the tools of the
utterance need its identity to write anything at all.

**S37 — The agent reads the sentence; the domain decides**

The model reads the sentence and proposes; the domain decides what is true
and what may be written. The domain never receives a sentence: it receives a
candidate and a decision, and the rules of the domain — what a purchase is,
what it needs, what may not be written — are the only authority over what
enters the book.

**S38 — Paca asks with the options she chooses between**

> When Paca is not sure which one you meant (two flours you buy, two milks), what should it do? — Ask with the likely options

Paca never asks an open question when it can name the options: it offers the
two or three candidates it found and the owner answers with one word.

**S39 — Paca asks only what it cannot infer**

> When several things are missing at once, how does Paca ask for them? — Only what it cannot infer

The domain states what is missing and why, and the agent asks for exactly
that. A question that the answer is already in the conversation is a
question the owner did not ask for.

**S40 — Everything waits for the yes**

> When must Paca show you the record and wait for your "yes" before it writes anything? — Always, before anything is written

There is no fast path: every purchase is shown in the owner's words and waits
for her yes. The word she uses to confirm is kept with the record.

**S41 — The draft is a fact of the business, not of the conversation**

The candidate is stored by the business, with its state, its revision and its
original sentence. The transport of the MCP server is stateless and the
context of the model is the only thing that carries a turn to the next one: a
candidate that lives only in the context is a candidate that can be
paraphrased, lost or silently changed between the proposal and the yes. For
the money, the confirmed record must be exactly the record the owner saw.

**S42 — The stated and the inferred are apart**

The fields the owner said and the fields Paca filled are never mixed, and the
answer of Paca names the inferred ones. The amount paid is **never**
inferred: if she did not say the price, the draft is missing it and Paca
asks. The day, the currency and the business are the three things Paca may
fill by herself.

**S43 — The measure and the price are value objects**

The quantity and its unit travel together, and the price is either the total
paid or the price of the unit: never two silent truths. When she gives both,
the two must agree; when they do not, the domain refuses and asks which one
is right.

**S44 — The draft expires, and a stale revision is refused**

A draft that was neither confirmed nor cancelled in seven days expires: the
price of a market does not wait. The confirmation carries the revision the
owner saw, so a draft that changed under her eyes is refused until it is
shown again.

**S45 — The money of the book counts whole pesos**

The rule of the book holds: the money is whole pesos. A price in centavos is
rounded to the nearest peso when it enters the book, and her exact words stay
in the utterance that is kept with the record.

**S46 — The names the business buys are its own catalog**

The catalog begins empty and grows with what she buys: the name she reads
and the names she uses for it. A name the catalog does not know is not an
error: it is recorded as she said it, and the answer tells her it is new.

**S47 — The tools of the utterance**

Five tools, and no other: `match_items` (the name she said, against the
catalog), `draft_purchase` (the proposal, with what is missing and what is
ambiguous), `confirm_purchase` (the writing), `cancel_draft` (the change of
mind) and `read_drafts` (the pending ones, for the recovery of a lost
conversation).

**S48 — The answer of Paca has three parts**

Every answer of the utterance says what it recorded, what it did not
understand, and what it still needs. The data of the three parts comes from
the domain; the wording is the agent's.

**S49 — The behavior of Paca lives in the soul**

The runtime of the agent is an upstream image: the repository owns two
seams of it, the identity of Paca in `hermes/soul.md` and the description of
the tools. Therefore the behavior of reading, asking, offering the options,
confirming and reporting is written in the soul, versioned with the code,
and the contract of every tool is derived from the domain and never written
by hand.

---

## The specs of the increments

The specs above are the rules of the feature. This section holds the spec of
each increment: the rules it answers, the acceptance that proves it, and the
surface it may touch. An increment is written here at the start of its own
session, and the worker that writes it reads this document and nothing else.

### T2 — the registration of the bakery

**The rules:** S1, S2, S3, S36.

**The acceptance:**

- The bakery is registered with the name the owner reads and the currency of
  its book. The name may not be empty; the currency is the peso when she has
  no opinion.
- The book of a business of kind `business` starts with the accounts of a
  business: the cash, the bank, the inventory, the sales, the cost of the
  sales, the labor, the overhead, the waste and the capital.
- A debt and an account that is not in the list are not in the seed: they are
  created when they happen.
- The business and its accounts are written in one transaction: a book that
  starts with half of its accounts lies about what the owner has.
- The businesses of a person are read back, with their kinds, and the personal
  business is one of them.
- The tool that lists the businesses says the kinds of a business, and the
  tool that opens one says that the name is required and that the currency
  of the book is the peso.

**In scope:** `packages/domain/business`, `packages/domain/account`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`, `apps/db-admin`.

### T3 — the draft and the tool that proposes it

**The rules:** S21, S37, S41, S42, S44, S47, S48.

**The acceptance:**

- The package `packages/domain/purchase` holds the draft: its state, its
  revision, the sentence it came from, its lines, what is missing, what is
  ambiguous, what was inferred, the day it was proposed and the day it
  expires.
- A line carries the name as the owner said it, the quantity with its unit,
  and the money: the total paid or the price of the unit, never two silent
  truths.
- The quantity and its unit travel together: a line without a unit is
  refused. The money is required and is **never inferred**: a line whose
  price was inferred is refused. When both the total paid and the price of
  the unit are given, they must agree.
- The day, the market and the invoice are optional; the day of today is the
  day of a draft that names none.
- A draft expires seven days after it was proposed, and an amendment grows
  its revision.
- The tool `draft_purchase` proposes the candidate: it validates it, keeps it
  as a draft, and answers what it understood, what is missing, what is
  ambiguous and what was inferred. It never writes the book.
- The table of the drafts, the repository that implements the port, and the
  container test that proves the rules of the table.

**In scope:** `packages/domain/purchase`, `packages/domain/business`,
`packages/domain/account`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T4 — the purchase and the tool that writes it

**The rules:** S16, S19, S20, S43, S45, S47, S48.

**The acceptance:**

- The purchase carries the business, the day, the market, the invoice, its
  lines, the sentence and the word of the confirmation. A purchase never
  changes: it is the fact of what was bought.
- The confirmation of a draft writes its purchases in one transaction, and
  the money of the payment moves from the account that pays — the cash of
  the business, unless the owner names another — to the inventory.
- A confirmation of an expired draft is refused; a confirmation of an old
  revision is refused; a refused confirmation leaves nothing behind.
- The money of the book counts whole pesos: a price in centavos is rounded
  to the nearest peso, and the sentence keeps the exact wording.
- The tool `confirm_purchase` writes it and answers the three parts: what it
  recorded, what it did not understand and what it still needs.
- The table of the purchases and of their lines, the repository, and the
  container test.

**In scope:** `packages/domain/purchase`, `packages/domain/ledger`,
`packages/domain/account`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T5 — the catalog of the names and the tool that matches them

**The rules:** S46, S47.

**The acceptance:**

- The package `packages/domain/ingredient` holds the thing the business buys:
  the name the owner reads and the names she uses for it.
- The matching of a spoken name answers exact, ambiguous — with its options —
  or unknown.
- A name the catalog does not know is not an error: it is recorded as she
  said it, and the answer tells her it is new.
- Two ingredients of the same business never carry the same name, and the
  catalog of a business starts empty.
- The tool `match_items` answers the candidates of a name.
- The table of the ingredients, the repository, and the container test.

**In scope:** `packages/domain/ingredient`, `packages/domain/purchase`,
`packages/domain/business`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T6 — the pending drafts: the tools that cancel them and read them

**The rules:** S41, S44, S47.

**The acceptance:**

- The state machine: a draft starts in `proposed` and ends in `confirmed`,
  `cancelled` or `expired`. It never returns to `proposed`.
- The tool `cancel_draft` drops a draft the owner does not want; the tool
  `read_drafts` lists the pending ones with what is missing in each, so a
  lost conversation can be recovered.
- Every error of the domain that the tools can return has a message written
  in the words of the caller.

**In scope:** `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T7 — the instructions of Paca in the soul

**The rules:** S38, S39, S40, S49.

**The acceptance:**

- The soul says how Paca reads the sentence, what she may fill by herself
  (the day, the currency, the business), that she asks only for what she
  cannot infer, that she offers the options when she is not sure which one
  the owner meant, that she shows the record in the owner's words and waits
  for the yes, and that she reports the three parts.
- The soul never invents a price, and it never writes a record that the
  owner did not confirm.
- The README of the agent names the behaviour.

**In scope:** `hermes/soul.md`, `hermes/README.md`.

### T8 — the recipe and the tool that captures it

**The rules:** S11, S12, S19, S34, S47.

**The acceptance:**

- The package `packages/domain/recipe` holds the recipe: the name, the
  ingredients with their grams, the steps with their times, their
  temperatures and their cues, the yield and the shelf life.
- The ingredient of a recipe is the ingredient of the catalog, never a second
  copy of the flour.
- The yield is a declared constant, never measured per bake. The steps keep
  their order, and a step may be named by its cue.
- The tool `record_recipe` writes the recipe with the confirmation of the
  owner, the way the purchase is written: the domain refuses a recipe that
  is not whole — no name, no yield, an ingredient without its grams — and
  the message of the refusal says what the agent has to ask for.
- The table of the recipes, the repository, and the container test.

**In scope:** `packages/domain/recipe`, `packages/domain/ingredient`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T9 — the reference prices and their tools

**The rules:** S17, S18, S47.

**The acceptance:**

- The domain holds the reference: the ingredient, the price, the unit, the
  market where it was seen, the source and the day it was seen.
- A reference is never the price of an ingredient: the price of the owner
  comes from her purchases.
- The tool `record_reference` writes it with the confirmation of the owner;
  the tool `read_references` reads them with their source and their day.
- The table of the references, the repository, and the container test.

**In scope:** `packages/domain/purchase`, `packages/domain/ingredient`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T10 — the ingredient and the unit, and the tools that complete them

**The rules:** S10, S13, S14, S43, S46.

**The acceptance:**

- The units: the gram, the kilogram, the package and the piece. The
  conversion of a unit into the gram is declared, never assumed.
- The conversion of the potato and the rate of the starter are declared
  constants.
- The two fats are two ingredients with their own certification, and
  "organic" is a claim that carries its evidence. The allergens of an
  ingredient are declared.
- The unit of a purchase line is the unit of the ingredient when the owner
  names none.
- The tools `read_ingredients` and `complete_ingredient` read and complete
  what is known of an ingredient.

**In scope:** `packages/domain/ingredient`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T11 — the stock and the layers, and the tool that reads them

**The rules:** S6, S16, S26.

**The acceptance:**

- The layer: the ingredient, the quantity, the unit, the money it cost, the
  day it was bought and the purchase it comes from.
- The stock of an ingredient is the sum of the layers that are still open.
  The valuation is FIFO: the oldest layer is the first one that leaves.
- The purchase is the fact; the layer is derived from it.
- The tool `read_stock` answers what the business holds and what it cost.

**In scope:** `packages/domain/purchase`, `packages/domain/ingredient`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T12 — the cost of a batch, and the tool that reads it

**The rules:** S5, S6, S16.

**The acceptance:**

- The cost of a batch is derived from the layers that the recipe consumes,
  and never typed.
- The recipe consumes the layers in the order of FIFO.
- The cost answers per unit, per gram and per kilogram.
- The tool `read_batch_cost` answers the cost of a recipe.

**In scope:** `packages/domain/recipe`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T13 — the labor, and the tool that records it

**The rules:** S5, S9.

**The acceptance:**

- The labor is measured in hours and in three kinds: the hands-on hours, the
  hours of the oven and the hours of the waiting.
- The value of the hour is declared by the owner.
- The labor of a batch is derived from the hours of its steps.
- The tool `record_labor` writes the hours and the value of the hour.

**In scope:** `packages/domain/recipe`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T14 — the overhead, and the tool that records it

**The rules:** S5, S9.

**The acceptance:**

- The overhead is the electricity of the house and the hours of the oven, and
  the allocation is declared by the owner.
- The overhead of a batch is derived, never typed.
- The tool `record_overhead` writes the electricity and the allocation.

**In scope:** `packages/domain/recipe`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T15 — the report of the real price, and the tool that reads it

**The rules:** S5, S10.

**The acceptance:**

- The report shows the three layers apart — the direct cost, the labor and
  the overhead — per unit, per gram and per kilogram, and the price of the
  owner next to them.
- The report never hides a layer inside another one.
- The tool `read_real_price` answers the report of a product.

**In scope:** `packages/domain/purchase`, `packages/domain/recipe`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T16 — the margin, and the tool that reads it

**The rules:** S10, S17, S18.

**The acceptance:**

- The margin compares the price of the owner with the direct cost, and with
  the reference of the market.
- Every reference carries its source and its day, and a reference older than
  the owner expects is shown with its age.
- The tool `read_margin` answers the margin of a product.

**In scope:** `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T17 — the production and the tool that records it

**The rules:** S22, S23, S47.

**The acceptance:**

- The production carries the recipe, the day, the grams that went in and the
  units that came out.
- A production is written in one transaction with the layers it consumes and
  the units it makes.
- The units of a production carry the day they expire.
- The tool `record_production` writes it.

**In scope:** `packages/domain/production`, `packages/domain/recipe`,
`packages/domain/purchase`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T18 — the expiry, and the tool that reads what expires

**The rules:** S23.

**The acceptance:**

- Every unit carries the day it expires, and the shelf life of the product
  behind it is what the owner corrects. The correction is kept.
- A unit that expired is never sold.
- The tool `read_expiring` answers what expires, and when.

**In scope:** `packages/domain/production`, `packages/domain/recipe`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T19 — the waste, and the tool that records it

**The rules:** S24, S25.

**The acceptance:**

- The waste is a movement: the product, the quantity, the unit and the
  reason — burnt, spoiled, dropped or returned.
- The waste releases the cost of what it was worth, and the money of the
  waste is never lost from the book.
- The reason is what a later analysis reads.
- The tool `record_waste` writes it.

**In scope:** `packages/domain/production`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T20 — the sale, and the tool that records it

**The rules:** S24, S26.

**The acceptance:**

- The sale moves the money in and releases the cost of the units it sells, in
  one transaction.
- The sale takes the units that expire first, and the sale of an expired
  unit is refused.
- The tool `record_sale` writes it.

**In scope:** `packages/domain/production`, `packages/domain/purchase`,
`packages/domain/ledger`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T21 — the consumption and the donation, and the tool that records them

**The rules:** S24.

**The acceptance:**

- The consumption of the owner and the donation release the cost like a
  sale, and they move no money.
- The tax treatment of the consumption is marked as pending the professional
  of the taxes: the domain records the movement and names the question.
- The donation is a social expense of the business.
- The tool `record_exit` writes the exit of a unit.

**In scope:** `packages/domain/production`, `packages/domain/purchase`,
`packages/domain/ledger`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T22 — the freshness first, and the tool that reads the rotation

**The rules:** S26.

**The acceptance:**

- The rotation follows the earliest expiry and the valuation follows FIFO:
  the two rules are apart and the model keeps them apart.
- The unit that leaves a production is the one that expires first.
- The tool `read_rotation` answers what leaves first and when it expires.

**In scope:** `packages/domain/production`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T23 — the party and its tools

**The rules:** S4.

**The acceptance:**

- The party: the person, the store with the legal human that represents it,
  and the farm.
- The roles: the client and the provider. A party may hold both.
- The address and the way of reaching the party are facts of the party, and
  the legal figure of a farm is a fact of the party, never a second person.
- The tools `open_party` and `read_parties`.

**In scope:** `packages/domain/party`, `packages/domain/user`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T24 — the providers, the invoices and the tax, and their tools

**The rules:** S2, S8.

**The acceptance:**

- The provider is a party, and the invoice carries its number, its day, its
  total and the tax that travels with it.
- The tax of an invoice is never assumed: it comes from the invoice.
- The purchase points at the invoice it came from.
- The tools `record_invoice` and `read_invoices`.

**In scope:** `packages/domain/party`, `packages/domain/purchase`,
`packages/domain/ledger`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T25 — the price list and the channel, and its tools

**The rules:** S27.

**The acceptance:**

- The product, the price and the channel that buys it: the retail and the
  wholesale.
- For now the two channels carry the same price, and the channel is already
  in the price so that the discount of the wholesale does not have to rebuild
  the history.
- The price of a product is a fact of the business, and never a column of
  the inventory.
- The tools `set_price` and `read_prices`.

**In scope:** `packages/domain/product`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T26 — the orders and the rounds, and their tools

**The rules:** S2, S28.

**The acceptance:**

- The order: the party, the products, the quantities and the day it is due.
- The round: the day and the orders it carries. An order that was not
  delivered stays open, and the round of the day answers what is still owed.
- The tools `record_order` and `read_orders`.

**In scope:** `packages/domain/order`, `packages/domain/party`,
`packages/domain/product`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`apps/mcp-server/internal/ports`, `packages/database`.

### T27 — the route, and the tool that writes it

**The rules:** S28.

**The acceptance:**

- The route is computed by the agent, which is the only component of PACA
  with access to the outside, and it is written back into the round.
- The route carries the order of the stops and what each one costs.
- A round without a route is a round that was never planned.
- The tool `record_route` writes it.

**In scope:** `packages/domain/order`, `apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

### T28 — the commitments and the reminders, and their tools

**The rules:** S2, S29.

**The acceptance:**

- The commitment is a fact of PACA: what is due and the day it is due. A
  commitment that was met is kept, with the day it was met.
- The alarm is the work of the agent: PACA holds the fact and the agent
  rings.
- The tools `record_commitment` and `read_commitments`, and the soul names
  the alarm.

**In scope:** `packages/domain/commitment`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`, `hermes/soul.md`,
`packages/database`.

### T29 — the learning log, and its tools

**The rules:** S35.

**The acceptance:**

- The entry: the day, the production or the batch it is about, what happened
  and what was learned.
- The log is what PACA learns from, and it is the memory of the errors and
  the wins of the owner.
- The log is never rewritten: a new entry answers an old one.
- The tools `record_learning` and `read_learning`.

**In scope:** `packages/domain/learning`, `packages/domain/production`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`,
`packages/database`.

### T30 — the capacity of a weekend, and the tool that reads it

**The rules:** S9, S22.

**The acceptance:**

- The capacity: the hours the owner has, and the units she can make with
  them.
- The capacity is derived from the labor of the recipes and the hours the
  owner declares.
- The capacity is never a promise of the calendar: it is what the weekend
  can carry.
- The tool `read_capacity` answers it.

**In scope:** `packages/domain/recipe`, `packages/domain/purchase`,
`apps/mcp-server/internal/application`,
`apps/mcp-server/internal/adapters/inbound/mcpserver`.

---

## Tasks

One line per increment: the identifier, the specs it answers, what it delivers,
the route and the commit that holds it. The routes are **inline** (the
orchestrator writes it) and **worker** (a bounded writer reads this document
and writes the unit). Every increment is cut again into steps at the start of
its own session, and every step carries its own test.

The default route is **inline**: the increments share the packages of the
domain and of the database, and the route is decided again per increment when
its session starts.

### Movement I — The bakery exists

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T1 | S21, S22, S37, S38, S39, S40 | The design of the flow of the utterance, written into this document | inline | this commit |
| T2 | S1, S2, S3, S36 | The registration of the bakery: the use case that opens it, the accounts of its book, the tools that open it and that read the businesses of the owner | inline | this commit |

### Movement II — Paca writes the record

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T3 | S21, S37, S41, S42, S44, S47, S48 | The draft of the utterance and the tool that proposes it | inline | this commit |
| T4 | S16, S19, S20, S43, S45, S47, S48 | The purchase and the tool that writes it | inline | this commit |
| T5 | S46, S47 | The catalog of the names the business buys and the tool that matches them | inline | this commit |
| T6 | S41, S44, S47 | The pending drafts: the tool that cancels them and the one that reads them | inline | this commit |
| T7 | S38, S39, S40, S49 | The instructions of Paca in the soul | inline | this commit |
| T8 | S11, S12, S19, S34, S47 | The recipe and the tool that captures it | inline | this commit |
| T9 | S17, S18, S47 | The reference prices and their tools | inline | this commit |

### Movement III — The cost

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T10 | S10, S13, S14, S43, S46 | The ingredient and the unit, and the tools that complete them | inline | this commit |
| T11 | S6, S16, S26 | The stock and the layers of the FIFO, and the tool that reads them | inline | `4b84d1b` and the correction of this commit |
| T12 | S5, S6, S16 | The cost of a batch, and the tool that reads it | inline | this commit |
| T13 | S5, S9 | The labor, and the tool that records it | inline | this commit |
| T14 | S5, S9 | The overhead, and the tool that records it | inline | this commit |
| T15 | S5, S10 | The report of the real price, and the tool that reads it | inline | this commit |
| T16 | S10, S17, S18 | The margin, and the tool that reads it | inline | this commit |

### Movement IV — From dough to bread

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T17 | S22, S23, S47 | The production and the tool that records it | inline | this commit |
| T18 | S23 | The expiry of every unit and the tool that reads what expires | inline | this commit |
| T19 | S24, S25 | The waste and the tool that records it | inline | this commit |
| T20 | S24, S26 | The sale and the tool that records it | inline | `this commit` |
| T21 | S24 | The consumption of the owner, the donation, and the tool that records them | inline | `pending` |
| T22 | S26 | The freshness first and the tool that reads the rotation | inline | `pending` |

### Movement V — The business around the bread

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T23 | S4 | The party and its tools | inline | `pending` |
| T24 | S2, S8 | The providers, the invoices and the tax, and their tools | inline | `pending` |
| T25 | S27 | The price list, the channel, and their tools | inline | `pending` |
| T26 | S2, S28 | The orders, the rounds, and their tools | inline | `pending` |
| T27 | S28 | The route of the round and the tool that writes it | inline | `pending` |
| T28 | S2, S29 | The commitments, the reminders, and their tools | inline | `pending` |
| T29 | S35 | The learning log and its tools | inline | `pending` |
| T30 | S9, S22 | The capacity of a weekend and the tool that reads it | inline | `pending` |

---

## Log

**L1** (2026-11-05): "ok I wanna add the capacity to register a new business, use this session only for explore and share ideas, do not make changes yet. My business is a bakery. I got in mind to open comming soon a new bakery in the comunity where I live, I need to have a way to have inventory, reciepes, and track costs and the real price of produce a bread. I off course i need create the new business with its own contability, but have providers information, calendar reminders, and I detected too a distrubution system that allows me to make deliveries to clients can be persons or stores. What else do you think am I missing it rn?"

**L2** (2026-11-05): The exploration of the repository answered the state of the ground. The domain of PACA holds the kind `business` and the rule that a person may own more than one business, and nothing else of the bakery: there is no unit of measure, no ingredient, no recipe, no product, no price, no order and no reminder in the whole tree. The ledger moves an integer amount between two accounts, so it cannot carry a gram. The port of the businesses cannot list the businesses of an owner, and no tool creates one.

**L3** (2026-11-05): The owner answered twelve questions. The answers are the specs S5 to S9, S26 to S31: the real price with its three layers, the full real inventory, the taxes in full, the payroll out of scope, no batches, one price list, the orders with the route optimization, the reminder as a fact of PACA with the alarm in the agent, the hidden kitchens, the pesos only and the waste recorded as movements. The clients answered themselves in the free text of the questionnaire, and became S4.

**L4** (2026-11-05): The owner gave two receipts and the shape of her weekend conversation, as examples. The reconciliation of the yields and the arithmetic of the two batches were computed during the session, and they are not in this document: the owner decided that the document carries no recipe and no receipt (S19), and that the capability to capture them is the feature itself (S20).

**L5** (2026-11-05): The price of the ingredients and the price of the market were researched during the session, from the wholesale bulletins of the DANE, from the retail listings and from the reported cost of a basic bread of the city. The numbers are deliberately **not** in this document: the owner brings the price of her own market on the day of the market, and the reference of the market is a capability of Paca (S18), not a table written by hand.

**L6** (2026-11-05): The owner decided the method: the work is divided the way a block of marble is divided, and the whole sculpture is not cut in one session (S32). The map of the movements and of the increments is in this document; the detail of each increment is written when its session starts. The first increment is the design of the flow of the utterance (T1), because the owner named it as the feature itself and because the ambiguity is what she needs Paca to reduce (S21).

**L7** (2026-11-05): The decisions the discussion left open, carried here so the next session finds them: the eggs of the farm are valued at the price of the community, and the farm may also sell them — the double counting is unresolved; the tax treatment of the consumption of the owner and of the donation is a question for the professional of the taxes; the shelf life of every product is approximate until the owner corrects it; the conversion of the potato and the rate of the starter are declared, not measured; the wholesale discount of the stores comes later, and the price carries the channel from the first day so it does not have to be rebuilt.

**L8** (2026-11-05): The skeleton of the feature is written: the movements, the increments, the rules and the decisions. The next session is the design of the utterance (T1), and it extends this document before any code of the feature is written.

**L9** (2026-11-05): The design of the utterance (T1) was made. The owner took three decisions that are now the specs S38, S39 and S40: Paca offers the options when it is not sure which one she meant, it asks only for what it cannot infer, and **everything waits for her yes**: there is no fast path.

**L10** (2026-11-05): The exploration of the agent side settled the shape of the flow. Hermes is not code in this repository: it is a pinned upstream image (`hermes/Dockerfile:20-22`), and the repository owns exactly two seams of it — the identity of Paca in `hermes/soul.md` and the description of the tools. The transport of the MCP server is stateless, the sessions live in the image and the memory lives in Engram. Therefore the behavior of reading, asking, offering the options, confirming and reporting is written in the soul (S49), and the candidate of a purchase is kept by the business as a draft, because a candidate that lives only in the context of the model is a candidate that can change between the proposal and the yes (S41).

**L11** (2026-11-05): The design corrected the map, and the correction is recorded here. The registration of the business was missing from the map of the thirty-one increments, although it is the first request the owner ever made and although every tool of the utterance needs the identity of a business: it is now Movement I (T2), before everything else (S36). The map has five movements and thirty-one increments.

**L12** (2026-11-05): The decisions the design left open, carried here for the sessions to come: the cents are rounded to the nearest peso because the book counts whole pesos (S45), and that is the one place where the money of the book may have to move to centavos; the draft expires in seven days (S44) and that number may become a setting of the business; the weight of a package is unknown until she says it, so "two sacks" is a measure without a conversion to the gram until the catalog of the units arrives (T11); and the catalog of the names begins empty, so the first purchase creates the first name.

**L13** (2026-11-05): The delivery of the feature was decided and written: the branch carries the whole block, the review unit is the increment, and the pull requests are the slices of about four hundred changed lines, stacked in the order of the movements. The chain strategy — the stack to `main` or the tracker — is the owner's decision when the first pull request opens, because the push and the pull requests are hers and not the feature's. The first slice of code opens with T2.

**L14** (2026-11-05): The map was corrected again, and this time by the repository itself. The gate of the structure asserts that every function of the tree is reachable from the entry point of a service, and it proves it with the analysis of the dead code: a function that no production code calls is debris. An increment that adds only a domain — the draft without its tool, the recipe without its capture — would therefore leave the tree in a state the gate refuses. Every increment of this feature is **vertical**: the rule of the domain, the use case, the tool the owner calls, the table behind it and the test that proves each one. The map is now thirty increments in five movements, and the spec of each one is written in "The specs of the increments" above.

**L15** (2026-11-05): T2 landed: the bakery is registered. The use case opens a business of the kind `business` with the nine accounts of its own book — the cash, the bank, the inventory, the sales, the cost of the sales, the labor, the overhead, the waste and the capital — and it writes the business and its accounts in one transaction. The two tools are `open_business` and `read_businesses`, and their descriptions read the kinds and the seed from the domain instead of writing them a second time.

Three things the increment taught, carried here for the ones to come:

- The name of a business is taken inside the businesses of its owner, and **the table does not hold that rule**: the domain holds it, by reading the businesses of the owner and refusing the duplicate. The rule of the name will move into the table when the owner opens her second business and the history proves the shape of it.
- The identity of a business answers as a string in the tools of the businesses, the way the onboarding answers it, while the tools of the book take it as a number. The bridge between the two is the agent, and it is a friction that the tools of the utterance will have to carry.
- Five tests of the tree assert the number of the tools, and they moved from twelve to fourteen. The count of the tools is a property of the transport, and every new tool pays it.

**L16** (2026-11-05): T3 landed: the draft of the utterance. The domain of the purchase holds the draft, its state, its revision, the sentence it came from, its lines, what is missing, what is ambiguous and what was inferred; the measure carries the quantity with its unit, and the money is either the total paid or the price of the unit, never two silent truths. The money is never inferred: a line whose price the agent filled is refused, and the draft says what it still needs. The tool `draft_purchase` proposes the candidate, keeps it as a draft of the business, and answers the four parts: what it understood, what it still needs, what it was not sure about and what it filled.

Three things the increment taught, carried here for the ones to come:

- The lines of a draft, what is missing, what is ambiguous and what was inferred are kept as JSONB: a draft is a proposal, not a fact. When the draft is confirmed, its lines become rows of the purchases.
- The money of the protocol is whole pesos, so a price in centavos is truncated before the domain sees it. The rounding rule of the book (S45) belongs to the increment that writes the purchase, and the sentence keeps the exact words of the owner until then.
- The state of a draft is `proposed` only, and the check of the table holds that one state. The cancellation and the expiry will widen it with a migration of their own.

**L17** (2026-11-05): T4 landed: the purchase. The domain of the confirmation refuses, by itself, a confirmation without the word of the owner, a draft that does not wait any more, a draft that changed under her eyes, a draft that expired, and a draft that still needs the money of one of its lines. The money of the book is whole pesos: the price of a line is rounded to the nearest peso, and the sentence keeps the exact wording. The movement of the payment comes from the account that pays — the cash of the business, unless the owner names another — and lands in the inventory, and the rule of that account lives in the domain: the seed of the book of a business now carries the **role** of every account it opens.

Four things the increment taught, carried here for the ones to come:

- A confirmation writes **one purchase** with its lines: the purchase is the header of the fact and the lines are rows of their own table. The purchase does not carry the accounts: the movement of the book carries them, with the sentence of the owner as its description.
- The confirmation of a draft is **guarded by the revision** the owner saw: the row is only confirmed when it still carries that revision. The draft under her eyes and the draft that was written are the same row.
- The table of the purchases refuses a purchase without a sentence, without the word of the confirmation, and a second purchase of the same draft: the rules of the domain are held by the table as a defense in depth.
- The README of the database library does not mention the tables of the drafts and of the purchases yet. The gap is real and the next increment closes it.

**L18** (2026-11-05): T7 landed: the soul carries the way Paca writes what the owner says. A sentence from the owner becomes a candidate and never a fact. She fills three things by herself — the day, the currency and the business — and nothing else; she never invents a price; when she is not sure which one the owner meant she names the two or three she knows instead of asking an open question; when several things are missing she asks only for what she cannot infer, and she groups what she asks in one message; she shows the whole record in the owner's words, with the fields she filled marked as hers, and waits; and she answers with the three parts, in this order: what she recorded, what she did not understand, and what she still needs. The sentence is kept verbatim with the record, so the record can always be read against it.

The soul was the only seam of the agent that the repository owns, and it is now the place where the promise of the feature lives: the tools of the utterance are the contract, and the soul is the behavior.

**L19** (2026-11-05): T5 landed: the catalog of the names. The new package of the ingredient holds the thing the business buys — the name the owner reads and the names she uses for it — and the rule of the matching: a name she said answers exact, ambiguous with its options, or unknown, and an unknown name is **new**, never an error. The tool `match_items` reads the catalog and changes nothing.

One decision the increment took, and it is the kind that only the tree can teach: the catalog has to grow somewhere, and a table that no production code writes is debris that the gate refuses. The growth therefore happens **at the confirmation**: every line whose name the catalog does not know becomes an ingredient of the business, recorded as she said it, in the same transaction that writes the purchase. A name the catalog knows keeps the ingredient it has, and an ambiguous name is not resolved: the line keeps the name as she said it.

Three things the increment left for the ones to come:

- The **aliases** are carried by the model, the column and the adapter, but only the name is written: the increment that completes an ingredient (T10) writes the names she uses.
- The answer of the confirmation does not name the new names: only `match_items` tells the owner that a name is new. It is a small answer that the next increment of the utterance can carry.
- The name of an ingredient is compared the way a person reads it — trimmed and case-insensitive — and the table holds the same rule with a unique index over `lower(btrim(name))`.

**L20** (2026-11-05): T6 landed: the pending drafts. The state machine of the draft lives in the domain — a draft starts in `proposed` and ends in `confirmed`, `cancelled` or `expired`, and it never returns to `proposed`. The tools are `cancel_draft` and `read_drafts`: the second one recovers the pending drafts of a business with what each one still needs, what it was not sure about and what Paca filled, so a lost conversation is never a lost record.

Three things the increment taught, carried here for the ones to come:

- The expiry is **computed by the domain when the draft is read**, on the day the caller names. The table needs no job to expire anything, and the row keeps the state it was written with: an expired draft is still `proposed` in the table, and the domain reads it as `expired`. The vocabulary of the domain and the check of the table agree, and the table was widened with a migration of its own.
- The amendment of a draft now takes the day, and it refuses a draft that ended or expired: an amendment is a move inside the life of a draft, not a way to bring it back.
- The answer of `confirm_purchase` now names the names the catalog did not know, so the agent can tell the owner that a name is new. The gap T5 left is closed.

**L21** (2026-11-05): T8 landed: the recipe. The new package holds the recipe with the name, the ingredients with their grams, the steps with their times, their temperatures and their cues, the yield and the shelf life. The ingredient of a recipe is the ingredient of the catalog — the identity and the grams, never a second copy of the flour. The yield is a declared constant and the steps keep their order. The tool `record_recipe` writes the recipe with the word of the owner, the way the purchase is written, and the domain refuses a recipe that is not whole: no name, no yield, an ingredient without its grams, a step without its instruction — and the message of the refusal says what the agent has to ask for.

Three things the increment taught, carried here for the ones to come:

- The port of the recipes **writes only**: nothing reads a recipe back yet. The cost of a batch (T12) and the production (T17) will widen it, and the rules they add will land in the domain and in the table together.
- The name of a recipe is **not unique inside the business** today, because the domain states no such rule, and the table holds none. If the owner wants it, the rule belongs in the domain first.
- The list of the tools of the running server was **stale since T5 and T6** — it was missing `match_items`, `cancel_draft` and `read_drafts` — and it is right now. The list of the tools is a contract of the transport, and it is proved by a test that only sees the running server.

**L22** (2026-11-05): T9 landed: the reference prices. A reference is what Paca researched in the market: the ingredient, the price, the unit of that price, the market where it was seen, the source and the day it was seen. It is **labelled as a reference** — the answer of the tool carries its kind — and it is never the price of the ingredient: the price the owner pays comes from her own purchases. The source is never empty, because a price nobody can check is a price nobody can trust. The tools are `record_reference`, which writes it with the word of the owner, and `read_references`, which reads them with their source and their day. The server now exposes twenty-two tools.

Two things the increment taught, carried here for the ones to come:

- The reference **does not move money**: no movement of the book, no account. The price of the owner lives only in her purchases, and the margin (T16) will read both sides — the cost of what she paid and the reference of the market — side by side.
- The market of a reference is **optional** today, because the only rule the domain states is that the source is never empty. If the owner wants the market to be part of the reference, the rule belongs in the domain first.

**L23** (2026-11-05): T10 landed: the ingredient and the unit. The units are the gram, the kilogram, the package and the piece, and the conversion of a unit into the gram is **declared**: the gram is one, the kilogram is a thousand, the package is the weight the owner declares, and **the piece is never converted** — the weight of a piece is not known until she says it. The certification carries its evidence and the evidence is never empty, the allergens are declared, and the aliases are written now: the gap T5 left is closed. The tools are `read_ingredients` — the catalog with what is known of each ingredient and what is still missing of it — and `complete_ingredient`, which writes what the owner declares.

Three things the increment taught, carried here for the ones to come:

- The **conversion of an ingredient is declared, never assumed**. The raw conversion is one to one because the owner decided that, for now, the grams of a recipe are the raw ones, and every other factor — the potato, the starter — reaches the book through `complete_ingredient`. Nothing in the tree invents a factor: the acceptance of the increment said "declared constants", and the values of those constants are the data of the owner, not of the repository. The mechanism is in place and proved; the numbers arrive at runtime.
- The **unit of a purchase line** is the unit of the ingredient when the owner names none: the use case asks the catalog and fills it, and the domain still refuses a line that ends without a unit.
- A declaration **never empties** a field the owner already gave: the completion of an ingredient fills what was missing and keeps what was there.

**L24** (2026-11-06): T11 landed: the stock and the layers of the FIFO. The layer is the part of a purchase that the business still holds: the ingredient of the catalog, the quantity with its unit, the money it cost, the day it was bought and the purchase it comes from. The **purchase is the fact and the layer is derived from it** — the layer is never typed by the owner and it is never a second truth about what was bought. The stock of an ingredient is the sum of the layers that are still open, and the **valuation is FIFO**: the oldest layer leaves first. The tool is `read_stock`, which answers what the business holds and what it cost.

Three things the increment taught, carried here for the ones to come:

- **The layers of an ingredient bought in different units are held apart.** The quantity and its unit travel together, and the book never adds a gram to a kilogram: the holding of an ingredient is the holding of one unit, and the stock holds one holding per unit.
- **A line the catalog cannot name has no layer, and the purchase still lands.** The layer names an ingredient of the catalog, so an ambiguous name waits for the owner; the purchase is written all the same, and the test pins that the confirmation of such a purchase writes no layer and loses nothing.
- **The expiry of the stock is not the expiry of the output.** The lot (S6) is the layer of the ingredient that was bought; the rotation of the perishable output by the earliest expiry belongs to the units of the bakery (T18, T22), and the two rules stay apart. The FIFO of the valuation is the rotation of the money, not of the product.

**L25** (2026-11-06): T11 was verified against an independent reviewer and the verification found a **regression**: the domain accepted a unit of the book that the table refused, so a purchase that confirmed before T11 now failed and wrote nothing — the rule of the unit was held by the table alone, which the contract forbids. The increment was corrected in the same branch, without changing its shape.

The correction:

- The layer carries the **canonical unit of the book** ("kg"), and the purchase keeps the **way the owner said it** ("kilos"). The layer is derived, so it speaks the unit of the book; the purchase is the fact, so it keeps the wording of the owner. A line whose unit is not one of the four units of the book has no layer, and the confirmation is refused by the domain, not by the table.
- The **nine rules of `Layer.Checked` are proved in the domain**, one test per rule, without a database: the rules of a domain are proved by the domain.
- The **real transaction** is proved against a container: a refused layer leaves neither the purchase nor the layer behind.
- A layer that claims **another business** is refused by the domain, and the read of the layers joins the purchases so a broken row is never read as the stock of a business.
- The dead surface of the package is gone (`Open`, `ByFIFO` and `NewLayer` are internal), and `read_stock` refuses a business that is not there.

Four things the verification taught, carried here for the ones to come:

- **A rule that the table holds alone is a rule that is lost.** The unit of the book was known by the domain and enforced by the table, and the two disagreed about a word the owner says. Every rule of the book that reaches a row needs its rule in the domain, and the table keeps it as defense in depth.
- **The transaction of the use case is proved with a fake until a container proves it.** The fake proves that the use case calls both writes; only the container proves that they share the transaction.
- **The money of a partly consumed layer has no rule yet** (F4 of the verification): a layer that holds a fifth of what it bought still carries the whole money. Nothing consumes a layer before T12 and T17, so the rule is not wrong today — but it belongs to T12, which states it in the domain before the first exit of a layer.
- **The money of the layer is not tied to the line of the purchase** (F5 of the verification): a row another writer broke with another positive money passes the table and the domain. The tie belongs to the increment that consumes the layer.

**Open for the owner**: the purchase line keeps the way the owner said the unit ("kilos") while the layer carries the unit of the book ("kg"). The alternative — the purchase line also carries the unit of the book, and the wording stays in the sentence of the owner — is one decision, and it belongs to the next increment that touches the line.

**L26** (2026-11-06): the owner decided the unit: **Paca saves the unit of the book**, whatever the way she says it ("kilos", "gramos", "paquetes" become "kg", "g", "paquete"), and the wording of her sentence and of the name of the ingredient stays as she said it. The rule lives in the domain — the measure of a line and the unit of a reference are read with the vocabulary of the owner and stored the way the book knows them — and a unit the book does not know ("libra") is refused with the error that names the four units. The tables hold the rule as defense in depth, and the migration normalises the rows that exist before it adds the rule.

T12 landed too: **the cost of a batch**. The cost is **derived, never typed**: it is read from the layers of the stock that the recipe consumes, in the order of the FIFO, and it answers per unit, per gram and per kilogram. The tool is `read_batch_cost`, and the transport carries twenty-six tools.

Four things the increment taught, carried here for the ones to come:

- **The money of what is left of a layer is the money of what it bought, pro rata.** The rule closes what the verification of T11 left open (F4): the stock carries the money of what it still holds, and a take never pays twice for the same gram.
- **The layer is tied to the line of its purchase** (F5): the domain re-derives the layer from the line and refuses it when the money, the quantity or the ingredient differ, and two container tests break a real row to prove that the read of the stock refuses it.
- **The cost is DIRECT and says its name.** The labour (T13) and the overhead (T14) are not in it, and the field is called `direct` so that no layer of cost hides inside another one.
- **An answer that does not know says so.** An ingredient the stock holds in pieces, or in a package without a declared weight, has no rate per gram: the cost names it instead of guessing.

**L27** (2026-11-06): T13 landed: **the labor**. The labor is measured in hours and in three kinds — the hands, the oven and the waiting — and the hours of each kind are held apart. The **value of the hour is declared by the owner** and never inferred: the money of a labor record is its hours times that value, rounded to whole pesos, and the value of the hour of a business is the last one she declared. The **labor of a batch is derived, never typed**: every step of the recipe adds its minutes to the kind of labor it carries. The direct cost and the labor travel APART (`direct` and `labor` in the answer), so no layer of cost hides inside another one. The tool is `record_labor`, and the transport carries twenty-seven tools.

Three things the increment taught, carried here for the ones to come:

- **The labor is the owner's alone**: the model has no employee and no payroll, and a domain test pins it. The payroll of an employee is a model of its own, and it does not exist yet.
- **The money of a labor record is derived, never stored**: the table keeps the hours and the value of the hour, and the money is computed from them — a second truth about the same money would drift.
- **The labor is a fact of the business, not a movement of the book**: it moves no money of the ledger yet. The account of the labor exists in the seed of T2, and the decision of whether the labor writes a movement belongs to T14 or T15.

**L28** (2026-11-06): T14 landed: **the overhead**. The owner declares three facts and never a price: the electricity the house used, the hours the oven ran, and the share of that electricity that belongs to the oven. The money of the overhead is **derived, never typed and never stored**: the electricity times the share, rounded to whole pesos; the rate of the house is that money divided by the hours the oven ran; the overhead of a batch is the hours of the oven of the steps of the recipe times that rate. The overhead travels BESIDE the direct cost and beside the labor — three layers of the cost, none added to another. The tool is `record_overhead`, and the transport carries twenty-eight tools.

Three things the increment taught, carried here for the ones to come:

- **A rate is the last thing the owner declared**: the value of the hour of the labor and the rate of the house follow the same rule. The table keeps the history and the book values with the last declaration.
- **A batch of an earlier day is valued with the rate of today.** That is an approximation of the electricity of a period, and the rule is stated in the domain so that nobody believes a number it does not have.
- **The overhead moves no money of the ledger**: the account exists in the seed of T2, and the decision of writing the movement belongs to T15.

**Open for the owner**: the share of the electricity that belongs to the oven was read as a share (0 < share ≤ 1). Another shape — a money per hour of the oven, for example — is possible, and the change lives in the domain, in the argument of the tool and in the migration.

**L29** (2026-11-06): T15 landed: **the report of the real price**. The report shows the three layers of the cost of a batch — the direct cost of the ingredients, the labor of the hours of the recipe and the overhead of the hours of the oven — each with its money and its rates per unit, per gram and per kilogram, their sum shown APART, and the price of the owner beside them, with the way she charges it, exactly as she declared it. The tool is `read_real_price`, and the transport carries twenty-nine tools.

Three things the increment taught, carried here for the ones to come:

- **The price of the owner is a value object, not a number**: the money, and the way she charges it (per unit, per gram or per kilogram). The price of the unit and the price of the gram are two different truths and they never mix.
- **A sum of a batch that cannot be baked is a number nobody can use**: the total is unknown when a layer is unknown or when the stock does not cover the recipe, and the report says so instead of guessing.
- **Nothing of the report is stored**: the price of the owner is an argument of the tool, because the price list of the products and the channel are T25. The report says "the owner has not declared the price of the product" when she declared none.

**L30** (2026-11-06): T16 landed: **the margin**. The margin compares the **price of the owner** with the **direct cost** of the batch and with the **reference of the market**: the profit over the cost (per unit, per gram, per kilogram and the percent over the cost) and the profit over the market are two numbers of their own, and neither hides the other. The price is hers and the book never moves it; the reference never becomes the price of the ingredient and never moves money — what she paid comes from the layers of the stock. The tool is `read_margin`, and the transport carries thirty tools.

Three things the increment taught, carried here for the ones to come:

- **A reference travels with its source, its day and its age**: the newest of every ingredient is the one that compares, and the older ones are the history that a later analysis reads (S17).
- **Only the gram and the kilogram are converted** into the grams of a recipe: a package is converted only when the owner declared its weight, and a piece is never converted.
- **The margin compares the DIRECT cost only**: the percent over the direct cost is a markup over the ingredients, and the tool says so. The margin over the whole cost arrives when the layers are read together.

**Deviation**: the rule of the margin lives in `packages/domain/recipe` (the domain of `purchase` cannot import `recipe`, and the margin needs the yield of the recipe), and the age of a reference lives in `packages/domain/purchase`, as the acceptance expected.

**L31** (2026-11-06): T17 landed: **the production**. The production is **derived, never typed**: the owner names the recipe (by identity, or by the name she reads — a name that more than one recipe carries is refused, because the book never guesses), confirms it with her word, and the book derives the grams that went in, the money of the layers that gave them, and the units that came out. The production, the layers it consumed and the units it makes are written in ONE transaction. The tool is `record_production`, and the transport carries thirty-one tools.

Three things the increment taught, carried here for the ones to come:

- **The layers really leave the stock**: the production takes them in the order of the FIFO, and the quantity that is LEFT of every layer is written back. The write is guarded (`WHERE id = … AND unit = … AND remaining = held`), so a layer that changed under the production is refused instead of being overwritten.
- **The money of a take is the money that was left before minus the money that is left after**: a gram is never paid for twice, and the formula of T12 was wrong for a partly consumed layer — the new one is right for both.
- **The expiry of a produced unit is the day of the production plus the declared shelf life of the recipe** — approximate until the owner corrects it, and the correction is T18.

**Deviation**: the model of the production lives in `packages/domain/recipe`, not in a package of its own: the production needs the yield and the shelf life of the recipe, and the direction of the dependencies forbids `purchase` from reaching for `recipe`.

**L32** (2026-11-06): T18 landed: **the expiry of every unit**. The shelf life of a recipe stays the declared constant it was, but it is now **correctable**, and the correction is **kept as a row of its own**: the last correction the owner declared is the shelf life that counts. Only the owner corrects it, with the number of days she says (never inferred), the day she corrected it and her word of confirmation. Every unit of every production carries the day it expires, and a correction moves the units that were already made — in one transaction, guarded, so a unit that changed under the correction is refused. A unit that expired is never sold. Two tools landed: `correct_shelf_life` and `read_expiring`; the transport carries thirty-three tools.

Three things the increment taught, carried here for the ones to come:

- **A correction is a row, not an edit**: the history of what she declared is kept, and the last one wins. A shelf life that is only the current value cannot be explained later.
- **A correction moves what was already made**: the expiry of the units already in the book follows the corrected shelf life, guarded and written whole — nothing is half-corrected.
- **The rotation by expiry stays apart from the FIFO of the valuation**: `read_expiring` only orders by expiry; which unit leaves a production is T22.

**Two things the owner may want to say**: the unit expired ON the day it expires (a shelf life of N days are N good days), and the correction of a shelf life needs the tool that records it — a rule without a way to reach it is a rule nobody can use, so the increment landed two tools and not one.

**L33** (2026-11-06): T19 landed: **the waste**. The waste is a **movement of the book**: it carries the product, the quantity, the unit and the reason (burnt, spoiled, dropped or returned). It takes the units of the product that are still in the book and **expire first** — the rotation of the perishable output, which stays apart from the FIFO of the valuation — and it releases the cost of what it was worth, derived, never typed: the money of the production the unit came out of, divided by the units that production made, times the units that left, in whole pesos. The money is not lost: it leaves the inventory and lands in the account of the waste, in the same transaction as the waste itself. The tool is `record_waste`, and the transport carries thirty-four tools.

Three things the increment taught, carried here for the ones to come:

- **A unit that left the book never leaves again**: the unique index over the unit is the rule of the storage, and the domain refuses a waste of units that are not there.
- **The reason is a column of its own**: burnt, spoiled, dropped or returned — the reason is what a later analysis reads, and it is not buried in a sentence.
- **The waste counts whole units**: the book never converts a weight into a number of units and never splits a unit. When the owner names a weight, the agent asks her how many units she lost.

**A gap this increment exposed, for the checkpoint**: the production (T17) never moved the ledger, so the account of the inventory does not follow the valuation of the stock — the waste moves money out of an inventory that never received it. The rule of the movements of the production belongs to the increment that reads them together (T15 and T20 read the money), and it is an open item of the checkpoint.

**L34** (2026-11-08): T20 landed: **the sale**. The sale is a movement of the book, and it moves the money in and releases the cost of the units it sells, **in one transaction**: the money the client paid lands in the account that receives it — the cash of the business, or the account the owner names — and it comes from the account of the sales; the cost of the units that left the book leaves the inventory and lands in the account of the cost of the sales. The money of the sale is what the client paid, and it is never inferred: the price list of the products is T25, and the domain never guesses the money of a sale.

The sale takes the units of the product that are still in the book and **expire first** — the rotation of the perishable output, which stays apart from the FIFO of the valuation — and the sale of a unit that expired is refused: the expired unit leaves the book as a waste, and the book says so. The cost of the sale is derived, never typed: the money of the production the unit came out of, divided by the units that production made. The FIFO of the layers was already taken when the production consumed them, so the money of the production is already FIFO-valued, and the sale reads that money instead of recomputing a valuation of its own. The tool is `record_sale`, and the transport carries thirty-five tools.

Three things the increment taught, carried here for the ones to come:

- **The exits of the book live in one table now.** The sale is the second exit, and the rule "a unit that left the book never leaves again" has to hold across both of them: the storage is the defense in depth of that rule, and one table owns it. The table `unit_exits` replaced `waste_units`, and the migration moved the rows of the waste into it. The lesson is for the exits to come — the consumption and the donation (T21): every exit writes its units in the same table, and the unit leaves the book once, whichever exit takes it.
- **A refused sale does not skip the expired unit.** The rotation takes the units that expire first, and the sale refuses the whole one when one of them expired: taking a fresher unit instead would hide a loss that the book has to show.
- **The cost of the sale and the cost of the waste are the same derivation.** The money of the production divided by the units it made is the cost of a unit, and the sale and the waste sum it over the units they took. The derivation lives in one place, and the next exit reuses it instead of writing its own.

**A gap this increment exposed, for the checkpoint**: the production (T17) still never moved the ledger, so the account of the inventory does not follow the valuation of the stock — the sale and the waste move money out of an inventory that never received it. The rule of the movements of the production belongs to the increment that reads them together, and it is an open item of the checkpoint.
