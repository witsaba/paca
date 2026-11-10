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
	"github.com/witsaba/paca/packages/domain/order"
	"github.com/witsaba/paca/packages/domain/party"
	"github.com/witsaba/paca/packages/domain/product"
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
	ports.ErrInvalidMaxAgeDays: "the expectation of the age of the references is not a number of days: send the " +
		"number of days a reference of the market may be old, and never a negative one",

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
	recipe.ErrMissingLaborConfirmation: "the labor carries no word of the owner: send the word she used to " +
		"confirm the labor ('sí', 'confirmado'), and it is kept with the record",
	recipe.ErrMissingOverheadConfirmation: "the overhead carries no word of the owner: send the word she used to " +
		"confirm the overhead ('sí', 'confirmado'), and it is kept with the record",
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
	recipe.ErrInvalidMinutes: "the minutes of a step are not a time: ask the owner how many minutes the step " +
		"takes, and never invent them",
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
	recipe.ErrInferredHourValue: "the value of the hour of the labor was filled by PACA: ask the owner what " +
		"the hour of her work is worth, and never fill it by yourself",
	recipe.ErrMissingDay: "the labor carries no day: say the day the labor was worked, and never invent one",
	recipe.ErrUnknownLaborKind: "the kind of the labor is not one of the three kinds PACA knows: use 'hands_on', " +
		"'oven' or 'waiting'",
	recipe.ErrMissingBusiness: "the business is required: send the identity of the business the record belongs to, " +
		"and PACA never invents one",
	recipe.ErrMissingElectricity: "the overhead carries no electricity: ask the owner what the electricity of the " +
		"house cost, and never fill it by yourself",
	recipe.ErrInferredElectricity: "the electricity of the overhead was filled by PACA: ask the owner what she " +
		"paid for the electricity of the house, and never fill it by yourself",
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
	recipe.ErrExpiredUnit: "the unit expired on that day: a unit that expired is never sold, and it has to be " +
		"written as a waste",
	recipe.ErrMissingSaleDay: "the sale carries no day: ask the owner the day of the sale, and never invent one",
	recipe.ErrMissingExpiry: "the unit of the production carries no day of expiry: read the productions of the " +
		"business again, and never invent a day of expiry",
	recipe.ErrInvalidWasteQuantity: "the quantity of the waste is not a whole number of units: ask the owner how " +
		"many units she lost, and never guess them",
	recipe.ErrInvalidWasteUnit: "the unit of the waste is not the piece: the waste of the bakery counts the " +
		"units it lost ('unidad'), and PACA never converts a weight into a number of units",
	recipe.ErrUnknownWasteReason: "the reason of the waste is not one of the four PACA knows: use 'burnt', " +
		"'spoiled', 'dropped' or 'returned', and ask the owner why she lost them",
	recipe.ErrMissingWasteDay: "the waste carries no day: ask the owner the day it happened, and never invent one",
	recipe.ErrMissingWasteConfirmation: "the waste carries no word of the owner: send the word she used to confirm " +
		"it ('sí', 'confirmado'), and keep it with the record",
	recipe.ErrInvalidWasteMoney: "the money of the waste is not a number of pesos: the money of the book counts " +
		"whole pesos, and PACA never infers it",
	recipe.ErrNotEnoughUnits: "the book holds no unit of that product to lose: read what expires, and ask the " +
		"owner how many units she lost",
	recipe.ErrNotEnoughUnitsToSell: "the book holds no unit of that product to sell: read what the bakery still " +
		"has, and ask the owner how many units she sold",
	recipe.ErrUnknownProduction: "the production of the unit is not there: read the productions of the business " +
		"again, and never guess the money of a unit",
	recipe.ErrInvalidSaleQuantity: "the quantity of the sale is not a whole number of units: ask the owner how " +
		"many units she sold, and never guess them",
	recipe.ErrInvalidSaleUnit: "the unit of the sale is not the piece: the sale of the bakery counts the units " +
		"it sold ('unidad'), and PACA never converts a weight into a number of units",
	recipe.ErrInvalidSaleMoney: "the money of the sale is not a positive number of pesos: ask the owner what the " +
		"client paid, and never fill it by yourself",
	recipe.ErrInvalidSaleCost: "the cost of the sale is not a number of pesos: the cost of the sale is the " +
		"money of the units that left the book, and it is never negative",
	recipe.ErrMissingSaleConfirmation: "the sale carries no word of the owner: send the word she used to confirm " +
		"it ('sí', 'confirmado'), and keep it with the record",
	recipe.ErrUnknownExitKind: "the kind of the exit is not one of the two PACA knows: use 'consumption' when the " +
		"owner ate the bread, and 'donation' when the bakery gave it away",
	recipe.ErrInvalidExitQuantity: "the quantity of the exit is not a whole number of units: ask the owner how " +
		"many units she consumed or gave away, and never guess them",
	recipe.ErrInvalidExitUnit: "the unit of the exit is not the piece: the exit of the bakery counts the units " +
		"it took ('unidad'), and PACA never converts a weight into a number of units",
	recipe.ErrMissingExitDay: "the exit carries no day: ask the owner the day it happened, and never invent one",
	recipe.ErrMissingExitConfirmation: "the exit carries no word of the owner: send the word she used to confirm " +
		"it ('sí', 'confirmado'), and keep it with the record",
	recipe.ErrInvalidExitCost: "the cost of the exit is not a number of pesos: the cost of the exit is the cost " +
		"of the units that left the book, and it is never negative",
	recipe.ErrMissingTaxTreatment: "the exit carries no tax treatment: the tax treatment of an exit waits for " +
		"the professional of the taxes of the owner, and PACA never decides it by herself",
	recipe.ErrNotEnoughUnitsToExit: "the book holds no unit of that product to exit: read what the bakery still " +
		"has, and ask the owner how many units she consumed or gave away",

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
	purchase.ErrLayerWithoutName: "the layer carries no name: the layer names the ingredient the owner bought, " +
		"the way she said it, and a layer with no name is never read as a layer of the stock",
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

	purchase.ErrMissingProvider: "the invoice names no provider: send the identity of the party that issued " +
		"it, the farm, the store or the person the bakery buys from",
	purchase.ErrNotAProvider: "the party does not hold the role of the provider: open the provider with the " +
		"tool of the parties, with the role of the provider, and send the identity it answered",
	purchase.ErrMissingPurchase: "the invoice names no purchase: send the identity of the purchase the " +
		"invoice came from, and the purchase the owner confirmed",
	purchase.ErrMissingInvoiceNumber: "the invoice carries no number: read the number the invoice carries, " +
		"the way it was printed on it",
	purchase.ErrMissingInvoiceDay: "the invoice carries no day: ask the owner the day the invoice was issued, " +
		"and never invent one",
	purchase.ErrInvalidInvoiceMoney: "the money of the invoice is not a number of pesos: the money of the " +
		"book counts whole pesos, and it is never negative",
	purchase.ErrInvalidRate: "the rate of the tax is not a rate: send the rate the invoice states, in " +
		"percent, and never fill it by yourself: there is no default of nineteen percent and no default of zero",
	purchase.ErrInvoiceDoesNotAddUp: "the total of the invoice is not the base plus the tax: read the " +
		"base, the rate and the total the invoice carries, and ask the owner which of the three is wrong",
	purchase.ErrInvoiceOfAnotherBusiness: "the invoice belongs to the business of its purchase: the invoice " +
		"and the purchase it came from belong to the same business, and PACA never attaches them apart",
	purchase.ErrDuplicateInvoiceNumber: "the business already holds an invoice with that number: the number " +
		"of an invoice is unique inside the business, and the agent should read the invoices of the business " +
		"before it writes one",
	purchase.ErrInvoiceOfAnotherPurchase: "the invoice names another purchase: the invoice is written against " +
		"the purchase it came from, and the agent should send the identity of the purchase the invoice names",
	purchase.ErrPurchaseAlreadyHasInvoice: "the purchase already carries an invoice: a purchase carries the " +
		"invoice that came with it, and the agent should read the invoices of the business before it writes one",

	party.ErrNotFound: "the party is not there: read the parties of the business to know the ones it holds, and " +
		"name one of them",
	party.ErrBusinessNotFound: "the business of the party is not there: check the business the party belongs " +
		"to, and read the parties of the business before naming one of them",
	party.ErrUnknownKind: "the kind of a party is not one of the kinds PACA knows: use 'person', 'store' or " +
		"'farm'",
	party.ErrInvalidName: "the name of a party is not empty: give the party the name the owner reads",
	party.ErrUnknownRole: "the role of a party is not one of the roles PACA knows: use 'client' (the party that " +
		"buys from the business) or 'provider' (the party the business buys from)",
	party.ErrNoRole: "the party holds no role: send the role of the client, the role of the provider, or both " +
		"of them, because a party that is neither is not a party of the book",
	party.ErrMissingLegalHuman: "the legal human that represents the party is not there: ask for the name of the " +
		"human that represents the store, and for the legal figure of the farm that has one",
	party.ErrUnexpectedLegalHuman: "this party of that kind is not represented by a legal human: a person is a " +
		"person, and the legal human of a store or of a farm is a fact of the party and never a second person",
	party.ErrWrongBusiness: "the parties belong to a business of its own: the Personal Business of a person holds " +
		"the money of that person, and not the clients and the providers of the bakery",
	party.ErrInvalidValue: "the database refused the party: check the kind, the name, the roles and the legal " +
		"human of it",
	party.ErrNoStorage: "the service has no storage of the parties: the parties cannot be reached",

	product.ErrNotFound: "there is no product with that identity: read the price list of the business to " +
		"know the products it holds, and name one of them",
	product.ErrInvalidName: "the name of a product is not empty: give the product the name the owner reads, " +
		"and never invent one",
	product.ErrDuplicateName: "the business already holds a product with that name: use the name the owner " +
		"reads for it, and do not write a second one",
	product.ErrDuplicateRecipe: "the recipe already backs another product of the business: one recipe backs " +
		"one product, and the price of that product is the one of the recipe the owner named",
	product.ErrMissingRecipe: "the product names no recipe: the bakery sells what it bakes, so ask the owner " +
		"which recipe the product comes from, and send the identity it answered",
	product.ErrUnknownRecipe: "the business holds no product and no recipe with that name: ask the owner " +
		"which recipe the product comes from, and read the recipes of the business before naming one",
	product.ErrMissingProduct: "the price names no product: send the identity of the product, or the recipe " +
		"it comes from, and the price is never written without it",
	product.ErrUnknownChannel: "the channel of the price is not one of the channels PACA knows: use 'retail' " +
		"(the client that buys at the door) or 'wholesale' (the store that buys many of them); the owner says " +
		"them as 'minorista' and 'mayorista' too",
	product.ErrInvalidPrice: "the price of a product is not a positive number of pesos: ask the owner what " +
		"she charges for the product, and never fill a price by yourself",
	product.ErrMissingDay: "the price carries no day: say the day the price was set, and never invent one",
	product.ErrWrongBusiness: "the price list belongs to a business of its own: the Personal Business of a " +
		"person holds the money of that person, and not the price list of the bakery",
	product.ErrOfAnotherBusiness: "the recipe and the product belong to the business that owns them: the " +
		"product of the bakery is backed by a recipe of its own business, and the price is of a product of it: " +
		"read the recipes of the business, and name one of them",
	product.ErrInvalidValue: "the database refused the price: check the channel, the money and the day of it",
	product.ErrNoStorage:    "the service has no storage of the price list: the price list cannot be reached",

	order.ErrNotFound: "there is no order with that identity: read the orders of the business to know the ones it " +
		"holds",
	order.ErrMissingBusiness: "the order belongs to no business: send the identity of the business the order " +
		"belongs to, and PACA never invents one",
	order.ErrMissingParty: "the order names no party: send the identity of the client that ordered, or the name " +
		"the owner reads for it",
	order.ErrPartyOfAnotherBusiness: "the party of the order belongs to another business: the client that " +
		"ordered is a party of the business the order belongs to, and PACA never attaches them apart",
	order.ErrNotAClient: "the party of the order does not hold the role of the client: the bakery does not owe " +
		"bread to a provider, and the client is opened with the role of the client",
	order.ErrMissingProduct: "the line of the order names no product: send the identity of the product of the " +
		"price list, or the name the owner reads for it",
	order.ErrProductOfAnotherBusiness: "the product of the order belongs to another business: the order carries " +
		"the products of the price list of the business it belongs to",
	order.ErrNoLines: "the order carries no line: read the sentence of the owner again and send the products she " +
		"ordered, with the quantity and the unit of each one",
	order.ErrInvalidQuantity: "the quantity of a line is not positive: ask the owner how much she ordered, and never " +
		"guess it",
	order.ErrSplitPiece: "the quantity of a line splits a piece: the book never splits a piece, so ask the owner " +
		"how many pieces she ordered, and never guess them",
	order.ErrMissingUnit: "the line of the order carries no unit: ask the owner how she charges it ('unidad', 'g', " +
		"'kilo'), and the quantity and the way she charges travel together",
	order.ErrUnknownUnit: "the unit of the line is not one of the units of the book: use 'unit' (the piece), " +
		"'gram' (the weight) or 'kilogram' (the weight), the way the owner charges",
	order.ErrBusinessNotFound: "there is no business with that identity: the business of the order is not there, " +
		"and PACA never invents one",
	order.ErrMissingDay: "the order carries no day: ask the owner the day the order is due on, and never invent one",
	order.ErrWrongBusiness: "the orders belong to a business of its own: the Personal Business of a person holds " +
		"the money of that person, and not the orders of the bakery",
	order.ErrUnknownState: "the state of an order is not one of the states PACA knows: use 'open' (the order " +
		"that waits for its round) or 'delivered' (the order the round delivered)",
	order.ErrInvalidValue: "the database refused the order: check the party, the day and the lines of it",
	order.ErrNoStorage:    "the service has no storage of the orders: the orders cannot be reached",
	order.ErrMissingRouteDay: "the route carries no day: read the round of the day you are visiting, and send the day " +
		"it is of",
	order.ErrNoStops: "the route carries no stop: read the round of the day you are visiting, and send the stops of it, " +
		"in the order you will visit the clients in",
	order.ErrRouteAlreadyWritten: "the route of that day was already written: read the round of that day and use the route " +
		"that is already there, or write the route of another day",
	order.ErrStopNotAnOrder: "the stop is not an order of that business: read the orders of the business, the ones " +
		"that are still owed, and send the identity of one of them",
	order.ErrDuplicateStop: "the route visits the same order in two stops: every order is visited once, and the total of " +
		"the route counts the money of a stop once",
	order.ErrInvalidCost: "the cost of the stop is less than nothing: the money of the transport to that client is " +
		"never negative, and it is sent in whole pesos — a stop the owner walks to costs zero",
	order.ErrAlreadyDelivered: "the order was delivered already: it is not owed any more, and it is not visited again",
}
