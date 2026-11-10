package learning

import "errors"

// The errors of the domain of the learning log.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
//
// Every error names ONE rule of the log, and the message of the agent says
// that rule in the words of the caller.
var (
	// ErrMissingBusiness is the error of an entry of no business: the log is
	// the memory of a business, and a memory of nobody is not a log.
	ErrMissingBusiness = errors.New("the entry of the log belongs to no business")

	// ErrMissingDay is the error of an entry of no day: an entry of no time
	// is an entry of nothing.
	ErrMissingDay = errors.New("the entry of the log carries no day")

	// ErrMissingRecipe is the error of an entry about no batch: the log
	// records what happened with the bread, and the bread is a recipe the
	// business makes.
	ErrMissingRecipe = errors.New("the entry of the log names the batch it is about")

	// ErrRecipeOfAnotherBusiness is the error of an entry about the batch of
	// another business: the entry is about the bread of the business that
	// owns the log.
	ErrRecipeOfAnotherBusiness = errors.New("the batch of the entry belongs to another business")

	// ErrProductionOfAnotherBusiness is the error of an entry about the
	// production of another business: the production is a run of the bread
	// of the business that owns the log.
	ErrProductionOfAnotherBusiness = errors.New("the production of the entry is not a production of that business")

	// ErrAnotherBatch is the error of an entry about a production that is
	// not a run of the batch the entry names: the book never carries two
	// truths for the same fact.
	ErrAnotherBatch = errors.New("the production of the entry is a run of another batch")

	// ErrMissingHappened is the error of an entry that says nothing
	// happened: an entry that says nothing is not an entry of the log.
	ErrMissingHappened = errors.New("the entry of the log says what happened")

	// ErrMissingLearned is the error of an entry that taught nothing: the
	// log is what PACA learns from, and an entry with no lesson is no
	// lesson.
	ErrMissingLearned = errors.New("the entry of the log says what was learned")

	// ErrEntryNotFound is the error of an answer to an entry the log does not
	// hold: the new entry answers an entry that is there.
	ErrEntryNotFound = errors.New("there is no entry of the log with that identity")

	// ErrEntryOfAnotherBusiness is the error of an answer to the entry of
	// another business: the log of a business answers its own entries.
	ErrEntryOfAnotherBusiness = errors.New("the entry the log answers belongs to another business")

	// ErrAlreadyAnswered is the error of an answer to an entry that was
	// answered already: the log is a chain and not a tree, and two answers of
	// the same entry would leave the owner with two truths.
	ErrAlreadyAnswered = errors.New("the entry of the log was answered already")

	// ErrWrongBusiness is the error of a log of a business that does not
	// keep its lessons: the log belongs to the business that sells the bread.
	ErrWrongBusiness = errors.New("the log belongs to a business of its own")

	// ErrBusinessNotFound is the error of the business of the log that is not
	// there.
	ErrBusinessNotFound = errors.New("the business of the log is not there")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no storage
	// behind it.
	ErrNoStorage = errors.New("the storage of the log is not open")
)

// Errors returns the errors of the domain of the learning log.
func Errors() []error {
	return []error{
		ErrMissingBusiness,
		ErrMissingDay,
		ErrMissingRecipe,
		ErrRecipeOfAnotherBusiness,
		ErrProductionOfAnotherBusiness,
		ErrAnotherBatch,
		ErrMissingHappened,
		ErrMissingLearned,
		ErrEntryNotFound,
		ErrEntryOfAnotherBusiness,
		ErrAlreadyAnswered,
		ErrWrongBusiness,
		ErrBusinessNotFound,
		ErrInvalidValue,
		ErrNoStorage,
	}
}
