package recipe

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// LaborKind is the kind of the labor of a step of the process of a recipe.
//
// The labor of the bakery is measured in three kinds: the hours the hands
// are on the dough, the hours of the oven, and the hours of the waiting.
//
// The kind belongs to the domain: the description of the tools reads it,
// the step of the recipe carries it, and the table of the steps holds the
// same rule as a defense in depth.
type LaborKind string

const (
	// LaborHandsOn is the labor of the hands: the amassing, the shaping
	// and everything the owner does with her hands.
	LaborHandsOn LaborKind = "hands_on"

	// LaborOven is the labor of the oven: the hours the batch bakes.
	LaborOven LaborKind = "oven"

	// LaborWaiting is the labor of the waiting: the hours the dough
	// rests, and the owner does nothing but wait.
	LaborWaiting LaborKind = "waiting"
)

// MinutesPerHour is the number of minutes of an hour.
//
// The step of a recipe carries its time in minutes, and the labor of the
// batch is measured in hours: the conversion is a rule of the book, and it
// is written here.
const MinutesPerHour = 60.0

// theSpokenLaborKinds are the names the owner and the agent say for the
// kinds of the labor.
//
// The values of the kinds are the ones the storage and the protocol carry:
// the names a person says are read the way a person reads them. The list is
// the vocabulary of the domain, and it holds the plain ways a person says
// the three kinds of the labor - and nothing else: a kind the book does
// not know stays unknown.
var theSpokenLaborKinds = map[string]LaborKind{
	"hands_on":  LaborHandsOn,
	"hands-on":  LaborHandsOn,
	"hands":     LaborHandsOn,
	"manos":     LaborHandsOn,
	"mano":      LaborHandsOn,
	"oven":      LaborOven,
	"horno":     LaborOven,
	"horneado":  LaborOven,
	"waiting":   LaborWaiting,
	"wait":      LaborWaiting,
	"espera":    LaborWaiting,
	"esperando": LaborWaiting,
	"reposo":    LaborWaiting,
}

// LaborKinds returns the kinds of the labor, in the order the owner reads
// them.
//
// The vocabulary belongs to the domain: the description of the tools reads
// it, and the check of the table holds the same rule as a defense in
// depth.
func LaborKinds() []LaborKind {
	return []LaborKind{LaborHandsOn, LaborOven, LaborWaiting}
}

// SpokenLaborKinds returns the ways the owner and the agent say the kinds
// of the labor, in the order a person reads them.
//
// The list is the declared vocabulary of the domain: the description of the
// tools reads it, so the agent sends the kind the way the owner said it.
// The list holds no kind the book does not know.
func SpokenLaborKinds() []string {
	spoken := make([]string, 0, len(theSpokenLaborKinds))
	for name := range theSpokenLaborKinds {
		spoken = append(spoken, name)
	}

	sort.Strings(spoken)

	return spoken
}

// String returns the name of the kind.
//
// It is the name the storage and the protocol carry, with no translation in
// between.
func (k LaborKind) String() string {
	return string(k)
}

// ParseLaborKind answers the kind of the labor the owner named.
//
// The name is read the way a person reads it: trimmed, and without case.
// The answer is the kind the book knows, never the way it was said: the
// storage and the protocol carry the three names of the kinds.
//
// The empty name is the labor of the hands: a step the owner did not
// qualify is a step she did with her hands.
//
// The error is ErrUnknownLaborKind when the name is not one of them: the
// kind of the labor is a rule of the book, and a kind the book does not
// know is never guessed.
func ParseLaborKind(spoken string) (LaborKind, error) {
	key := strings.ToLower(strings.TrimSpace(spoken))
	if key == "" {
		return LaborHandsOn, nil
	}

	kind, known := theSpokenLaborKinds[key]
	if !known {
		return "", fmt.Errorf("%w: %q", ErrUnknownLaborKind, spoken)
	}

	return kind, nil
}

// Hours are the hours of the labor, in the three kinds of it.
//
// The hours are measured in hours, and the three kinds are held apart: the
// answer carries the hours of the hands, of the oven and of the waiting
// apart, and the total of them is their sum.
type Hours struct {
	// HandsOn is the number of hours the hands of the owner were on the
	// dough.
	HandsOn float64

	// Oven is the number of hours the batch was in the oven.
	Oven float64

	// Waiting is the number of hours the dough waited.
	Waiting float64
}

// NewHours builds the hours of the labor.
//
// The rule of the domain: the hours are not negative. A number of hours
// that is negative is a number nobody worked.
func NewHours(handsOn, oven, waiting float64) (Hours, error) {
	if handsOn < 0 || oven < 0 || waiting < 0 {
		return Hours{}, fmt.Errorf("%w: the hours are %v, %v and %v",
			ErrInvalidHours, handsOn, oven, waiting)
	}

	return Hours{HandsOn: handsOn, Oven: oven, Waiting: waiting}, nil
}

// Total answers the number of hours of the labor, in the three kinds
// together.
//
// The hours of the labor of the owner are the hours of the three kinds
// together: the hands, the oven and the waiting.
func (h Hours) Total() float64 {
	return h.HandsOn + h.Oven + h.Waiting
}

// With answers the hours with the hours of one kind added to them.
//
// The answer is the hours the labor had before, and the hours of the kind
// added to it: the labor of a batch is derived from the steps, and every
// step adds its hours to the kind it belongs to.
func (h Hours) With(kind LaborKind, hours float64) Hours {
	switch kind {
	case LaborHandsOn:
		h.HandsOn += hours
	case LaborOven:
		h.Oven += hours
	case LaborWaiting:
		h.Waiting += hours
	}

	return h
}

// LaborID is the identity of a labor.
//
// The number is generated by the database: it is the counter of the labors
// and not a number of the person. The type exists so that an identifier
// never travels as a plain integer by accident.
type LaborID int64

// Labor is the labor of the owner, as she declared it.
//
// It carries the hours of the labor in the three kinds of it, the value of
// the hour, the day it was worked and the word of the owner.
//
// The labor of the PAC is the labor of the owner alone: the payroll of the
// employees is out of scope for now, and the model of the labor of an
// employee does not exist yet. The labor of the bakery names no employee,
// because it has none.
type Labor struct {
	// ID is the identity of the labor. It is empty until the storage
	// writes the row.
	ID LaborID

	// BusinessID is the business the labor belongs to.
	BusinessID business.ID

	// Day is the day the labor was worked.
	Day ledger.Date

	// Hours are the hours of the labor, in the three kinds of it.
	Hours Hours

	// HourValue is the value of the hour, in whole pesos: what the
	// owner declared the hour of her work is worth.
	HourValue ledger.Amount

	// HourValueInferred says that the value of the hour was filled by
	// the agent and not said by the owner: the money of the work of a
	// person is never inferred.
	//
	// It is never true in a labor that the book built: the inferred
	// money is refused, the way the purchase refuses the money of a line
	// that was inferred.
	HourValueInferred bool

	// Money is the money of the labor, in whole pesos: the hours of it
	// times the value of the hour.
	Money ledger.Amount

	// Utterance is the sentence of the owner, verbatim. It is empty when
	// she said none.
	Utterance string

	// ConfirmedWith is the word she used to confirm the labor.
	ConfirmedWith string

	// CreatedAt is the moment the row was created. The storage filled it.
	CreatedAt time.Time
}

// LaborDefinition is what the agent read about a labor.
//
// It is the candidate the agent produced, and the domain decides what may
// enter the book of the business.
type LaborDefinition struct {
	// Hours are the hours of the labor, in the three kinds of it.
	Hours Hours

	// HourValue is the value of the hour, in whole pesos. It is declared
	// by the owner, and never inferred.
	HourValue ledger.Amount

	// HourValueInferred says that the value of the hour was filled by
	// the agent and not said by the owner: the money of the work of a
	// person is never inferred.
	HourValueInferred bool

	// Day is the day the labor was worked.
	Day ledger.Date

	// Utterance is the sentence of the owner, verbatim. It is optional.
	Utterance string

	// ConfirmedWith is the word she used to confirm the labor.
	ConfirmedWith string
}

// NewLabor builds the labor of a business.
//
// The rules of the domain hold here:
//
//   - the labor belongs to a business: the labor of the bakery is the
//     labor of the business that works it;
//   - the labor carries hours: the hours of the owner are the fact, and a
//     labor without hours is nothing;
//   - the value of the hour is declared by the owner: it is positive, and
//     it is never inferred, because the money of the work of a person is
//     not a number PACA may invent. A value the agent filled is refused,
//     the way the purchase refuses the money of a line that was inferred;
//   - the labor carries the day it was worked;
//   - the labor carries the word of the owner: it is written with her
//     confirmation, the way the purchase is written.
//
// The money of the labor is the hours of it times the value of the hour,
// and the money of the book counts whole pesos: the answer is rounded to
// the nearest peso.
//
// The error is the error of the domain: the labor carries its own error
// when it carries no word of the owner, and the error of the recipe is
// never the answer of a labor.
func NewLabor(own business.ID, definition LaborDefinition) (Labor, error) {
	if own <= 0 {
		return Labor{}, fmt.Errorf("%w: the labor is of the business %d", ErrMissingBusiness, own)
	}

	if definition.Hours.Total() <= 0 {
		return Labor{}, fmt.Errorf("%w: the labor carries no hours", ErrMissingHours)
	}

	if definition.HourValueInferred {
		return Labor{}, fmt.Errorf("%w: the value of the hour is %d",
			ErrInferredHourValue, definition.HourValue)
	}

	if definition.HourValue <= 0 {
		return Labor{}, fmt.Errorf("%w: the value of the hour is %d",
			ErrMissingHourValue, definition.HourValue)
	}

	if definition.Day.Time().IsZero() {
		return Labor{}, fmt.Errorf("%w: the labor carries the day it was worked", ErrMissingDay)
	}

	word := strings.TrimSpace(definition.ConfirmedWith)
	if word == "" {
		return Labor{}, fmt.Errorf("%w: the labor carries no word of the owner", ErrMissingLaborConfirmation)
	}

	return Labor{
		BusinessID:        own,
		Day:               definition.Day,
		Hours:             definition.Hours,
		HourValue:         definition.HourValue,
		HourValueInferred: definition.HourValueInferred,
		Money:             moneyOfTheHours(definition.Hours, definition.HourValue),
		Utterance:         strings.TrimSpace(definition.Utterance),
		ConfirmedWith:     word,
	}, nil
}

// moneyOfTheHours answers the money of the hours of the labor.
//
// The rule of the domain: the money of the labor is the hours of it times
// the value of the hour. The money of the book counts whole pesos, and the
// money of the labor is whole pesos: the answer is rounded to the nearest
// peso.
func moneyOfTheHours(hours Hours, value ledger.Amount) ledger.Amount {
	if value <= 0 {
		return 0
	}

	return ledger.Amount(math.Round(hours.Total() * float64(value)))
}

// TheHourValueOf answers the value of the hour that the business declared.
//
// The rule of the domain: the value of the hour is what the owner declared,
// and the last declaration is the one that counts: the value of the hour
// of a market does not wait.
//
// The answer is zero when the business declared none: the value of the hour
// is never inferred, and the labor that needs it says that it does not
// know it.
func TheHourValueOf(labors []Labor) ledger.Amount {
	if len(labors) == 0 {
		return 0
	}

	latest := labors[0]

	for _, one := range labors[1:] {
		if one.Day.After(latest.Day) || (one.Day == latest.Day && one.ID > latest.ID) {
			latest = one
		}
	}

	return latest.HourValue
}

// LaborCost is the labor of a batch of a recipe.
//
// It is the hours of the steps of the recipe, in the three kinds of the
// labor, valued with the value of the hour the owner declared.
type LaborCost struct {
	// Hours are the hours of the steps of the recipe, in the three kinds
	// of the labor.
	Hours Hours

	// Value is the value of the hour that valued the labor, in whole
	// pesos.
	Value ledger.Amount

	// Money is the money of the labor of the batch, in whole pesos.
	Money ledger.Amount

	// Unknown says that the money of the labor is not derivable: the
	// owner has not declared the value of the hour, and the money is
	// never inferred.
	Unknown bool

	// Reason says why the money of the labor is not derivable, in the
	// words of the book.
	Reason string
}

// LaborOf derives the labor of a batch of the recipe.
//
// The rule of the domain: the labor of a batch is derived from the hours
// of its steps, and never typed. Every step adds the hours of its time to
// the kind of the labor it is: the hands, the oven or the waiting.
//
// The money of the labor is the hours of it times the value of the hour,
// and the value of the hour is what the owner declared: when she declared
// none, the money of the labor is not derivable, and the answer says so
// instead of guessing it.
//
// The error is the error of the domain: a step whose kind of labor the book
// does not know is not a step of the recipe, and a step of negative minutes
// is not a time.
func LaborOf(baked Recipe, value ledger.Amount) (LaborCost, error) {
	hours := Hours{}

	for _, one := range baked.Steps {
		if one.Minutes < 0 {
			return LaborCost{}, fmt.Errorf("%w: the step carries %d minutes", ErrInvalidMinutes, one.Minutes)
		}

		// The step the owner gave no time to is not a step of the labor:
		// the zero of the time is the absence of it.
		if one.Minutes == 0 {
			continue
		}

		kind, err := ParseLaborKind(string(one.Kind))
		if err != nil {
			return LaborCost{}, err
		}

		hours = hours.With(kind, float64(one.Minutes)/MinutesPerHour)
	}

	labor := LaborCost{Hours: hours, Value: value}

	if value <= 0 {
		labor.Unknown = true
		labor.Reason = "the owner has not declared the value of the hour"

		return labor, nil
	}

	labor.Money = moneyOfTheHours(hours, value)

	return labor, nil
}
