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
- **Scope:** the second business, the units of measure, the events of the business (the purchase, the production, the exit), the cost of a batch, the real price of a product, the parties, the orders and the reminders.
- **Out of scope, for now:** the payroll of employees, the multi-currency, the suggestions of improvement (the data is recorded for them; the suggestions themselves are a later work), and every piece of data of the owner (see S19).

---

## The method

The maintainer asked for the work to be divided the way Michelangelo divided
the marble (S30): **granular but incremental, never the whole in one session**.
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
kill the feature come first. The utterance (Movement I) is the feature itself
(S28): if Paca cannot turn a sentence into a record, nothing else matters. The
cost (Movement II) is the second risk: if the real price cannot be derived
from what was really paid, the whole accounting is a fiction. The tax work
(Movement IV) is heavy but unsurprising, so it comes last.

**The detail of a cut is written when the cut is made.** The map of every
increment is here; the spec of an increment is written at the start of its own
session, quoting the owner. A distant cut on unseasoned marble is how the
block breaks.

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

### The method

**S32 — The division of the work**

> use a similar method of the micelian works. Divide, and divide in order to have granular but incremental tasks. I dont want you work entirely in the bakery business in a single session but divided.

**S33 — The skeleton first, and the flow of the utterance designed before any code**

> use the 3

The flow of the utterance — what Paca infers, what it asks, what it refuses —
is designed in a session of its own, and this document is extended with it
before the first line of the feature is written.

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

### Movement I — Paca writes the record

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T1 | S20, S21, S33 | The flow of the utterance, designed: what Paca infers, what it asks, what it refuses, and what it answers back. A design, written into this document | inline | `pending` — `docs(odd): the flow of the utterance` |
| T2 | S16, S19, S20, S21 | The domain of the utterance: the candidate record, the ambiguity of an ingredient, the confirmation | inline | `pending` |
| T3 | S16, S18 | The domain of the purchase: the event, the price really paid, the day, the market, the invoice, and the sentence it came from | inline | `pending` |
| T4 | S3, S16 | The table of the purchases and the repository that implements the port | inline | `pending` |
| T5 | S16, S20 | The tool that records the purchase from the candidate, and the answer of Paca: what it recorded, what it did not understand, what it needs | inline | `pending` |
| T6 | S11, S12, S19, S34 | The domain of the recipe: the ingredients with their grams and conversions, the steps with their times, temperatures and cues, the shelf life | inline | `pending` |
| T7 | S18 | The research of the reference prices: public sources, their date, and the comparison against what the owner paid | inline | `pending` |

### Movement II — The cost

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T8 | S5, S10, S13, S14 | The domain of the ingredient and of the unit: the gram, the kilogram, the conversions, the allergens, and the two fats apart | inline | `pending` |
| T9 | S6, S16, S26 | The domain of the stock and of the FIFO layers: what the owner holds and what it cost her | inline | `pending` |
| T10 | S5, S6, S16 | The domain of the cost of a batch: derived from the layers, never typed | inline | `pending` |
| T11 | S5, S9, S31 | The labor: the hands-on hours, the oven hours and the waiting apart, at the hourly value the owner declares | inline | `pending` |
| T12 | S5, S9 | The overhead: the electricity of the house allocated by the oven hours | inline | `pending` |
| T13 | S5, S10 | The report of the real price: the three layers apart, per unit, per gram and per kilogram | inline | `pending` |
| T14 | S10, S17, S18 | The margin against the price of the owner and against the reference of the market | inline | `pending` |

### Movement III — From dough to bread

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T15 | S22, S23 | The domain of the production: the recipe, the grams, the units it makes | inline | `pending` |
| T16 | S23 | The expiry of every unit, and the shelf life of the product behind it | inline | `pending` |
| T17 | S24, S25 | The waste: the exit, the quantity, and the reason | inline | `pending` |
| T18 | S24 | The sale: the revenue and the cost of the unit, in one transaction | inline | `pending` |
| T19 | S24 | The consumption by the owner and the donation: the two poles of the fork the owner named | inline | `pending` |
| T20 | S26 | The freshness first: the rotation by the earliest expiry, apart from the valuation | inline | `pending` |

### Movement IV — The business around the bread

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T21 | S2, S4 | The domain of the party: the person, the store with its legal human, the farm, and the roles | inline | `pending` |
| T22 | S2, S8 | The providers, the invoices they carry, and the tax that travels with them | inline | `pending` |
| T23 | S27 | The price list of the products and the channel that buys it | inline | `pending` |
| T24 | S2, S28 | The orders and the rounds of the delivery | inline | `pending` |
| T25 | S28 | The route of the round, computed by the agent, which is the only component with access to the outside | inline | `pending` |
| T26 | S2, S29 | The commitments of the owner and the reminders that fire them | inline | `pending` |
| T27 | S35 | The learning log: the errors and the wins of the owner, and what Paca teaches back | inline | `pending` |
| T28 | S9, S22 | The capacity of a weekend: the hours the owner has, and the units she can make with them | inline | `pending` |

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
