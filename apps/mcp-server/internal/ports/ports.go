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
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
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
	// was worked and without the word of the owner.
	Record(ctx context.Context, request RecordLabor) (recipe.Labor, error)
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
