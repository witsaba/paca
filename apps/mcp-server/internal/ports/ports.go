// Package ports holds the contracts of the hexagonal application: what comes
// in, and what goes out.
//
// The application layer depends on these interfaces only. The adapters
// implement them, and nothing in the middle knows which adapter it is talking
// to.
package ports

import (
	"context"
	"errors"

	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/commitment"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/user"
	healthports "github.com/witsaba/paca/packages/health/ports"
)

// ErrNoDatabase is the error of a use case with no database behind it.
//
// The service may run without a database: the tools are still there, and
// every call says that the database is not.
var ErrNoDatabase = errors.New("the service has no database configured")

// ErrCounterpartyRequired is the error of a debt that names nobody.
//
// A debt is the other side of the money: a loan of a friend, a product to
// pay later. A debt without the name of the other side is a bucket of
// money, and nobody waits for its payment.
var ErrCounterpartyRequired = errors.New("the debt needs the name of the other side of the money")

// ErrNotADebt is the error of a payment that pays something that is not a
// debt.
//
// The accounts of a book are many, and only some of them carry a debt: the
// cash does not, and neither does the salary.
var ErrNotADebt = errors.New("that account does not carry a debt")

// ErrInvalidMaxAgeDays is the error of an expectation of the age that is
// not a number of days.
//
// The owner says how old a reference of the market may be, and a negative
// number of days is not an age anybody expects.
var ErrInvalidMaxAgeDays = errors.New("the expectation of the age of a reference is not negative")

// HealthQuerier is the inbound port of the health of the service.
//
// The health has one owner: the shared health library. The service publishes
// the contract under its own name, and the contract it publishes is the one
// of the library.
type HealthQuerier = healthports.HealthQuerier

// Checker is the outbound port of the health of the service: it reports
// whether one component of the service is healthy.
//
// The health has one owner: the shared health library. The service publishes
// the contract under its own name, and the contract it publishes is the one
// of the library.
type Checker = healthports.Checker

// Pinger is the outbound port that asks the database whether it answers.
//
// The capability has one owner: the library that owns the database publishes
// the pool, and the pool answers the question. The service depends on the
// question and never on the handle that answers it.
type Pinger interface {
	// Ping returns nil when the database answers.
	Ping(ctx context.Context) error
}

// Clock is the outbound port that tells the application what time it is.
//
// The clock of the health and the clock of the book are the same clock, and
// the contract of the clock has one owner: the shared health library.
type Clock = healthports.Clock

// UsersQuerier is the inbound port of the tools of the people.
//
// The tools of the Model Context Protocol speak in text: a document ID, a
// name, a phone, an email. The use case speaks in the model of the domain,
// and the port is where the two meet.
type UsersQuerier interface {
	// Find returns the person who carries that document ID.
	//
	// The document ID is required: it is the number the person carries, and
	// nothing generates it.
	Find(ctx context.Context, documentID string) (user.User, error)

	// List returns the people of the storage, in the order of their
	// document ID.
	//
	// The limit is the number of people the caller wants to read: the
	// bound of the list belongs to the domain.
	List(ctx context.Context, limit int) ([]user.User, error)

	// ChangeStatus moves a user to another status and writes the
	// observation of the move.
	ChangeStatus(ctx context.Context, documentID, status, observation string) (user.User, error)

	// UpdateContactData writes the data of a user. A field that is empty is
	// a field that does not change.
	UpdateContactData(ctx context.Context, documentID, fullName, phone, email string) (user.User, error)
}

// Onboarding is the outcome of a complete onboarding: the person, the
// business that holds their money, and the accounts of that business.
//
// It is the answer of the tool, and it is the shape of the contract between
// the agent and the service.
type Onboarding struct {
	// User is the person, registered and active.
	User user.User

	// Business is the personal business of the person.
	Business business.Business

	// Accounts are the accounts the business starts with.
	Accounts []account.Account
}

// OnboardingCompleter is the inbound port of the tool that completes the
// onboarding of a person.
//
// The onboarding is one operation: the person is registered, their personal
// business is opened with its accounts, and they move to the state of the
// people who use the product.
type OnboardingCompleter interface {
	// Complete registers the person, opens their personal business with its
	// accounts, and moves them to the active status.
	//
	// The document ID and the name are required: they are the number the
	// person carries and the name the team recognises them by. The phone
	// and the email are optional.
	Complete(ctx context.Context, documentID, fullName, phone, email string) (Onboarding, error)
}

// RegisteredBusiness is the outcome of the opening of a business: the
// business, and the accounts of the book it starts with.
//
// It is the answer of the tool, and it is the shape of the contract between
// the agent and the service.
type RegisteredBusiness struct {
	// Business is the business that was opened.
	Business business.Business

	// Accounts are the accounts the book of the business starts with.
	Accounts []account.Account
}

// BusinessOpener is the inbound port of the tool that opens a business.
//
// A business of a person is the second business of that person: the
// Personal Business is opened by the onboarding, and this tool opens the
// others, the ones with a book of their own.
type BusinessOpener interface {
	// Open opens a business of the owner and writes the accounts of its
	// book.
	//
	// The document ID of the owner and the name of the business are
	// required: a business without a name has nothing to show in the book
	// of its owner. The currency is optional, and it is the currency of
	// the domain when the caller names none.
	//
	// The business and its accounts are written in one transaction, and
	// the error is business.ErrAlreadyExists when the name of the business
	// is taken inside the businesses of its owner.
	Open(ctx context.Context, documentID, name, currency string) (RegisteredBusiness, error)
}

// BusinessQuerier is the inbound port of the tool that reads the businesses
// of a person.
//
// A person may own more than one business, and the personal one is one of
// them.
type BusinessQuerier interface {
	// List returns the businesses of the owner, in the order of their
	// identity.
	//
	// The document ID of the owner is required, and the list is empty when
	// the person holds no business at all.
	List(ctx context.Context, documentID string) ([]business.Business, error)
}

// BookQuerier is the inbound port of the tools of the book.
//
// The tools of the Model Context Protocol speak in text and in numbers: the
// business, the two accounts, the amount and the day. The use case speaks in
// the model of the domain, and the port is where the two meet.
type BookQuerier interface {
	// Record writes a movement of the book.
	//
	// The money lands in the account of the destination and comes from the
	// account of the source. The day is the one the caller names, and the
	// day of today when the caller names none.
	Record(
		ctx context.Context,
		businessID business.ID,
		fromAccountID, toAccountID account.ID,
		amount ledger.Amount,
		date, description string,
	) (ledger.Entry, error)

	// Movements returns the movements of the book between two days.
	Movements(ctx context.Context, businessID business.ID, from, to string) ([]ledger.Entry, error)

	// Balances returns the money of the business and the balance of every
	// account of its book.
	Balances(ctx context.Context, businessID business.ID, asOf string) (ledger.Book, []ledger.Balance, error)
}

// DebtPayment is the outcome of the payment of a debt: the movement that
// paid it and the debt that is left.
type DebtPayment struct {
	// Movement is the movement that paid the debt.
	Movement ledger.Entry

	// Debt is the debt after the payment: what the person still owes.
	Debt ledger.Debt
}

// MonthSummary is the closing of a month: what the person earned, what
// they spent, and what they still owe.
type MonthSummary struct {
	// From is the first day of the month.
	From ledger.Date

	// To is the last day of the month.
	To ledger.Date

	// Summary is the money of the month.
	Summary ledger.Summary

	// Debts are the debts that were open at the end of the month.
	Debts []ledger.Debt
}

// DebtQuerier is the inbound port of the tools of the debts.
//
// The tools of the Model Context Protocol speak in text and in numbers: the
// business, the counterparty, the money and the day. The use case speaks in
// the model of the domain, and the port is where the two meet.
type DebtQuerier interface {
	// Record opens a debt and writes the money that opened it.
	//
	// The counterparty is REQUIRED: it is the other side of the money, and
	// a debt without it is a bucket of money. The money comes from the debt
	// and lands in the account the caller names: the cash when a friend
	// lends the money, the expense when a product is paid later.
	//
	// The day is the day the caller names, and the day of today when the
	// caller names none. The name of the account of the debt is optional,
	// and it is derived from the counterparty when the caller names none.
	Record(
		ctx context.Context,
		businessID business.ID,
		counterparty string,
		amount ledger.Amount,
		toAccountID account.ID,
		date, description, dueDate, name string,
	) (ledger.Debt, error)

	// Debts returns the debts the book still holds: what the person owes,
	// to whom, and the day it falls due.
	Debts(ctx context.Context, businessID business.ID) ([]ledger.Debt, error)

	// Pay writes the payment of a debt.
	//
	// The money leaves the account the caller names and lands in the debt,
	// and the payment may not pass what the debt still holds.
	Pay(
		ctx context.Context,
		businessID business.ID,
		debtAccountID account.ID,
		amount ledger.Amount,
		fromAccountID account.ID,
		date, description string,
	) (DebtPayment, error)
}

// PurchaseLine is a line the agent read from the sentence of the owner: the
// name as she said it, the quantity, the unit and the money.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
type PurchaseLine struct {
	// Name is the name as the owner said it.
	Name string

	// Quantity is how much of the thing the owner named.
	Quantity float64

	// Unit is the unit of that quantity.
	//
	// It is empty when the owner named none: the use case asks the
	// catalog of the names, and the unit of the ingredient of the line is
	// the unit of the line. The domain still refuses a line that ends
	// without a unit.
	Unit string

	// Total is the money that was paid for the whole line, in pesos.
	Total int64

	// UnitPrice is the price of one unit of the line, in pesos.
	UnitPrice int64

	// Inferred says that the price was filled by the agent and not said by
	// the owner. The money is never inferred, and the domain refuses it.
	Inferred bool

	// IngredientID is the identity of the ingredient of the catalog the
	// name belongs to.
	//
	// It is zero when the catalog does not know the name: the line keeps
	// the name as the owner said it.
	IngredientID int64
}

// PurchaseAmbiguity is a name the agent was not sure about, with the options
// it offers.
//
// Paca never asks an open question when it can name the options.
type PurchaseAmbiguity struct {
	// Name is the name as the owner said it.
	Name string

	// Options are the candidates the agent found.
	Options []string
}

// DraftPurchase is the candidate the agent read from the sentence of the
// owner, and the business it belongs to.
//
// It is the shape of the contract between the agent and the service: the
// sentence of the owner travels verbatim, and the lines travel as the
// agent read them.
type DraftPurchase struct {
	// BusinessID is the business the purchase belongs to.
	BusinessID business.ID

	// DraftID is the draft the candidate amends. It is zero when the
	// candidate is a new one.
	DraftID purchase.DraftID

	// Utterance is the sentence of the owner, verbatim.
	Utterance string

	// Lines are the lines the agent read.
	Lines []PurchaseLine

	// Day is the day the owner named. It is the day of today when the
	// agent names none.
	Day string

	// Market is the market the owner named. It is empty when she named
	// none.
	Market string

	// Invoice is the invoice the owner named. It is empty when she named
	// none.
	Invoice string

	// Ambiguities are the names the agent was not sure about.
	Ambiguities []PurchaseAmbiguity
}

// PurchaseDrafter is the inbound port of the tool that drafts a purchase.
//
// The tool proposes the candidate the agent read from the sentence of the
// owner, and the use case keeps it as a draft of the business. The draft
// waits for the yes of the owner: the tool never writes the book.
type PurchaseDrafter interface {
	// Draft keeps the candidate as a draft of the business, and answers
	// the draft with what it understood, what it still needs, what it was
	// not sure about and what it filled by itself.
	//
	// The identity of the business and the sentence of the owner are
	// required. The day, the market and the invoice are optional.
	//
	// The error is the error of the domain: the domain refuses a line
	// without a unit, a price that was inferred and a sentence that is
	// not there.
	Draft(ctx context.Context, request DraftPurchase) (purchase.Draft, error)
}

// ConfirmPurchase is the yes of the owner: the draft she confirmed, the
// revision she saw, the word she used and the account that pays.
//
// It is the shape of the contract between the agent and the service: the
// revision is what makes the confirmation exact, and the word is what is
// kept with the record.
type ConfirmPurchase struct {
	// BusinessID is the business the purchase belongs to.
	BusinessID business.ID

	// DraftID is the draft the owner confirmed.
	DraftID purchase.DraftID

	// Revision is the revision of the draft the owner saw.
	Revision int

	// ConfirmedWith is the word the owner used to confirm it.
	ConfirmedWith string

	// FromAccountID is the account the money of the payment comes from.
	// It is zero when the owner named none, and the money then comes from
	// the cash of the business.
	FromAccountID account.ID
}

// PurchaseConfirmer is the inbound port of the tool that confirms a draft.
//
// The tool writes the purchase of a draft the owner confirmed, and the
// money of the payment moves with it: the purchase and the movement of the
// money are written in one transaction, or none of them is.
type PurchaseConfirmer interface {
	// Confirm writes the purchase of the draft and answers with it and
	// with the three parts of the answer of Paca: what it recorded, what
	// it did not understand and what it still needs.
	//
	// The identity of the business, the identity of the draft, the
	// revision the owner saw and the word she used are required. The
	// account that pays is optional, and the cash of the business pays
	// when the owner names none.
	//
	// The error is the error of the domain: the domain refuses the
	// confirmation of an expired draft, of a draft that changed under the
	// eyes of the owner and of a draft that still needs the money of one
	// of its lines.
	Confirm(ctx context.Context, request ConfirmPurchase) (purchase.Confirmation, error)
}

// CancelDraft is the draft the owner dropped: the business it belongs to
// and the draft itself.
//
// It is the shape of the contract between the agent and the service: the
// draft belongs to its business, and the call names both.
type CancelDraft struct {
	// BusinessID is the business the draft belongs to.
	BusinessID business.ID

	// DraftID is the draft the owner does not want.
	DraftID purchase.DraftID
}

// PurchaseCanceller is the inbound port of the tool that cancels a draft.
//
// The tool drops the draft the owner does not want, and it answers with the
// draft it cancelled.
type PurchaseCanceller interface {
	// Cancel drops the draft the owner does not want, and answers with the
	// draft it cancelled.
	//
	// The identity of the business and the identity of the draft are
	// required.
	//
	// The error is the error of the domain: the domain refuses the
	// cancellation of a draft of another business, of a draft that is
	// confirmed already and of a draft that expired.
	Cancel(ctx context.Context, request CancelDraft) (purchase.Draft, error)
}

// ReadDrafts is the business whose pending drafts are read, and the day of
// the reading.
//
// It is the shape of the contract between the agent and the service: the
// day is optional, and it is the day of today when the agent names none.
type ReadDrafts struct {
	// BusinessID is the business whose drafts are read.
	BusinessID business.ID

	// Day is the day of the reading. It is the day of today when the
	// agent names none.
	Day string
}

// DraftsReader is the inbound port of the tool that reads the pending
// drafts of a business.
//
// The pending drafts are the ones that wait for the yes of the owner: the
// answer carries what each one still needs, what each one was not sure
// about and what Paca filled by herself.
type DraftsReader interface {
	// Read answers the pending drafts of the business, in the order of
	// their identity.
	//
	// The identity of the business is required, and the list is empty when
	// the business holds no pending draft.
	//
	// The error is the error of the domain: the domain refuses a day that
	// is not a day of the book.
	Read(ctx context.Context, request ReadDrafts) ([]purchase.Draft, error)
}

// MatchItems is the name the owner said, and the business whose catalog
// answers for it.
//
// It is the shape of the contract between the agent and the service: the
// name travels as she said it, and the catalog answers for it.
type MatchItems struct {
	// BusinessID is the business that bought.
	BusinessID business.ID

	// Name is the name as the owner said it.
	Name string
}

// ItemMatcher is the inbound port of the tool that matches a name against
// the catalog of a business.
//
// The catalog is the catalog of the names the business buys: it begins
// empty and grows with what the owner buys.
type ItemMatcher interface {
	// Match answers the name the owner said against the catalog of the
	// business: the ingredient it names, the options it offers when
	// several answer it, or the news that the catalog does not know it.
	//
	// The identity of the business and the name are required.
	//
	// The error is the error of the domain: the domain refuses a name
	// that is not there, and a catalog that carries two ingredients with
	// the same name.
	Match(ctx context.Context, request MatchItems) (ingredient.Match, error)
}

// ReadIngredients is the business whose catalog is read.
//
// It is the shape of the contract between the agent and the service: the
// catalog is the one of the business the agent names.
type ReadIngredients struct {
	// BusinessID is the business whose catalog is read.
	BusinessID business.ID
}

// IngredientsReader is the inbound port of the tool that reads the catalog
// of a business.
//
// The answer carries what is known of every ingredient of the catalog and
// what is still missing of each one.
type IngredientsReader interface {
	// Read answers the ingredients of the business, in the order of their
	// identity.
	//
	// The identity of the business is required, and the list is empty
	// when the business bought nothing yet: the catalog of a business
	// begins empty.
	//
	// The error is the error of the domain: the domain refuses a catalog
	// that carries two ingredients with the same name.
	Read(ctx context.Context, request ReadIngredients) ([]ingredient.Ingredient, error)
}

// CompleteIngredient is what the owner declares about an ingredient: the
// unit it is bought in, the weight of a package of it, the allergens it
// carries, the certification it has with the evidence of that
// certification, the names she uses for it and the conversion of its raw
// grams into its prepared ones.
//
// It is the shape of the contract between the agent and the service: every
// field is what the owner declared, and a field the owner did not declare
// stays as it was.
type CompleteIngredient struct {
	// BusinessID is the business the ingredient belongs to.
	BusinessID business.ID

	// IngredientID is the identity of the ingredient of the catalog.
	IngredientID int64

	// Unit is the unit the ingredient is bought in ('g', 'kg', 'paquete',
	// 'unidad'). It is empty when the owner declared none.
	Unit string

	// PackageGrams is the weight of a package of the ingredient, in
	// grams. It is zero when the owner declared none.
	PackageGrams float64

	// Allergens are the allergens the ingredient carries. It is nil when
	// the owner declared none.
	Allergens []string

	// CertificationName is the claim the ingredient carries ('Orgánico').
	// It is empty when the owner declared none.
	CertificationName string

	// CertificationEvidence is the document that proves the claim. It is
	// never empty when there is a claim.
	CertificationEvidence string

	// Aliases are the names the owner uses for the ingredient. It is nil
	// when she declared none.
	Aliases []string

	// Conversion is the grams of the raw that one gram of the prepared
	// ingredient takes. It is zero when the owner declared none, and the
	// ingredient keeps the one to one of the raw until then.
	Conversion float64
}

// IngredientCompleter is the inbound port of the tool that completes an
// ingredient.
//
// The tool writes what the owner declared about the thing the business
// buys, and answers with the ingredient as it is now.
type IngredientCompleter interface {
	// Complete writes what the owner declared about the ingredient of the
	// catalog, and answers with the ingredient as it is now.
	//
	// The identity of the business and the identity of the ingredient are
	// required. The unit, the weight of a package, the allergens, the
	// certification with its evidence, the aliases and the conversion are
	// optional, and the ingredient keeps what it had when the owner
	// declared none.
	//
	// The error is the error of the domain: the domain refuses an
	// ingredient the catalog of the business does not hold, a unit the
	// book does not know, a certification without its evidence and a
	// conversion that is not positive.
	Complete(ctx context.Context, request CompleteIngredient) (ingredient.Ingredient, error)
}

// RecipeIngredient is an ingredient of a recipe: the identity of the
// ingredient of the catalog and the grams of it.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
type RecipeIngredient struct {
	// IngredientID is the identity of the ingredient of the catalog.
	IngredientID int64

	// Grams is how many grams of it the recipe needs.
	Grams float64
}

// RecipeStep is one step of the process of a recipe.
//
// The instruction is what the owner does, the cue is the sign she looks
// for, and the time and the temperature are there only when she named
// them. The kind of the labor is there only when she named it too: the
// labor of the hands is the one a step carries when it names none.
type RecipeStep struct {
	// Instruction is what the owner does in this step.
	Instruction string

	// Cue is the sign the owner looks for.
	Cue string

	// Minutes is the time of the step, in minutes. It is zero when the
	// owner named none.
	Minutes int

	// Temperature is the temperature of the step, in degrees Celsius. It
	// is zero when the owner named none.
	Temperature float64

	// LaborKind is the kind of the labor the step is ('hands_on',
	// 'oven', 'waiting'). It is empty when the owner named none, and
	// the labor of the hands is the one a step carries then.
	LaborKind string
}

// RecordRecipe is the recipe the owner confirmed: the business it belongs
// to, its name, its ingredients, its steps, its yield, its shelf life and
// the word she used to confirm it.
//
// It is the shape of the contract between the agent and the service: the
// sentence of the owner travels verbatim when she said one, and the
// identity of every ingredient travels with the grams of it.
type RecordRecipe struct {
	// BusinessID is the business the recipe belongs to.
	BusinessID business.ID

	// Name is the name the owner reads.
	Name string

	// Ingredients are the ingredients of the recipe, with their grams.
	Ingredients []RecipeIngredient

	// Steps are the steps of the process, in the order the owner said
	// them.
	Steps []RecipeStep

	// YieldUnits is the number of units the recipe makes.
	YieldUnits int

	// YieldGrams is the weight of one unit, in grams.
	YieldGrams float64

	// ShelfLifeDays is the number of days what the recipe makes keeps.
	ShelfLifeDays int

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word she used to confirm the recipe.
	ConfirmedWith string
}

// RecipeRecorder is the inbound port of the tool that records a recipe.
//
// The tool writes the recipe with the confirmation of the owner, the way the
// purchase is written: the domain refuses a recipe that is not whole, and
// the answer tells the agent what it still needs to ask for.
type RecipeRecorder interface {
	// Record writes the recipe of the business and answers it with it.
	//
	// The identity of the business, the name of the recipe, its
	// ingredients with their grams, its steps, the yield, the shelf life
	// and the word of the owner are required. The sentence of the owner
	// is optional.
	//
	// The error is the error of the domain: the domain refuses a recipe
	// without a name, without a yield, with an ingredient without its
	// grams, with a step without its instruction, and with an ingredient
	// the catalog of the business does not hold.
	Record(ctx context.Context, request RecordRecipe) (recipe.Recipe, error)
}

// ReadStock is the business whose stock is read, and the ingredient the
// caller asks about.
//
// It is the shape of the contract between the agent and the service: the
// ingredient is optional, and the zero identity answers the whole
// business.
type ReadStock struct {
	// BusinessID is the business whose stock is read.
	BusinessID business.ID

	// IngredientID is the ingredient the caller asks about. It is zero
	// when the caller asks for the whole stock of the business.
	IngredientID int64
}

// StockReader is the inbound port of the tool that reads the stock of a
// business.
//
// The stock is the sum of the layers that are still open, and the
// valuation is FIFO: the oldest layer is the first one that leaves.
type StockReader interface {
	// Read answers what the business holds: the layers that are still
	// open, in the order they leave the stock, and the money they cost.
	//
	// The identity of the business is required, and the ingredient is
	// optional. The stock is empty when the business bought nothing yet.
	//
	// The error is the error of the domain: the domain refuses an
	// ingredient the catalog of the business does not hold.
	Read(ctx context.Context, request ReadStock) (purchase.Stock, error)
}

// ReadBatchCost is the recipe whose cost is read, and the business it
// belongs to.
//
// It is the shape of the contract between the agent and the service: the
// recipe is optional, and the zero identity and the empty name answer the
// cost of every recipe of the business.
type ReadBatchCost struct {
	// BusinessID is the business whose cost is read.
	BusinessID business.ID

	// RecipeID is the recipe the caller asks about. It is zero when the
	// caller names the recipe with its name, or when the caller asks for
	// every recipe of the business.
	RecipeID int64

	// RecipeName is the name the owner reads of the recipe. It is empty
	// when the caller names the recipe with its identity.
	RecipeName string
}

// BatchCostReader is the inbound port of the tool that reads the cost of a
// batch.
//
// The cost of a batch is derived from the layers that the recipe consumes,
// and never typed: the tool reads what the recipe costs right now.
type BatchCostReader interface {
	// Read answers the cost of the batches of the recipes of the business.
	//
	// The identity of the business is required. The recipe is optional:
	// the identity of a recipe answers the cost of that recipe, the name
	// of a recipe answers the cost of the recipes that carry it, and
	// neither of them answers the cost of every recipe of the business.
	//
	// The error is the error of the domain: the domain refuses a recipe
	// that is not there, a recipe of another business and a business that
	// is not there.
	Read(ctx context.Context, request ReadBatchCost) ([]recipe.Cost, error)
}

// ReadRealPrice is the recipe whose real price is read, the business it
// belongs to and the price the owner charges.
//
// It is the shape of the contract between the agent and the service: the
// price is what the owner declared, with the way she charges, and PACA
// never infers either of them.
type ReadRealPrice struct {
	// BusinessID is the business whose report is read.
	BusinessID business.ID

	// RecipeID is the recipe the caller asks about. It is zero when the
	// caller names the recipe with its name, or when the caller asks for
	// every recipe of the business.
	RecipeID int64

	// RecipeName is the name the owner reads of the recipe. It is empty
	// when the caller names the recipe with its identity.
	RecipeName string

	// Price is the money the owner charges for the product, in whole
	// pesos. It is zero when she declared none: the price of the owner is
	// never inferred.
	Price int64

	// PriceBasis is the way she charges: 'unit', 'gram' or 'kilogram'. It
	// is empty when she declared no price.
	PriceBasis string
}

// RealPriceReader is the inbound port of the tool that reads the report of
// the real price.
//
// The report shows the three layers of the cost apart - the direct cost,
// the labor and the overhead - per unit, per gram and per kilogram, and
// the price of the owner beside them.
type RealPriceReader interface {
	// Read answers the report of the real price of the products of the
	// business.
	//
	// The identity of the business is required. The recipe is optional:
	// the identity of a recipe answers the report of that recipe, the
	// name of a recipe answers the report of the recipes that carry it,
	// and neither of them answers the report of every recipe of the
	// business.
	//
	// The price of the owner is optional: it is what she declared, with
	// the way she charges, and a report of a price nobody declared says
	// that it is not known.
	//
	// The error is the error of the domain: the domain refuses a recipe
	// that is not there, a recipe of another business, a business that is
	// not there and a way of charging the book does not know.
	Read(ctx context.Context, request ReadRealPrice) ([]recipe.RealPrice, error)
}

// ReadMargin is the product whose margin is read, the business it belongs
// to, the price the owner charges and how old a reference may be.
//
// It is the shape of the contract between the agent and the service: the
// price is what the owner declared, with the way she charges, and PACA
// never infers either of them.
type ReadMargin struct {
	// BusinessID is the business whose margin is read.
	BusinessID business.ID

	// RecipeID is the recipe the caller asks about. It is zero when the
	// caller names the recipe with its name, or when the caller asks for
	// every recipe of the business.
	RecipeID int64

	// RecipeName is the name the owner reads of the recipe. It is empty
	// when the caller names the recipe with its identity.
	RecipeName string

	// Price is the money the owner charges for the product, in whole
	// pesos. It is zero when she declared none: the price of the owner is
	// never inferred.
	Price int64

	// PriceBasis is the way she charges: 'unit', 'gram' or 'kilogram'. It
	// is empty when she declared no price.
	PriceBasis string

	// MaxAgeDays is the number of days a reference of the market may be
	// old. It is zero when the owner expects no age: every reference
	// shows its age anyway.
	MaxAgeDays int
}

// MarginReader is the inbound port of the tool that reads the margin of a
// product.
//
// The margin compares the price of the owner with the direct cost of the
// batch and with the reference of the market, and every reference carries
// its source and its day.
type MarginReader interface {
	// Read answers the margin of the products of the business.
	//
	// The identity of the business is required. The recipe is optional:
	// the identity of a recipe answers the margin of that recipe, the
	// name of a recipe answers the margin of the recipes that carry it,
	// and neither of them answers the margin of every recipe of the
	// business.
	//
	// The price of the owner is optional: it is what she declared, with
	// the way she charges, and a margin of a price nobody declared says
	// that it is not known.
	//
	// The expectation of the age is optional: it is the number of days a
	// reference of the market may be old.
	//
	// The error is the error of the domain: the domain refuses a recipe
	// that is not there, a recipe of another business, a business that is
	// not there and a way of charging the book does not know.
	Read(ctx context.Context, request ReadMargin) ([]recipe.Margin, error)
}

// RecordLabor is the labor the owner declared: the hours she worked, in
// the three kinds of the labor, the value of the hour, the day it was
// worked and the word she used to confirm it.
//
// It is the shape of the contract between the agent and the service: the
// hours and the value of the hour are what she said, and PACA never infers
// either of them.
type RecordLabor struct {
	// BusinessID is the business the labor belongs to.
	BusinessID business.ID

	// HandsOnHours is the number of hours the hands of the owner were
	// on the dough.
	HandsOnHours float64

	// OvenHours is the number of hours the batch was in the oven.
	OvenHours float64

	// WaitingHours is the number of hours the dough waited.
	WaitingHours float64

	// HourValue is the value of the hour, in whole pesos. It is declared
	// by the owner, and it is never inferred.
	HourValue int64

	// HourValueInferred says that the value of the hour was filled by the
	// agent and not said by the owner.
	//
	// The money of the work of a person is never inferred, and the domain
	// refuses a labor that carries it: the way the line of a purchase
	// carries the money PACA filled by itself.
	HourValueInferred bool

	// Day is the day the labor was worked. It is the day of today when
	// the agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word the owner used to confirm the labor.
	ConfirmedWith string
}

// LaborRecorder is the inbound port of the tool that records the labor.
//
// The tool writes the hours the owner worked and the value of the hour she
// declared, and the domain decides what a labor is.
type LaborRecorder interface {
	// Record writes the labor of the business and answers it with it.
	//
	// The identity of the business, the hours of the labor, the value of
	// the hour and the word of the owner are required. The day and the
	// sentence of the owner are optional, and the day is the day of
	// today when the agent names none.
	//
	// The error is the error of the domain: the domain refuses a labor
	// without hours, without the value of the hour, without the day it
	// was worked and without the word of the owner, and it refuses a
	// labor whose value of the hour was filled by the agent.
	Record(ctx context.Context, request RecordLabor) (recipe.Labor, error)
}

// RecordOverhead is the overhead the owner declared: the electricity of the
// house, the hours the oven ran, the allocation of the electricity that
// belongs to the oven, the day it was declared and the word she used to
// confirm it.
//
// It is the shape of the contract between the agent and the service: the
// electricity and the allocation are what she said, and PACA never infers
// either of them.
type RecordOverhead struct {
	// BusinessID is the business the overhead belongs to.
	BusinessID business.ID

	// Electricity is the money of the electricity of the house, in whole
	// pesos. It is declared by the owner, and it is never inferred.
	Electricity int64

	// ElectricityInferred says that the electricity was filled by the
	// agent and not said by the owner.
	//
	// The money of the house is never inferred, and the domain refuses an
	// overhead that carries it: the way the line of a purchase carries
	// the money PACA filled by itself.
	ElectricityInferred bool

	// OvenHours is the number of hours the oven ran.
	OvenHours float64

	// Allocation is the share of the electricity of the house that belongs
	// to the oven.
	Allocation float64

	// Day is the day the overhead was declared. It is the day of today
	// when the agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word the owner used to confirm the overhead.
	ConfirmedWith string
}

// OverheadRecorder is the inbound port of the tool that records the
// overhead.
//
// The tool writes the electricity of the house, the hours of the oven and
// the allocation the owner declared, and the domain decides what an
// overhead is.
type OverheadRecorder interface {
	// Record writes the overhead of the business and answers it with it.
	//
	// The identity of the business, the electricity of the house, the
	// hours of the oven, the allocation and the word of the owner are
	// required. The day and the sentence of the owner are optional, and
	// the day is the day of today when the agent names none.
	//
	// The error is the error of the domain: the domain refuses an
	// overhead without the electricity of the house, without the hours of
	// the oven, without an allocation that is a share, without the day it
	// was declared and without the word of the owner, and it refuses an
	// overhead whose electricity was filled by the agent.
	Record(ctx context.Context, request RecordOverhead) (recipe.Overhead, error)
}

// SummaryQuerier is the inbound port of the tool that closes a month.
//
// The closing of a month is what the person earned, what they spent, and
// what they still owe: the three numbers a person asks for when they say
// "how did this month go?".
type SummaryQuerier interface {
	// Month reads the closing of a month: the income, the expenses, the
	// net, and the debts that were open at the end of it.
	//
	// The month is the one the caller names, and the month that just ended
	// when the caller names none.
	Month(ctx context.Context, businessID business.ID, year, month int) (MonthSummary, error)
}

// RecordReference is the reference the research found: the ingredient it is
// about, the price that was seen, the unit of that price, the market where
// it was seen, the place it was read, the day it was read and the word the
// owner used to confirm it.
//
// It is the shape of the contract between the agent and the service: a
// reference is a price fact of the business, and it is never the price of
// the ingredient.
type RecordReference struct {
	// BusinessID is the business the reference belongs to.
	BusinessID business.ID

	// IngredientID is the identity of the ingredient of the catalog the
	// reference is about.
	IngredientID int64

	// Price is the price that was seen, in whole pesos.
	Price int64

	// Unit is the unit of that price.
	Unit string

	// Market is the market where the price was seen. It is empty when the
	// research named none.
	Market string

	// Source is the place the price was read. It is required: a price
	// without a source is a price nobody can check.
	Source string

	// Day is the day the price was seen. It is the day of today when the
	// agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word the owner used to confirm it.
	ConfirmedWith string
}

// ReferenceRecorder is the inbound port of the tool that records a
// reference.
//
// The tool writes the reference with the confirmation of the owner, the way
// the purchase is written: the domain refuses a reference that is not
// whole, and the answer tells the agent what it still needs to ask for.
type ReferenceRecorder interface {
	// Record writes the reference of the business and answers it with it.
	//
	// The identity of the business, the identity of the ingredient, the
	// price, the unit, the source and the word of the owner are required.
	// The market and the sentence of the owner are optional, and the day
	// is the day of today when the agent names none.
	//
	// The error is the error of the domain: the domain refuses a
	// reference without the ingredient of the catalog, without a price,
	// without its unit, without its source, without its day and without
	// the word of the owner.
	Record(ctx context.Context, request RecordReference) (purchase.Reference, error)
}

// ReadReferences is the business whose references are read, and the
// ingredient the caller asks about.
//
// It is the shape of the contract between the agent and the service: the
// ingredient is optional, and the zero identity answers every reference of
// the business.
type ReadReferences struct {
	// BusinessID is the business whose references are read.
	BusinessID business.ID

	// IngredientID is the ingredient the caller asks about. It is zero
	// when the caller asks for every reference of the business.
	IngredientID int64
}

// ReferencesReader is the inbound port of the tool that reads the reference
// prices of a business.
//
// The references are read with their source and their day: a reference
// without them is a price nobody can check.
type ReferencesReader interface {
	// Read answers the references of the business, in the order of their
	// identity: the ones of one ingredient, or every one of them.
	//
	// The identity of the business is required, and the list is empty when
	// the business holds no reference.
	//
	// The error is the error of the domain: the domain refuses an
	// ingredient the catalog of the business does not hold.
	Read(ctx context.Context, request ReadReferences) ([]purchase.Reference, error)
}

// RecordProduction is the production the owner confirmed: the recipe she
// made, the day she made it and the word she used to confirm it.
//
// It is the shape of the contract between the agent and the service: the
// recipe and the day are what she said, and PACA never infers either of
// them. The grams that went in and the units that came out are derived by
// the domain: the book never asks the owner for a number it can read.
type RecordProduction struct {
	// BusinessID is the business the production belongs to.
	BusinessID business.ID

	// RecipeID is the identity of the recipe she made. It is zero when
	// the agent names the recipe with its name.
	RecipeID int64

	// RecipeName is the name the owner reads of the recipe she made. It is
	// empty when the agent names the recipe with its identity.
	RecipeName string

	// Day is the day the production was made. It is the day of today
	// when the agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word the owner used to confirm the production.
	ConfirmedWith string
}

// ProductionRecorder is the inbound port of the tool that records the
// production.
//
// The tool writes the production of the business and answers it with it: the
// recipe she made, the day, the grams that went in, the money of the layers
// the production consumed and the units that came out, each one with the
// day it expires.
//
// The identity of the business, the recipe and the word of the owner are
// required. The day and the sentence of the owner are optional, and the day
// is the day of today when the agent names none.
//
// The error is the error of the domain: the domain refuses a production
// without a recipe, without the day it was made and without the word of the
// owner, and it refuses a production the stock cannot cover.
type ProductionRecorder interface {
	// Record writes the production of the business and answers it with it.
	Record(ctx context.Context, request RecordProduction) (recipe.Production, error)
}

// CorrectShelfLife is the correction the owner declared of the shelf life of
// a recipe: the recipe she corrected, the number of days what it makes keeps
// and the word she used to confirm it.
//
// It is the shape of the contract between the agent and the service: the
// number of days is what she said, and PACA never infers it. The day of the
// expiry of the units is derived by the domain from that number.
type CorrectShelfLife struct {
	// BusinessID is the business the recipe belongs to.
	BusinessID business.ID

	// RecipeID is the identity of the recipe she corrected. It is zero
	// when the agent names the recipe with its name.
	RecipeID int64

	// RecipeName is the name the owner reads of the recipe she corrected.
	// It is empty when the agent names the recipe with its identity.
	RecipeName string

	// ShelfLifeDays is the number of days what the recipe makes keeps,
	// the way she corrected it.
	ShelfLifeDays int

	// Day is the day the correction was declared. It is the day of today
	// when the agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word the owner used to confirm the correction.
	ConfirmedWith string
}

// CorrectedShelfLife is the outcome of the correction of the shelf life:
// the correction that was written and the units whose day of expiry moved.
//
// It is the answer of the tool, and it is the shape of the contract between
// the agent and the service.
type CorrectedShelfLife struct {
	// Correction is the correction that was written.
	Correction recipe.ShelfLifeCorrection

	// Units are the units whose day of expiry moved: the day each one
	// expired on before the correction and the day it expires on now.
	Units []recipe.CorrectedUnit
}

// ShelfLifeCorrector is the inbound port of the tool that corrects the
// shelf life of a recipe.
//
// The tool writes the correction the owner confirmed and answers it with it:
// the recipe, the number of days, the day it was declared and the units
// whose day of expiry moved.
//
// The identity of the business, the recipe, the number of days and the word
// of the owner are required. The day and the sentence of the owner are
// optional, and the day is the day of today when the agent names none.
//
// The error is the error of the domain: the domain refuses a correction
// without days, without the day it was declared and without the word of the
// owner.
type ShelfLifeCorrector interface {
	// Correct writes the correction of the shelf life and answers it with
	// it, and with the units whose day of expiry moved.
	Correct(ctx context.Context, request CorrectShelfLife) (CorrectedShelfLife, error)
}

// ReadExpiring is the business whose expiry is read.
//
// It is the shape of the contract between the agent and the service: the
// business is the only thing the agent names, and the days and the expiry
// of every unit are read by the book.
type ReadExpiring struct {
	// BusinessID is the business whose expiry is read.
	BusinessID business.ID
}

// ExpiringReader is the inbound port of the tool that reads what expires.
//
// The units are read with the day they expire and the days that are left:
// the owner asks what expires and when, and the book answers with the day
// of every unit of the business.
type ExpiringReader interface {
	// Read answers the units of the business in the order they expire,
	// with the day the answer was read on.
	//
	// The identity of the business is required, and the list is empty
	// when the business made no production yet.
	//
	// The error is the error of the domain: the domain refuses the
	// business that is not there.
	Read(ctx context.Context, request ReadExpiring) (recipe.Expiry, error)
}

// ReadRotation is the business whose rotation is read, and the product the
// caller asks about.
//
// It is the shape of the contract between the agent and the service: the
// product is optional, and the zero identity with the empty name answer the
// rotation of the whole book of the business.
type ReadRotation struct {
	// BusinessID is the business whose rotation is read.
	BusinessID business.ID

	// RecipeID is the identity of the product the caller asks about. It is
	// zero when the caller names the product with its name, or when the
	// caller asks for the whole book of the business.
	RecipeID int64

	// RecipeName is the name the owner reads of the product. It is empty
	// when the caller names the product with its identity.
	RecipeName string
}

// RotationReader is the inbound port of the tool that reads the rotation.
//
// The rotation is the order in which the units of the book leave it: the
// unit that expires first is the first one that leaves. The valuation of
// the stock follows the FIFO, and it is another rule: the answer of the
// rotation carries no money.
type RotationReader interface {
	// Read answers the units of the book in the order they leave it, with
	// the day each one expires and the days that are left of it.
	//
	// The identity of the business is required, and the product is
	// optional. The rotation holds only the units that are still in the
	// book, and it is empty when the business made no production yet.
	//
	// The error is the error of the domain: the domain refuses the
	// business that is not there, and the product the business does not
	// hold.
	Read(ctx context.Context, request ReadRotation) (recipe.Rotation, error)
}

// RecordWaste is the waste the owner confirmed: the product she lost, the
// quantity she lost with its unit, the reason, the day it happened and the
// word she used to confirm it.
//
// It is the shape of the contract between the agent and the service: the
// quantity, the unit and the reason are what she said, and PACA never
// infers any of them. The money of the waste is derived by the book: it is
// the cost the waste releases, and it is never typed.
type RecordWaste struct {
	// BusinessID is the business the waste belongs to.
	BusinessID business.ID

	// RecipeID is the identity of the product she lost. It is zero when
	// the agent names the product with its name.
	RecipeID int64

	// RecipeName is the name the owner reads of the product she lost. It
	// is empty when the agent names the product with its identity.
	RecipeName string

	// Quantity is how many units of the product the bakery lost.
	Quantity float64

	// Unit is the unit of that quantity. It is empty when the owner named
	// none, and the waste counts the units it lost then.
	Unit string

	// Reason is why the bakery lost them: 'burnt', 'spoiled', 'dropped'
	// or 'returned'.
	Reason string

	// Day is the day the waste happened. It is the day of today when the
	// agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word she used to confirm the waste.
	ConfirmedWith string
}

// WasteRecorder is the inbound port of the tool that records the waste.
//
// The tool writes the waste of the business and answers it with it: the
// product, the quantity, the unit, the reason, the money of what it was
// worth and the units that left the book.
//
// The identity of the business, the product, the quantity, the reason and the
// word of the owner are required. The unit, the day and the sentence of the
// owner are optional, and the day is the day of today when the agent names
// none.
//
// The error is the error of the domain: the domain refuses a waste without
// a product, without a whole number of units, without a reason of the book,
// without the day it happened and without the word of the owner, and it
// refuses a waste of units the book does not hold.
type WasteRecorder interface {
	// Record writes the waste of the business and answers it with it.
	Record(ctx context.Context, request RecordWaste) (recipe.Waste, error)
}

// RecordSale is the sale the owner confirmed: the product she sold, the
// quantity she sold with its unit, the money the client paid, the account
// that receives the money, the day it happened and the word she used to
// confirm it.
//
// It is the shape of the contract between the agent and the service: the
// quantity, the unit and the money are what she said, and PACA never
// infers any of them. The cost of the sale is derived by the book: it is
// the cost the sale releases, and it is never typed.
type RecordSale struct {
	// BusinessID is the business the sale belongs to.
	BusinessID business.ID

	// RecipeID is the identity of the product she sold. It is zero when
	// the agent names the product with its name.
	RecipeID int64

	// RecipeName is the name the owner reads of the product she sold. It
	// is empty when the agent names the product with its identity.
	RecipeName string

	// Quantity is how many units of the product the bakery sold.
	Quantity float64

	// Unit is the unit of that quantity. It is empty when the owner named
	// none, and the sale counts the units it sold then.
	Unit string

	// Money is the money the client paid, in whole pesos.
	Money int64

	// ReceivingAccountID is the account the money of the sale lands in.
	// It is zero when the owner named none, and the money then lands in
	// the cash of the business.
	ReceivingAccountID account.ID

	// Day is the day the sale happened. It is the day of today when the
	// agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word she used to confirm the sale.
	ConfirmedWith string
}

// SaleRecorder is the inbound port of the tool that records the sale.
//
// The tool writes the sale of the business and answers it with it: the
// product, the quantity, the unit, the money the client paid, the cost the
// sale released and the units that left the book.
//
// The identity of the business, the product, the quantity, the money and
// the word of the owner are required. The unit, the account that receives
// the money, the day and the sentence of the owner are optional, and the
// day is the day of today when the agent names none.
//
// The error is the error of the domain: the domain refuses a sale without a
// product, without a whole number of units, without money, without the day
// it happened and without the word of the owner, and it refuses the sale of
// a unit that expired.
type SaleRecorder interface {
	// Record writes the sale of the business and answers it with it.
	Record(ctx context.Context, request RecordSale) (recipe.Sale, error)
}

// RecordExit is the exit the owner confirmed: the consumption of the owner
// or the donation, the product, the quantity she took with its unit, the
// day it happened and the word she used to confirm it.
//
// It is the shape of the contract between the agent and the service: the
// kind of the exit, the quantity and the unit are what she said, and PACA
// never infers any of them. The cost of the exit is derived by the book: it
// is the cost the exit releases, and it is never typed.
type RecordExit struct {
	// BusinessID is the business the exit belongs to.
	BusinessID business.ID

	// RecipeID is the identity of the product she consumed or gave away.
	// It is zero when the agent names the product with its name.
	RecipeID int64

	// RecipeName is the name the owner reads of the product she consumed
	// or gave away. It is empty when the agent names the product with its
	// identity.
	RecipeName string

	// Kind is the kind of the exit: 'consumption' or 'donation'.
	Kind string

	// Quantity is how many units of the product the exit took.
	Quantity float64

	// Unit is the unit of that quantity. It is empty when the owner named
	// none, and the exit counts the units it took then.
	Unit string

	// Day is the day the exit happened. It is the day of today when the
	// agent names none.
	Day string

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word she used to confirm the exit.
	ConfirmedWith string
}

// ExitRecorder is the inbound port of the tool that records the exit.
//
// The tool writes the exit of the business and answers it with it: the kind
// of the exit, the product, the quantity, the unit, the cost it released,
// the tax treatment it waits for and the units that left the book.
//
// The identity of the business, the product, the kind of the exit, the
// quantity and the word of the owner are required. The unit, the day and
// the sentence of the owner are optional, and the day is the day of today
// when the agent names none.
//
// The error is the error of the domain: the domain refuses an exit without
// a product, without a whole number of units, without a kind of the book,
// without the day it happened and without the word of the owner, and it
// refuses an exit of units the book does not hold.
type ExitRecorder interface {
	// Record writes the exit of the business and answers it with it.
	Record(ctx context.Context, request RecordExit) (recipe.Exit, error)
}

// OpenParty is the party the owner wants in the book of the business: the
// person, the store or the farm that buys from the bakery or that the
// bakery buys from.
//
// It is the shape of the contract between the agent and the service: the
// legal human that represents a store or a farm travels as a name and as
// its legal figure, and it is a fact of the party and never a second
// person.
type OpenParty struct {
	// BusinessID is the business that knows the party and that owns it.
	//
	// It is the business of its own: the Personal Business of a person
	// holds the money of that person, and not the clients and the
	// providers of the bakery.
	BusinessID int64

	// Kind is the kind of the party: 'person', 'store' or 'farm'.
	Kind string

	// Name is the name the owner reads.
	//
	// It is not unique inside the business: two parties may share it.
	Name string

	// Roles are the roles the party holds: 'client', 'provider', or both
	// of them.
	Roles []string

	// Phone is the phone number of the party. It is optional.
	Phone string

	// Email is the email address of the party. It is optional.
	Email string

	// Address is the address of the party. It is optional.
	Address string

	// LegalHumanName is the name of the human that represents the party.
	// It is required for a store, and it is refused for a person.
	LegalHumanName string

	// LegalFigure is the legal figure the human has. It is optional, and
	// it is the figure of the farm that has one.
	LegalFigure string

	// TaxID is the tax identity of the party: the NIT of a person, of a
	// store or of a farm. It is optional, and PACA never assumes one.
	TaxID string
}

// PartyOpener is the inbound port of the tool that opens a party.
//
// The tool writes the party of the business and answers it with it: the
// kind, the name, the roles and the facts of the party.
type PartyOpener interface {
	// Open writes the party of the business and answers it with it.
	//
	// The identity of the business, the kind, the name and the roles are
	// required. The phone, the email, the address, the name of the legal
	// human and its legal figure are optional, and the kind of the party
	// decides what the legal human may be.
	//
	// The error is the error of the domain: the domain refuses a party of
	// the personal business of a person, a party without a name, a party
	// that holds no role, a store without the legal human that represents
	// it and a person that carries one.
	Open(ctx context.Context, request OpenParty) (party.Party, error)
}

// ReadParties is the business whose parties are read.
//
// It is the shape of the contract between the agent and the service: the
// business is the only thing the agent names.
type ReadParties struct {
	// BusinessID is the business whose parties are read.
	BusinessID int64
}

// RecordInvoice is the invoice the owner shows: the party that issued it,
// the purchase it came from, its number, its day, the base the tax applies
// to, the rate the invoice states, its total and the reference of the
// document the owner shows.
//
// It is the shape of the contract between the agent and the service: the
// base and the rate are what the invoice states, and PACA never infers
// either of them. The total is what the invoice says too, and PACA refuses
// an invoice that does not add up.
type RecordInvoice struct {
	// BusinessID is the business that bought and that holds the invoice.
	BusinessID business.ID

	// ProviderID is the identity of the party that issued the invoice:
	// the farm, the store or the person the business buys from. It holds
	// the role of the provider.
	ProviderID int64

	// PurchaseID is the identity of the purchase the invoice came from.
	// The purchase points at the invoice.
	PurchaseID int64

	// Number is the number the invoice carries. It is what identifies it
	// inside the business that holds it.
	Number string

	// Day is the day the invoice was issued. It is the day of today when
	// the agent names none.
	Day string

	// Base is the money the tax applies to, in whole pesos.
	Base int64

	// Rate is the rate the invoice was issued with, in percent: 19, 19.5,
	// 0. It is never assumed, and PACA never fills it by itself.
	Rate float64

	// Total is the money of the whole invoice, in whole pesos. It is the
	// base plus the tax, and an invoice that does not add up is refused.
	Total int64

	// Reference is the reference of the document the owner shows. It is
	// optional, and it is never the mark of an electronic invoice.
	Reference string
}

// InvoiceRecorder is the inbound port of the tool that records an invoice.
//
// The tool writes the invoice against the purchase it came from, and the
// purchase points at it. The invoice moves NO money: the purchase already
// moved the money of the payment when the owner bought.
type InvoiceRecorder interface {
	// Record writes the invoice of the business and answers it with it.
	//
	// The identity of the business, the identity of the provider, the
	// identity of the purchase, the number, the base, the rate and the
	// total are required. The day is optional, and it is the day of
	// today when the agent names none. The reference is optional.
	//
	// The error is the error of the domain: the domain refuses a
	// provider that does not hold the role of the provider, an invoice
	// that does not add up, an invoice of another business and a number
	// the business already holds.
	Record(ctx context.Context, request RecordInvoice) (purchase.Invoice, error)
}

// ReadInvoices is the business whose invoices are read.
//
// It is the shape of the contract between the agent and the service: the
// business is the only thing the agent names.
type ReadInvoices struct {
	// BusinessID is the business whose invoices are read.
	BusinessID business.ID
}

// InvoicesReader is the inbound port of the tool that reads the invoices of
// a business.
//
// The answer carries the number of every invoice, its day, its base, its
// rate, its tax and its total: the tax of an invoice is never assumed, and
// it comes from the invoice.
type InvoicesReader interface {
	// Read answers the invoices of the business, in the order of their
	// identity.
	//
	// The identity of the business is required, and the list is empty
	// when the business holds no invoice.
	//
	// The error is the error of the domain.
	Read(ctx context.Context, request ReadInvoices) ([]purchase.Invoice, error)
}

// PartiesReader is the inbound port of the tool that reads the parties of a
// business.
//
// The answer carries every party of the business: the name is not unique
// inside it, and the ones that share it are all of them.
type PartiesReader interface {
	// Read answers the parties of the business, in the order of their
	// identity.
	//
	// The identity of the business is required, and the list is empty
	// when the business knows no party yet.
	//
	// The error is the error of the domain.
	Read(ctx context.Context, request ReadParties) ([]party.Party, error)
}

// SetPrice is the price the owner declares for a product of a channel.
//
// It is the shape of the contract between the agent and the service: the
// product is named by the recipe it comes from or by the name the owner
// reads, and the channel is the way it is bought.
type SetPrice struct {
	// BusinessID is the business the price list belongs to.
	BusinessID business.ID

	// RecipeID is the identity of the recipe the product comes from. It
	// is zero when the agent names the recipe with its name, or when it
	// names the product with the name the owner reads.
	RecipeID int64

	// RecipeName is the name of the recipe the product comes from. It is
	// empty when the agent names the recipe with its identity, or when it
	// names the product with the name the owner reads.
	RecipeName string

	// Name is the name the owner reads of the product. It is empty when
	// the agent names the product with its recipe.
	Name string

	// Channel is the channel the price was set for: 'retail' or
	// 'wholesale'. It is required, and there is no default channel.
	Channel string

	// Basis is the way the owner charges: 'unit', 'gram' or 'kilogram'.
	// It is required, and there is no default: the money of a price is
	// only a price when the book knows what it is the price of.
	Basis string

	// Amount is the money of the product in that channel, in whole pesos.
	// It is required, and it is never inferred.
	Amount int64

	// TaxTreatment is the tax treatment the owner stated for the product.
	// It is optional: an empty one leaves the treatment of the product as
	// it was, and PACA never derives it from the price.
	TaxTreatment string

	// Day is the day the price was set. It is the day of today when the
	// agent names none.
	Day string
}

// PriceRecorder is the inbound port of the tool that sets the price of a
// product.
//
// The tool writes the price of the product for the channel, and it opens
// the product the first time it is named: the product of the bakery comes
// into being the day the owner puts a price on it.
type PriceRecorder interface {
	// Set writes the price of the product for the channel, and answers it
	// with the product and the price.
	//
	// The identity of the business, the channel and the money are
	// required. The product is named by the recipe it comes from or by
	// the name the owner reads. The day is optional, and it is the day of
	// today when the agent names none.
	//
	// The error is the error of the domain: the domain refuses a price
	// that is not a positive number of pesos, a channel it does not know
	// and a product whose name or recipe is taken.
	Set(ctx context.Context, request SetPrice) (product.Listed, error)
}

// ReadPrices is the business whose price list is read, and the day the
// list is read at.
//
// It is the shape of the contract between the agent and the service: the
// day is optional, and it is the day of today when the agent names none.
type ReadPrices struct {
	// BusinessID is the business whose price list is read.
	BusinessID business.ID

	// RecipeID is the identity of the recipe the product comes from. It
	// is zero when the agent names the recipe with its name, or when it
	// asks for the whole price list.
	RecipeID int64

	// RecipeName is the name of the recipe the product comes from. It is
	// empty when the agent names the recipe with its identity, or when it
	// asks for the whole price list.
	RecipeName string

	// Name is the name the owner reads of the product. It is empty when
	// the agent asks for the whole price list.
	Name string

	// Channel is the channel the price list is read for. It is empty
	// when the agent asks for the whole price list.
	Channel string

	// Day is the day the price list is read at. It is the day of today
	// when the agent names none.
	Day string
}

// PricesReader is the inbound port of the tool that reads the price list
// of a business.
//
// The answer carries the price that holds for every product and every
// channel on the day the list is read at: the last price set on or before
// that day.
type PricesReader interface {
	// Read answers the price list of the business, in the order of the
	// products and of the channels.
	//
	// The identity of the business is required. The product, the channel
	// and the day are optional: the day is the day of today when the
	// agent names none, and the product and the channel answer the whole
	// list when the agent names none.
	//
	// The error is the error of the domain: the domain refuses a day that
	// is not a day of the book, a channel it does not know and a product
	// the business does not hold.
	Read(ctx context.Context, request ReadPrices) ([]product.Listed, error)
}

// RecordOrder is the order the owner wants in the book of the business:
// the party that ordered, the day it is due, and the lines with the
// products and the quantities of them.
//
// It is the shape of the contract between the agent and the service: the
// party is named by its identity or by the name the owner reads, and the
// product of a line is named the same way.
type RecordOrder struct {
	// BusinessID is the business the order belongs to.
	BusinessID business.ID

	// PartyID is the identity of the party that ordered: the client of
	// the business. It is zero when the agent names the party with the
	// name the owner reads.
	PartyID int64

	// PartyName is the name the owner reads of the party that ordered.
	// It is empty when the agent names the party with its identity.
	PartyName string

	// DueOn is the day the order is due on. It is required: the order
	// carries the day it is due.
	DueOn string

	// Lines are the lines of the order: the products with the quantities
	// of them and the units the owner said.
	Lines []OrderLine
}

// OrderLine is one line of the order the agent read: the product, the
// quantity of it and the unit the owner said.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
type OrderLine struct {
	// ProductID is the identity of the product the line is of. It is zero
	// when the agent names the product with the name the owner reads.
	ProductID int64

	// ProductName is the name the owner reads of the product. It is empty
	// when the agent names the product with its identity.
	ProductName string

	// Quantity is how much of the product the line carries. It is the
	// number of pieces when the owner charges by the piece, and the amount
	// of weight when she charges by the weight.
	Quantity float64

	// Unit is the unit of that quantity, the way the owner said it: the
	// domain reads it in the vocabulary of the book.
	Unit string
}

// OrderRecorder is the inbound port of the tool that records an order.
//
// The tool writes the order of the business with the lines of it, and it
// answers with the order it wrote.
type OrderRecorder interface {
	// Record writes the order of the business and answers it with it.
	//
	// The identity of the business, the party that ordered, the day it
	// is due and the lines of the order are required. The party and the
	// product of a line are named by their identity or by the name the
	// owner reads.
	//
	// The error is the error of the domain: the domain refuses the
	// order without a day, without a line, the party that is not a
	// client, the party of another business and the product of another
	// business.
	Record(ctx context.Context, request RecordOrder) (order.Order, error)
}

// ReadOrders is the business whose orders are read, and the day of the
// round that is read.
//
// It is the shape of the contract between the agent and the service: the
// day is optional, and the whole book of the orders is read when the agent
// names none.
type ReadOrders struct {
	// BusinessID is the business whose orders are read.
	BusinessID business.ID

	// Day is the day of the round that is read. It is empty when the
	// agent asks for the whole book of the orders.
	Day string
}

// OrdersReader is the inbound port of the tool that reads the orders of a
// business.
//
// The answer carries the orders the round of the day carries and the ones
// that are still owed: every open order of the business, the ones that are
// due on the day and the ones that fell due on an earlier day.
type OrdersReader interface {
	// Read answers the round of the day: the orders it carries and the
	// ones that are still owed.
	//
	// The identity of the business is required. The day is optional: the
	// round of the day answers the orders that are due on it, and the
	// whole book of the orders is read when the agent names none.
	//
	// The error is the error of the domain: the domain refuses the
	// business that is not there, the business that does not hold orders
	// and the day that is not a day of the book.
	Read(ctx context.Context, request ReadOrders) (order.Round, error)
}

// RecordRoute is the route the agent computed for the round of a day.
//
// It is the shape of the contract between the agent and the service: the
// agent is the one that knows the streets, and it writes back the order it
// will visit the clients in and what each visit costs.
type RecordRoute struct {
	// BusinessID is the business the route belongs to.
	BusinessID business.ID

	// Day is the day of the round the route is of. It is required: the
	// route is written back into the round of that day.
	Day string

	// Stops are the stops of the route, in the order the agent will visit
	// the clients in.
	Stops []RouteStop
}

// RouteStop is one stop of the route the agent read: the order the stop
// visits and what the stop costs.
//
// The port speaks in the words of the agent and in numbers: the use case
// turns them into the model of the domain, and the domain decides what may
// enter the book.
type RouteStop struct {
	// OrderID is the identity of the order the stop visits.
	OrderID int64

	// Cost is what the stop costs: the transport to that client, in whole
	// pesos. It is never negative, and it may be zero: a stop the owner
	// walks to costs nothing.
	Cost int64
}

// RouteRecorder is the inbound port of the tool that records the route.
//
// The tool writes the route of the round of a day, and the route delivers
// the orders of its stops.
type RouteRecorder interface {
	// Record writes the route of the round of the day and answers it with
	// it.
	//
	// The identity of the business, the day of the round and the stops of
	// the route are required.
	//
	// The error is the error of the domain: the domain refuses the
	// business that does not hold orders, the route without a day, the
	// route without a stop, the stop that is not an order the business
	// owes, the same order in two stops, the cost that is less than
	// nothing and the second route of a day that was already written.
	Record(ctx context.Context, request RecordRoute) (order.Route, error)
}

// RecordCommitment is the promise the agent read: what is due, in the words
// of the owner, and the day it is due.
//
// It is the shape of the contract between the agent and the service: the
// promise carries no money and no party of its own, and the fields the
// domain refuses travel as the agent sent them, so the domain is the one
// that answers.
type RecordCommitment struct {
	// BusinessID is the business that owes the promise.
	BusinessID business.ID

	// What is what is due, in the words of the owner.
	What string

	// DueOn is the day the promise is due on, written the way a person
	// writes it.
	DueOn string
}

// MeetCommitment is the promise that was kept and the day it was kept.
//
// The commitment is named by its OWN identity, never by its text: two
// promises that say the same thing on the same day are two promises.
type MeetCommitment struct {
	// BusinessID is the business that owes the promise.
	BusinessID business.ID

	// CommitmentID is the identity of the promise that was kept.
	CommitmentID int64

	// MetOn is the day the promise was met, written the way a person
	// writes it.
	MetOn string
}

// ReadCommitments is the book of the promises of a business, read for a day.
//
// It is the shape of the contract between the agent and the service: the day
// is optional, and the book of the promises is read whole when the agent
// names none.
type ReadCommitments struct {
	// BusinessID is the business whose promises are read.
	BusinessID business.ID

	// Day is the day the calendar is read for. It is empty when the agent
	// asks for the whole book of the promises.
	Day string
}

// CommitmentRecorder is the inbound port of the tools that write the
// promises of the business: the promise itself, and the day it was met.
type CommitmentRecorder interface {
	// Record writes the promise of the business and answers it with it.
	//
	// The identity of the business, what is due and the day it is due are
	// required, and the fields travel as the agent sent them: the domain
	// is the one that refuses the promise without a business, without a
	// what and without a day.
	//
	// The error is the error of the domain: the domain refuses the promise
	// of no business, the promise that says nothing about what is due, the
	// promise without a day, the business that is not there and the
	// business that does not hold promises.
	Record(ctx context.Context, request RecordCommitment) (commitment.Commitment, error)

	// Meet writes the day the promise was met and answers the promise that
	// was kept.
	//
	// The promise is named by its own identity, and the day it was met is
	// required. The promise that was met is KEPT, with the day it was met.
	//
	// The error is the error of the domain: the domain refuses the promise
	// that was met already, the promise that carries no day of meeting,
	// the promise that is not there and the business that does not hold
	// promises.
	Meet(ctx context.Context, request MeetCommitment) (commitment.Commitment, error)
}

// CommitmentReader is the inbound port of the tool that reads the calendar
// of the promises.
//
// The answer carries the promises that are due on the day and the ones that
// are still owed: every open promise of the business, the ones that are due
// on the day and the ones that fell due on an earlier day. The alarm is the
// work of the agent, and this is what the agent reads to ring.
type CommitmentReader interface {
	// Read answers the calendar of the day: the promises due on it and the
	// ones that are still owed.
	//
	// The identity of the business is required. The day is optional: the
	// calendar of the day carries the promises that are due on it, and the
	// whole book of the promises is read when the agent names none.
	//
	// The error is the error of the domain: the domain refuses the
	// business that is not there, the business that does not hold promises
	// and the day that is not a day of the book.
	Read(ctx context.Context, request ReadCommitments) (commitment.Calendar, error)
}
