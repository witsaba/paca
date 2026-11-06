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
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
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

	business.ErrNotFound: "there is no personal business for that person: the onboarding opens one",
	business.ErrAlreadyExists: "that business already exists: the name of a business is taken inside the " +
		"businesses of its owner, and the personal business of a person is opened once",
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
	ledger.ErrInvalidPeriod: "the period of the book ends before it begins: the first day comes before the last one",
	ledger.ErrPaymentTooLarge: "the payment passes what the debt still holds: the person cannot pay more than what " +
		"they owe",
	ledger.ErrNotFound:  "there is no book for that business: check the business with the person before trying again",
	ledger.ErrNoStorage: "the service has no storage of the book: the book cannot be reached",

	ports.ErrCounterpartyRequired: "the debt needs the name of the other side of the money: ask the person who they " +
		"owe it to, or who owes it to them",
	ports.ErrNotADebt: "that account does not carry a debt: the debts are the accounts of the book that name the " +
		"other side of the money",

	purchase.ErrNotFound: "there is no draft with that identity: read the pending drafts of the business before " +
		"correcting one of them",
	purchase.ErrInvalidName:     "the name of a line is not empty: write the name the person read, the way she said it",
	purchase.ErrInvalidQuantity: "the quantity of a line is not positive: ask the person how much she bought",
	purchase.ErrMissingUnit: "the line carries no unit: ask the person in which unit she bought it ('kilos', " +
		"'paquetes', 'unidades'), and PACA keeps the unit of the book",
	purchase.ErrInvalidPrice: "the money of a line is not a number of pesos: ask the person what she paid",
	purchase.ErrPriceMismatch: "the money paid and the price of the unit do not agree: ask the person which of the " +
		"two is the right one",
	purchase.ErrInferredPrice: "the money of a line is never inferred: ask the person what she paid, and never " +
		"fill a price by yourself",
	purchase.ErrMissingUtterance: "the draft carries no sentence of the owner: send the sentence as she said it, " +
		"and not a summary of it",
	purchase.ErrNoLines: "the draft carries no line: read the sentence of the owner again and send the lines " +
		"she said",
	purchase.ErrNoOptions: "the ambiguity carries no option: offer the candidates PACA found, so the person " +
		"chooses with one word",
	purchase.ErrWrongBusiness: "the draft belongs to another business: draft the purchase of the business the " +
		"person names",
	purchase.ErrMissingConfirmation: "the confirmation carries no word: send the word the person used to confirm " +
		"the draft ('sí', 'confirmado')",
	purchase.ErrNotProposed: "the draft is not waiting for the yes of the person: it was confirmed already, or it " +
		"was dropped",
	purchase.ErrNotCancellable: "the draft does not wait any more: it was confirmed already, it was dropped " +
		"already, or it expired, and a draft that ended is never dropped again",
	purchase.ErrStaleRevision: "the draft changed under the eyes of the person: show it again and ask for the yes " +
		"once more",
	purchase.ErrExpiredDraft: "the draft expired: the price of a market does not wait, and the draft must be " +
		"drafted again",
	purchase.ErrStillNeeds: "the draft still needs the money of one of its lines: ask the person what she paid, " +
		"and never fill a price by yourself",
	purchase.ErrInvalidValue: "the database refused the draft: check the sentence, the lines and the day of it",
	purchase.ErrNoStorage:    "the service has no storage of the drafts: the drafts cannot be reached",

	ingredient.ErrNotFound: "there is no ingredient with that identity: read the catalog of the business before " +
		"naming one of its ingredients",
	ingredient.ErrInvalidName: "the name of an ingredient is not empty: write the name the person reads, the way " +
		"she said it",
	ingredient.ErrDuplicateName: "the business already has an ingredient with that name: use the name the person " +
		"reads for it, and do not write a second one",
	ingredient.ErrWrongBusiness: "the ingredient belongs to another business: match the name against the catalog " +
		"of the business the person names",
	ingredient.ErrInvalidValue: "the database refused the ingredient: check the name of it",
	ingredient.ErrNoStorage:    "the service has no storage of the ingredients: the catalog cannot be reached",
	ingredient.ErrUnknownUnit: "the unit is not one of the units PACA knows: use 'g', 'kg', 'paquete' or " +
		"'unidad', the way the owner buys the thing",
	ingredient.ErrInvalidQuantity: "the quantity of the ingredient is not positive: ask the owner how much of it " +
		"she buys",
	ingredient.ErrMissingPackageWeight: "the weight of the package is not declared: ask the owner how much the " +
		"package weighs, in grams, and never assume it",
	ingredient.ErrInvalidPackageWeight: "the weight of the package is not a positive number of grams: ask the " +
		"owner how much the package weighs",
	ingredient.ErrInvalidCertification: "the certification carries no name: ask the owner for the claim she " +
		"makes about the ingredient ('Orgánico')",
	ingredient.ErrMissingEvidence: "the certification carries no evidence: ask the owner for the document that " +
		"proves the claim, because a claim nobody can check is a wish",
	ingredient.ErrInvalidConversion: "the conversion of the ingredient is not positive: it is the grams of the " +
		"raw that one gram of the prepared ingredient takes",
	ingredient.ErrUnknownConversion: "the piece is one unit of the ingredient, and its weight is never assumed: " +
		"ask the owner how much the piece weighs",

	recipe.ErrNotFound: "the business or the ingredient of the recipe is not there: check the business, and match " +
		"the ingredient against the catalog of that business before writing the recipe",
	recipe.ErrMissingName: "the recipe carries no name: ask the owner for the name she reads ('Pan de molde'), " +
		"and never invent one",
	recipe.ErrMissingConfirmation: "the recipe carries no word of the owner: send the word she used to confirm it " +
		"('sí', 'confirmado'), and keep it with the record",
	recipe.ErrNoIngredients: "the recipe carries no ingredient: read the sentence of the owner again and send the " +
		"ingredients she said, with their grams",
	recipe.ErrMissingGrams: "an ingredient of the recipe carries no grams: ask the owner how many grams of it the " +
		"recipe needs, and never guess them",
	recipe.ErrUnknownIngredient: "the ingredient of the recipe is not one of the catalog of the business: match the " +
		"name against the catalog first, and send the identity it answered",
	recipe.ErrNoSteps: "the recipe carries no step: a recipe is a guide to learn and to teach, so send the steps " +
		"the owner said, in that order",
	recipe.ErrMissingInstruction: "a step of the recipe carries no instruction: ask the owner what she does in " +
		"that step",
	recipe.ErrInvalidTemperature: "the temperature of a step is below the absolute zero: ask the owner for the " +
		"temperature of the oven, in degrees Celsius",
	recipe.ErrInvalidYield: "the yield of the recipe is not positive: ask the owner how many units the recipe " +
		"makes and how much each one weighs",
	recipe.ErrMissingShelfLife: "the recipe carries no shelf life: ask the owner how many days what the recipe " +
		"makes keeps, and correct it later when she knows it better",
	recipe.ErrInvalidValue: "the database refused the recipe: check the name, the ingredients, the steps, the " +
		"yield and the shelf life of it",
	recipe.ErrNoStorage: "the service has no storage of the recipes: the book of the bakery cannot be reached",
	recipe.ErrWrongBusiness: "the recipe belongs to another business: read the cost of a batch of the " +
		"business the person names",
	recipe.ErrInvalidHours: "the hours of the labor are not a number of hours: a negative number of hours is a " +
		"number nobody worked",
	recipe.ErrMissingHours: "the labor carries no hours: ask the owner how many hours she worked, and in which " +
		"of the three kinds of the labor",
	recipe.ErrMissingHourValue: "the value of the hour is not declared: ask the owner what the hour of her work " +
		"is worth, and never fill it by yourself",
	recipe.ErrMissingDay: "the labor carries no day: say the day the labor was worked, and never invent one",
	recipe.ErrUnknownLaborKind: "the kind of the labor is not one of the three kinds PACA knows: use 'hands_on', " +
		"'oven' or 'waiting'",
	recipe.ErrMissingBusiness: "the business is required: send the identity of the business the record belongs to, " +
		"and PACA never invents one",
	recipe.ErrMissingElectricity: "the overhead carries no electricity: ask the owner what the electricity of the " +
		"house cost, and never fill it by yourself",
	recipe.ErrMissingOvenHours: "the overhead carries no hours of the oven: ask the owner how many hours the oven " +
		"ran, because without them the money of the oven has no rate",
	recipe.ErrInvalidAllocation: "the allocation is not a share of the electricity: ask the owner what share of " +
		"the electricity of the house belongs to the oven (0.6 is six of every ten pesos)",
	recipe.ErrMissingRecipe: "the production names no recipe: ask the owner what she made, and send the recipe she means",
	recipe.ErrAmbiguousRecipe: "the business holds more than one recipe with that name: ask the owner which recipe " +
		"she made, and send the identity of the one she means",
	recipe.ErrUnknownRecipe: "the business holds no recipe with that name: read the recipes of the business, and send " +
		"the identity of the recipe she made",
	recipe.ErrMissingProductionDay: "the production carries no day: ask the owner what day she made it, and never invent one",
	recipe.ErrMissingProductionConfirmation: "the production carries no word of the owner: send the word she used to " +
		"confirm it ('sí', 'confirmado'), and keep it with the record",
	recipe.ErrMissingProductionGrams: "the production carries no grams that went in: the stock does not hold what " +
		"the recipe needs, and PACA never guesses the grams",
	recipe.ErrMissingProductionUnits: "the production made no unit: the recipe declares no yield, and a production " +
		"without units is not a production",
	recipe.ErrInvalidProductionMoney: "the money of the production is not a number of pesos: the money of the book " +
		"counts whole pesos, and PACA never infers it",
	recipe.ErrMissingStock: "the stock does not hold what the recipe needs: record the purchase of the missing " +
		"ingredient first, and record the production after it",
	recipe.ErrUnknownStock: "the stock holds an ingredient in a unit the book cannot weigh: ask the owner how much " +
		"the package weighs, and never assume the weight of a piece",

	purchase.ErrMissingIngredient: "the reference names nothing of the catalog: match the name against the catalog " +
		"of the business first, and send the identity it answered",
	purchase.ErrInvalidReferencePrice: "the price of the reference is not a positive number of pesos: ask for the " +
		"price that was seen, and never fill it by yourself",
	purchase.ErrMissingReferenceUnit: "the price of the reference carries no unit: ask in which unit the price was " +
		"seen ('kg', 'gramos', 'paquete', 'unidad'), and PACA keeps the unit of the book",
	purchase.ErrMissingSource: "the reference carries no source: say where the price was read, because a price " +
		"without a source is a price nobody can check",
	purchase.ErrMissingDay: "the reference carries no day: say the day the price was seen, and never invent one",
	purchase.ErrLayerWithoutIngredient: "the layer names nothing of the catalog: match the name against the catalog " +
		"of the business first, and send the identity it answered",
	purchase.ErrInvalidLayer: "the layer is not whole: what it holds is not what it bought, and the storage " +
		"refused the row",
	purchase.ErrInvalidPosition: "the layer comes from no line of its purchase: the storage refused the row",
	purchase.ErrInvalidUnit: "the unit of the layer is not one of the units PACA knows: use 'g', 'kg', " +
		"'paquete' or 'unidad', the way the owner buys the thing",
	purchase.ErrLayerOfAnotherBusiness: "the layer belongs to the business of its purchase: the stock of a " +
		"business never holds the layer of a purchase of another one",
	purchase.ErrMissingBusiness: "the business is required: send the identity of the business whose stock " +
		"you read, and PACA never invents one",
	purchase.ErrLayerNotTied: "the row of the layer was broken: the money of the layer is the money of the " +
		"line of the purchase it comes from, and a layer that carries another money is never read as a layer " +
		"of the stock",
	purchase.ErrInvalidOwnerPrice: "the price of the owner is not a price: ask the owner what she charges for " +
		"the product, and never fill a price by yourself",
	purchase.ErrUnknownPriceBasis: "the way the owner charges is not one of the three the book knows: use " +
		"'unit' (the price of one piece), 'gram' (the price of one gram) or 'kilogram' (the price of one " +
		"kilogram)",
}
