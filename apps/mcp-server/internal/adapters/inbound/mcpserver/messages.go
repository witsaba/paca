// Package mcpserver adapts the service to the Model Context Protocol.
//
// This file holds the messages the agent reads when a tool fails.
//
// Every error of the domain has a message here, written in the words of the
// caller: the agent reads the message, and it must know what to do about it.
// The catalogue of the domain (`user.Errors`, `business.Errors`,
// `account.Errors` and `ledger.Errors`) names the errors that must be here,
// and a test proves that every one of them is.
//
// The file holds the messages of every tool of the service, because the
// message is the same promise wherever the tool lives: the agent reads it
// the same way.
package mcpserver

import (
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/user"
)

// The messages the agent reads when a tool fails.
//
// Every error of the domain has a message here, written in the words of the
// caller: the agent reads the message, and it must know what to do about it.
// The catalogue of the domain (`user.Errors`) names the errors that must be
// here, and a test proves that every one of them is.
var theMessages = map[error]string{
	user.ErrInvalidID:             "the document ID is not a number: ask the person for the national document ID, with digits only",
	user.ErrNameRequired:          "the name of the person is required: ask the person for their name before registering them",
	user.ErrInvalidPhone:          "the phone number is not an international number: ask for it again with the country code, for example '+57 321 253 9984'",
	user.ErrInvalidEmail:          "the email address is not valid: ask the person to confirm it",
	user.ErrAlreadyExists:         "there is already a user with that document ID: do not register the person again",
	user.ErrEmailTaken:            "that email address is already registered: ask the person for another one",
	user.ErrPhoneTaken:            "that phone number is already registered: ask the person for another one",
	user.ErrNotFound:              "there is no user with that document ID: check the document with the person before trying again",
	user.ErrInvalidTransition:     "that status change is not allowed: a user never returns to the onboarding, and a status never changes to itself",
	user.ErrObservationRequired:   "that status needs a reason: ask the person why, and send the observation",
	user.ErrUnexpectedObservation: "that status takes no reason: send the observation only when the new status is inactive or blocked",
	user.ErrUnknownStatus:         "the status must be active, inactive or blocked",
	user.ErrConcurrentChange:      "the status of the user changed at the same time: read the user again and try the change once more",
	user.ErrInvalidValue:          "the database refused the row: check the name, the phone and the email of the person",
	user.ErrNoStorage:             "the service has no storage of the users: the users cannot be reached",
	ports.ErrNoDatabase:           "the service has no database configured: the users cannot be reached",

	business.ErrNotFound:        "there is no personal business for that person: the onboarding opens one",
	business.ErrAlreadyExists:   "that business already exists: the personal business of a person is opened once",
	business.ErrInvalidName:     "the name of the business is not empty: give the business the name the person reads",
	business.ErrInvalidCurrency: "the currency of the business is not empty: give the business the currency of its book",
	business.ErrUnknownKind: "the kind of the business is not one of the kinds PACA knows: " +
		"use 'personal' for the business of a person",
	business.ErrInvalidValue: "the database refused the business: check the name and the currency of it",
	business.ErrNoStorage:    "the service has no storage of the businesses: the book cannot be reached",

	account.ErrNotFound:      "there is no account with that identity: read the accounts of the business first",
	account.ErrAlreadyExists: "that account already exists in this business: the name of an account is taken inside it",
	account.ErrInvalidName:   "the name of the account is not empty: give the account the name the person reads",
	account.ErrUnknownKind: "the kind of the account is not one of the kinds PACA knows: " +
		"use asset, liability, income, expense or equity",
	account.ErrDebtNotAllowed: "only an account that owes or is owed carries a counterparty and a due date: " +
		"use a liability or an asset for the debt",
	account.ErrInvalidValue: "the database refused the account: check the name and the kind of it",
	account.ErrNoStorage:    "the service has no storage of the accounts: the book cannot be reached",

	ledger.ErrInvalidAmount: "the amount of the movement is not positive: the money that moves is always a " +
		"positive number of pesos",
	ledger.ErrSameAccount: "the two sides of the movement are the same account: the money goes from one account " +
		"to another one",
	ledger.ErrInvalidDate: "the date is not a day of the book: write it the way a person writes it, " +
		"for example '2026-03-01'",
	ledger.ErrInvalidDescription: "the description of the movement is not empty: write what the money was for, " +
		"in the words of the person",
	ledger.ErrNotFound:  "there is no book for that business: check the business with the person before trying again",
	ledger.ErrNoStorage: "the service has no storage of the book: the book cannot be reached",
}
