package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/learning"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// The names the database gives to the constraints of the table of the
// learning log.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	// learningsBusinessConstraint is the name of the foreign key that refuses
	// the entry of a business that is not there.
	learningsBusinessConstraint = "learnings_business_id_fkey"

	// learningsBatchConstraint is the name of the composite foreign key that
	// refuses the entry about the batch of another business.
	learningsBatchConstraint = "learnings_batch_of_its_business_fkey"

	// learningsProductionConstraint is the name of the composite foreign key
	// that refuses the entry about the production of another business.
	learningsProductionConstraint = "learnings_production_of_its_business_fkey"

	// learningsProductionBatchConstraint is the name of the composite foreign
	// key that refuses the entry about a production that is not a run of the
	// batch it names.
	learningsProductionBatchConstraint = "learnings_production_of_its_batch_fkey"

	// learningsAnswerConstraint is the name of the composite foreign key that
	// refuses the answer of the entry of another business.
	learningsAnswerConstraint = "learnings_answer_of_its_business_fkey"

	// learningsAnsweredOnceIndex is the name of the partial unique index that
	// refuses the second answer of the same entry.
	learningsAnsweredOnceIndex = "learnings_answered_once"
)

// LearningRepository is the adapter of the storage of the learning log.
//
// The entry is one row: the day it is about, the batch, the production when
// it is about one run of the batch, what happened, what was learned and the
// entry it answers.
//
// The log is never rewritten: the adapter writes entries and reads them
// back, and it holds no update and no delete of the words of an entry.
//
// The row never decides what an entry is: the model is built by the domain,
// through its own constructors.
type LearningRepository struct {
	pool *Pool
}

// NewLearningRepository returns the repository of the learning log, bound to
// the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewLearningRepository(pool *Pool) (*LearningRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &LearningRepository{pool: pool}, nil
}

// Create writes the entry and answers it with the identity that the database
// generated.
//
// The production and the entry it answers are written only when the entry
// names them, and they are NULL otherwise: the entry about the batch alone
// carries no production, and the entry that answers no other one answers
// none.
//
// The error is learning.ErrBusinessNotFound when the business of the entry is
// not there, learning.ErrRecipeOfAnotherBusiness when the batch is not a
// batch of the business, learning.ErrProductionOfAnotherBusiness when the
// production is not a production of the business, learning.ErrAnotherBatch
// when the production is not a run of the named batch,
// learning.ErrEntryOfAnotherBusiness when the entry it answers is not an
// entry of the business, learning.ErrAlreadyAnswered when the entry it
// answers was answered already, learning.ErrInvalidValue when the row breaks
// a rule of the table, and learning.ErrNoStorage when the storage is not
// open.
func (r *LearningRepository) Create(ctx context.Context, written learning.Entry) (learning.Entry, error) {
	if r == nil || r.pool == nil {
		return learning.Entry{}, learning.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateLearning(ctx, db.CreateLearningParams{
		BusinessID:   int64(written.BusinessID),
		Day:          dateOf(written.Day),
		RecipeID:     int64(written.RecipeID),
		ProductionID: theProductionOfTheEntry(written.ProductionID),
		Happened:     written.Happened,
		Learned:      written.Learned,
		AnswersID:    theEntryItAnswers(written.Answers),
	})
	if err != nil {
		return learning.Entry{}, writeLearningError(err)
	}

	return learningOf(row.ID, row.BusinessID, row.RecipeID, row.Day, row.ProductionID, row.Happened,
		row.Learned, row.AnswersID, row.CreatedAt, row.RecipeName)
}

// List returns the entries of the business, in the order they were written,
// each one with the batch it names and the entry it answers.
//
// The list holds the entries of the business and of no other one, and it is
// empty when the business wrote nothing yet. An entry is never deleted: the
// list holds the ones that were answered too.
//
// The error is learning.ErrNoStorage when the storage is not open.
func (r *LearningRepository) List(ctx context.Context, own business.ID) ([]learning.Entry, error) {
	if r == nil || r.pool == nil {
		return nil, learning.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListLearningsByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeLearningError(err)
	}

	entries := make([]learning.Entry, 0, len(rows))

	for _, row := range rows {
		built, err := learningOf(row.ID, row.BusinessID, row.RecipeID, row.Day, row.ProductionID,
			row.Happened, row.Learned, row.AnswersID, row.CreatedAt, row.RecipeName)
		if err != nil {
			return nil, err
		}

		entries = append(entries, built)
	}

	return entries, nil
}

// learningOf builds the entry of the log of a row and asks the domain whether
// it is whole.
//
// The row never decides what an entry is: a row that another writer broke is
// refused by the domain, and the adapter says it with
// learning.ErrInvalidValue.
func learningOf(
	id, businessID, recipeID int64,
	day pgtype.Date,
	productionID pgtype.Int8,
	happened, learned string,
	answersID pgtype.Int8,
	createdAt pgtype.Timestamptz,
	recipeName string,
) (learning.Entry, error) {
	built, err := learning.NewFromFacts(learning.Facts{
		ID:           learning.ID(id),
		BusinessID:   business.ID(businessID),
		Day:          ledger.NewDate(day.Time),
		RecipeID:     recipe.ID(recipeID),
		RecipeName:   recipeName,
		ProductionID: productionIDOfTheRow(productionID),
		Happened:     happened,
		Learned:      learned,
		Answers:      answersIDOfTheRow(answersID),
	})
	if err != nil {
		return learning.Entry{}, fmt.Errorf("%w: the row %d of the learnings breaks the domain: %w",
			learning.ErrInvalidValue, id, err)
	}

	built.CreatedAt = timestamp(createdAt)

	return built, nil
}

// theProductionOfTheEntry returns the production the entry names, as the
// column of the table holds it: the production, or NULL when the entry is
// about the batch alone.
func theProductionOfTheEntry(id recipe.ProductionID) pgtype.Int8 {
	if id == 0 {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: int64(id), Valid: true}
}

// theEntryItAnswers returns the entry the new one answers, as the column of
// the table holds it: the entry, or NULL when the new one answers no other
// one.
func theEntryItAnswers(id learning.ID) pgtype.Int8 {
	if id == 0 {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: int64(id), Valid: true}
}

// productionIDOfTheRow answers the production the row names, and the absence
// of it when the column is NULL.
func productionIDOfTheRow(id pgtype.Int8) recipe.ProductionID {
	if !id.Valid {
		return 0
	}

	return recipe.ProductionID(id.Int64)
}

// answersIDOfTheRow answers the entry the row answers, and the absence of it
// when the column is NULL.
func answersIDOfTheRow(id pgtype.Int8) learning.ID {
	if !id.Valid {
		return 0
	}

	return learning.ID(id.Int64)
}

// writeLearningError translates the truth of the database into the errors of
// the port of the log.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth: the business of the entry is a business that is there, the batch and
// the production are of that business, the production is a run of the batch
// the entry names, the entry it answers is an entry of that business, and it
// was not answered already.
func writeLearningError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return learning.ErrInvalidValue
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case foreignKeyViolation:
		switch pgErr.ConstraintName {
		case learningsBusinessConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				learning.ErrBusinessNotFound, pgErr.ConstraintName)
		case learningsBatchConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				learning.ErrRecipeOfAnotherBusiness, pgErr.ConstraintName)
		case learningsProductionConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				learning.ErrProductionOfAnotherBusiness, pgErr.ConstraintName)
		case learningsProductionBatchConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				learning.ErrAnotherBatch, pgErr.ConstraintName)
		case learningsAnswerConstraint:
			return fmt.Errorf("%w: the constraint %q refused the row",
				learning.ErrEntryOfAnotherBusiness, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the constraint %q refused the row", learning.ErrInvalidValue, pgErr.ConstraintName)
	case uniqueViolation:
		if pgErr.ConstraintName == learningsAnsweredOnceIndex {
			return fmt.Errorf("%w: the index %q refused the row",
				learning.ErrAlreadyAnswered, pgErr.ConstraintName)
		}

		return fmt.Errorf("%w: the constraint %q refused the row", learning.ErrInvalidValue, pgErr.ConstraintName)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", learning.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
