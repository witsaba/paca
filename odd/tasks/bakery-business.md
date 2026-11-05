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

## The flow of the utterance

The design of T1, the first cut of the feature. It is the shape of the
increment T3 to T8, and the reason of every rule of them.

### Who reads and who decides

| The agent (the model in Hermes) | The domain (the packages of PACA) |
|---|---|
| Reads the sentence | Says what a purchase is |
| Proposes the lines, the quantities, the prices | Says what a purchase needs |
| Matches the spoken name against the catalog | Refuses the incomplete, the contradictory and the stale |
| Asks, and offers the options | States what is missing, and why |
| Shows the record in the owner's words | Writes the record, and keeps the sentence |
| Waits for the yes | Keeps the states, the revision and the expiry |

The boundary is the contract of the repository: the domain never receives a
sentence, and the model never decides what is true.

### The states of the draft

```
                ┌──────────────► cancelled
                │
    proposed ───┼──────────────► confirmed
                │
                └──────────────► expired   (seven days without an answer)
```

A draft that is confirmed becomes purchases: one transaction, all the lines
or none.

### The flow, in seven steps

1. **The sentence.** The owner says it in the conversation: "lets prepare the
   molde bread. I bought this flour and cost...". The sentence is kept
   verbatim, from the first moment to the record.
2. **The reading.** The agent extracts the lines: the name as she said it,
   the quantity, the unit, the money, the day, the market, the invoice. The
   money is only what she said (S42).
3. **The matching.** The agent calls `match_items` with the name she said.
   The answer is exact, ambiguous with its options, or unknown (S46).
4. **The proposal.** The agent calls `draft_purchase`. The domain validates
   the candidate, stores the draft with its state and its revision, and
   answers what it understood, what is missing, and what is ambiguous.
5. **The questions.** The agent asks only what the domain said is missing,
   and for an ambiguity it offers the options (S38, S39). Every question
   names the field it is about.
6. **The yes.** The agent shows the whole record in the owner's words, with
   the inferred fields marked as inferred, and waits. Her answer is kept
   with the record (S40).
7. **The writing and the answer.** The agent calls `confirm_purchase` with
   the revision she saw. The domain writes the purchases in one transaction
   and answers the record; the agent reports the three parts (S48).

### The tools and their contracts

| Tool | What it does | What it answers | What it never does |
|---|---|---|---|
| `match_items` | Matches the name she said against the catalog of the business | The candidates, with their identity and their name | It never writes |
| `draft_purchase` | Validates the candidate and keeps it as a draft | The draft, what is missing, what is ambiguous, what was inferred | It never writes the book |
| `confirm_purchase` | Writes the purchases of the draft, in one transaction | The purchases that were written | It never writes a draft that expired or that changed under her eyes |
| `cancel_draft` | Drops a draft she does not want | The cancelled draft | It never writes the book |
| `read_drafts` | Lists the pending drafts of the business | The drafts, with what is missing in each | It never writes |

Every description of these tools is derived from the domain, and every error
of the domain has a message written in the words of the caller, the way the
messages of the book are written today: the agent reads the message and knows
what to ask.

### The rules the domain enforces

- **The business is required.** Every draft belongs to a business; a draft of
  a business that does not exist is refused.
- **The amount is never inferred.** A draft without the money is missing it,
  and the answer says so.
- **The price is one truth.** The total paid and the price of the unit must
  agree; when they do not, the domain refuses and asks which one is right.
- **The revision must be the one she saw.** A confirmation of an old
  revision is refused: the record is shown again first.
- **The draft expires in seven days.** An expired draft is refused with the
  reason, and the purchase is dictated again.
- **The money is whole pesos.** Centavos are rounded to the nearest peso, and
  the sentence keeps the exact wording.
- **The sentence is never rewritten.** The record keeps it as it was said.

### The answer of Paca

Three parts, always: what it recorded, what it did not understand, and what
it still needs. The data comes from the domain; the wording is the agent's,
in the language of the owner.

### What the design leaves open

- The rounding of the centavos (S45) follows the rule of the book, and it is
  the one place where the money of the book may have to move to centavos.
- The seven days of the expiry are a rule of the domain and may become a
  setting of the business.
- The weight of a package is unknown until she says it: "two sacks" is a
  measure whose conversion to the gram comes with the catalog of the units.
- The catalog of the names begins empty: the first purchase creates the first
  name.

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
| T2 | S1, S2, S3, S36 | The registration of the bakery: the use case that opens a business of its own kind, the accounts of its own book, the tool that opens it and the tool that lists the businesses of the owner | inline | `pending` |

### Movement II — Paca writes the record

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T3 | S21, S37, S41, S42, S44 | The domain of the utterance: the draft, its states, its revision, its expiry, what is missing and what is ambiguous | inline | `pending` |
| T4 | S16, S19, S20, S43, S45 | The domain of the purchase: the line, the measure, the price, the money, the day, the market, the invoice, the sentence and the word of the confirmation | inline | `pending` |
| T5 | S46 | The domain of the catalog of the names the business buys, and the matching of a spoken name | inline | `pending` |
| T6 | S36, S41 | The tables of the drafts and of the purchases | inline | `pending` |
| T7 | S47, S48 | The tools: `match_items`, `draft_purchase`, `confirm_purchase`, `cancel_draft`, `read_drafts` | inline | `pending` |
| T8 | S38, S39, S40, S49 | The instructions of Paca in the soul: how she reads, asks, offers the options, confirms and reports | inline | `pending` |
| T9 | S11, S12, S19, S34 | The domain of the recipe: the ingredients with their grams and conversions, the steps with their times, temperatures and cues, the shelf life | inline | `pending` |
| T10 | S17, S18 | The research of the reference prices: public sources, their date, and the comparison against what the owner paid | inline | `pending` |

### Movement III — The cost

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T11 | S10, S13, S14, S43 | The domain of the ingredient and of the unit: the gram, the kilogram, the conversions, the packages, the allergens, and the two fats apart | inline | `pending` |
| T12 | S6, S16, S26 | The domain of the stock and of the FIFO layers: what the owner holds and what it cost her | inline | `pending` |
| T13 | S5, S6, S16 | The domain of the cost of a batch: derived from the layers, never typed | inline | `pending` |
| T14 | S5, S9 | The labor: the hands-on hours, the oven hours and the waiting apart, at the hourly value the owner declares | inline | `pending` |
| T15 | S5, S9 | The overhead: the electricity of the house allocated by the oven hours | inline | `pending` |
| T16 | S5, S10 | The report of the real price: the three layers apart, per unit, per gram and per kilogram | inline | `pending` |
| T17 | S10, S17, S18 | The margin against the price of the owner and against the reference of the market | inline | `pending` |

### Movement IV — From dough to bread

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T18 | S22 | The domain of the production: the recipe, the grams, the units it makes | inline | `pending` |
| T19 | S23 | The expiry of every unit, and the shelf life of the product behind it | inline | `pending` |
| T20 | S24, S25 | The waste: the exit, the quantity, and the reason | inline | `pending` |
| T21 | S24 | The sale: the revenue and the cost of the unit, in one transaction | inline | `pending` |
| T22 | S24 | The consumption by the owner and the donation: the two poles of the fork the owner named | inline | `pending` |
| T23 | S26 | The freshness first: the rotation by the earliest expiry, apart from the valuation | inline | `pending` |

### Movement V — The business around the bread

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T24 | S4 | The domain of the party: the person, the store with its legal human, the farm, and the roles | inline | `pending` |
| T25 | S2, S8 | The providers, the invoices they carry, and the tax that travels with them | inline | `pending` |
| T26 | S27 | The price list of the products and the channel that buys it | inline | `pending` |
| T27 | S2, S28 | The orders and the rounds of the delivery | inline | `pending` |
| T28 | S28 | The route of the round, computed by the agent, which is the only component with access to the outside | inline | `pending` |
| T29 | S2, S29 | The commitments of the owner and the reminders that fire them | inline | `pending` |
| T30 | S35 | The learning log: the errors and the wins of the owner, and what Paca teaches back | inline | `pending` |
| T31 | S9, S22 | The capacity of a weekend: the hours the owner has, and the units she can make with them | inline | `pending` |

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
