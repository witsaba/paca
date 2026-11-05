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
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
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
	Unit string

	// Total is the money that was paid for the whole line, in pesos.
	Total int64

	// UnitPrice is the price of one unit of the line, in pesos.
	UnitPrice int64

	// Inferred says that the price was filled by the agent and not said by
	// the owner. The money is never inferred, and the domain refuses it.
	Inferred bool
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
	DraftID purchase.ID

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
