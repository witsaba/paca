package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/learning"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// LearningService is the use case of the learning log of the bakery.
//
// The log is the memory of the errors and the wins of the owner: what
// happened, in her words, and what was learned from it. It is not a sale and
// it is not a movement of the book: the log carries no money and it never
// moves a peso.
//
// The log is NEVER REWRITTEN. A correction is a NEW entry that answers the
// old one, and the old one keeps its own words: the use case writes entries
// and reads them, and it holds no update and no delete of an entry.
//
// The use case never decides what an entry is: the business that owns it, the
// day, the batch, the production, the words of the owner and the entry it
// answers are the rules of the domain, and the use case only asks the domain.
type LearningService struct {
	learnings   learning.Repository
	businesses  business.Repository
	recipes     recipe.Reader
	productions recipe.ProductionReader
}

// NewLearningService builds the use case of the learning log.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The businesses are
// needed too: the log belongs to a business of its own, and the Personal
// Business of a person holds the money of that person and not the lessons of
// the bakery. The recipes and the productions are read because the entry
// NAMES the batch it is about and the run of it: the recipe and the
// production stay the owners of their own facts.
func NewLearningService(
	learnings learning.Repository,
	businesses business.Repository,
	recipes recipe.Reader,
	productions recipe.ProductionReader,
) *LearningService {
	return &LearningService{
		learnings:   learnings,
		businesses:  businesses,
		recipes:     recipes,
		productions: productions,
	}
}

// Record writes the entry of the log and answers it with it.
//
// The rules are the ones of the domain: the use case reads the business that
// owns the memory, asks the domain whether a business of that kind keeps its
// lessons, reads the batch the entry is about, reads the productions of the
// business to find the run the entry names, reads the log to find the entry it
// answers, and asks the domain to build the entry. It never decides what an
// entry of the log is.
//
// The correction is a NEW entry: the entry that was answered is not touched,
// and the chain of the answers says which word is the last one.
func (s *LearningService) Record(ctx context.Context, request ports.RecordLearning) (learning.Entry, error) {
	if s == nil || s.learnings == nil || s.businesses == nil || s.recipes == nil || s.productions == nil {
		return learning.Entry{}, ports.ErrNoDatabase
	}

	owner, err := s.theBusinessOfTheLog(ctx, request.BusinessID)
	if err != nil {
		return learning.Entry{}, err
	}

	day, err := theDayOfTheEntry(request.Day)
	if err != nil {
		return learning.Entry{}, err
	}

	batch, err := s.theBatch(ctx, request.RecipeID)
	if err != nil {
		return learning.Entry{}, err
	}

	production, err := s.theProduction(ctx, owner.ID, request.ProductionID)
	if err != nil {
		return learning.Entry{}, err
	}

	answers, err := s.theEntryItAnswers(ctx, owner.ID, request.AnswersID)
	if err != nil {
		return learning.Entry{}, err
	}

	built, err := learning.New(owner.ID, day, batch, production, request.Happened, request.Learned, answers)
	if err != nil {
		return learning.Entry{}, err
	}

	return s.learnings.Create(ctx, built)
}

// Read answers the log of the business: the entries in the order they were
// written, each one with the entry it answers, and the ones that are the last
// word of their chain.
//
// The rules are the ones of the domain: the use case reads the business whose
// memory is asked for, asks the domain whether a business of that kind keeps
// its lessons, reads the entries of the business and hands them to the domain,
// and the domain answers the log.
func (s *LearningService) Read(ctx context.Context, request ports.ReadLearnings) (learning.Log, error) {
	if s == nil || s.learnings == nil || s.businesses == nil {
		return learning.Log{}, ports.ErrNoDatabase
	}

	owner, err := s.theBusinessOfTheLog(ctx, request.BusinessID)
	if err != nil {
		return learning.Log{}, err
	}

	held, err := s.learnings.List(ctx, owner.ID)
	if err != nil {
		return learning.Log{}, err
	}

	return learning.TheLog(held), nil
}

// theBusinessOfTheLog answers the business that owns the memory.
//
// The rules are the ones of the domain: the log belongs to a business of its
// own, and the business that is not there is refused with the refusal of the
// log: the agent is told about the business of the log, and never about the
// personal business of a person nor about the onboarding.
//
// The error is learning.ErrBusinessNotFound when the business is not there,
// and learning.ErrWrongBusiness when the business does not keep its lessons.
func (s *LearningService) theBusinessOfTheLog(ctx context.Context, id business.ID) (business.Business, error) {
	owner, err := s.businesses.Find(ctx, id)
	if err != nil {
		if errors.Is(err, business.ErrNotFound) {
			return business.Business{}, fmt.Errorf("%w: the business %d", learning.ErrBusinessNotFound, id)
		}

		return business.Business{}, err
	}

	if err := learning.OwnedBy(owner.Kind); err != nil {
		return business.Business{}, err
	}

	return owner, nil
}

// theBatch answers the batch the entry is about.
//
// The rule of the domain: the entry is about a batch, and the batch is a
// recipe of the business. The recipe that is not there is refused with the
// refusal of the log, and the domain refuses it when the agent names no batch
// at all: the use case asks for no recipe when the agent names none, and it
// manufactures no refusal of its own.
//
// The error is learning.ErrMissingRecipe when the agent named a recipe the
// book does not hold.
func (s *LearningService) theBatch(ctx context.Context, id int64) (recipe.Recipe, error) {
	if id == 0 {
		return recipe.Recipe{}, nil
	}

	found, err := s.recipes.Find(ctx, recipe.ID(id))
	if err != nil {
		if errors.Is(err, recipe.ErrNotFound) {
			return recipe.Recipe{}, fmt.Errorf("%w: the book holds no batch %d", learning.ErrMissingRecipe, id)
		}

		return recipe.Recipe{}, err
	}

	return found, nil
}

// theProduction answers the run of the batch the entry is about, when the
// agent names one.
//
// The rule of the domain: the production is a production of the business, and
// it is a run of the NAMED batch. The productions of the business are the ones
// the business holds, so a production of another business is not one of them,
// and the domain refuses the production that is not a run of the batch the
// entry names.
//
// The error is learning.ErrProductionOfAnotherBusiness when the business holds
// no production with that identity.
func (s *LearningService) theProduction(
	ctx context.Context,
	own business.ID,
	id int64,
) (*recipe.Production, error) {
	if id == 0 {
		return nil, nil
	}

	held, err := s.productions.List(ctx, own)
	if err != nil {
		return nil, err
	}

	for _, one := range held {
		if int64(one.ID) == id {
			run := one
			return &run, nil
		}
	}

	return nil, fmt.Errorf("%w: the business %d holds no production %d",
		learning.ErrProductionOfAnotherBusiness, own, id)
}

// theEntryItAnswers answers the entry the new one answers, when the agent
// names one.
//
// The rules of the answer are the ones of the domain, and the use case asks
// the log for them: the entry that is answered exists, it is of the business,
// and it was not answered already. The use case decides none of them.
func (s *LearningService) theEntryItAnswers(
	ctx context.Context,
	own business.ID,
	id int64,
) (*learning.ID, error) {
	if id == 0 {
		return nil, nil
	}

	held, err := s.learnings.List(ctx, own)
	if err != nil {
		return nil, err
	}

	answered, err := learning.TheLog(held).Answering(learning.ID(id), own)
	if err != nil {
		return nil, err
	}

	answer := answered.ID

	return &answer, nil
}

// theDayOfTheEntry reads the day the entry is about.
//
// The rule of the domain: the entry carries its day, and the log never invents
// one. The use case does not hold the rule and it does not manufacture its
// refusal: a day the agent names none of travels as the empty day, and the
// domain refuses it in its own words.
//
// The error is ledger.ErrInvalidDate when the day is not a day of the book.
func theDayOfTheEntry(day string) (ledger.Date, error) {
	trimmed := strings.TrimSpace(day)
	if trimmed == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(trimmed)
}
