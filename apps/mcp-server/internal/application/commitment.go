package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/commitment"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// CommitmentService is the use case of the promises of the bakery.
//
// The commitment is a fact of PACA: what is due, in the words of the owner,
// and the day it is due. The commitment that was met is kept, with the day
// it was met.
//
// The ALARM is the work of the agent. PACA holds the fact of what is due and
// the day it is due, and the reading of the calendar is what the agent reads
// to ring: the calendar of a day answers what is due on it and every promise
// that is still owed.
//
// The use case never decides what a promise is: the business that owns it,
// what is due, the day it is due and the day it was met are the rules of the
// domain, and the use case only asks the domain.
type CommitmentService struct {
	commitments commitment.Repository
	businesses  business.Repository
}

// NewCommitmentService builds the use case of the commitments.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The businesses are
// needed too: the promise belongs to a business of its own, and the Personal
// Business of a person holds the money of that person and not the promises of
// the bakery.
func NewCommitmentService(commitments commitment.Repository, businesses business.Repository) *CommitmentService {
	return &CommitmentService{
		commitments: commitments,
		businesses:  businesses,
	}
}

// Record writes the promise of the business and answers it with it.
//
// The rules are the ones of the domain: the use case reads the business that
// owes the promise, asks the domain whether a business of that kind holds
// promises, reads the day it is due and asks the domain to build the promise.
// It never decides what a promise is.
//
// The promise is written OPEN: it carries no day of meeting, because nobody
// said it was met.
func (s *CommitmentService) Record(ctx context.Context, request ports.RecordCommitment) (commitment.Commitment, error) {
	if s == nil || s.commitments == nil || s.businesses == nil {
		return commitment.Commitment{}, ports.ErrNoDatabase
	}

	owner, err := s.theBusiness(ctx, request.BusinessID)
	if err != nil {
		return commitment.Commitment{}, err
	}

	dueOn, err := theDayOfThePromise(request.DueOn)
	if err != nil {
		return commitment.Commitment{}, err
	}

	built, err := commitment.New(owner.ID, request.What, dueOn)
	if err != nil {
		return commitment.Commitment{}, err
	}

	return s.commitments.Create(ctx, built)
}

// Meet writes the day the promise was met and answers the promise that was
// kept.
//
// The rules are the ones of the domain: the use case reads the business that
// owes the promise, reads the promises the business holds, finds the one the
// agent named by its own identity, reads the day it was met and asks the
// domain for the promise that was met. It never decides whether a promise may
// be met.
func (s *CommitmentService) Meet(ctx context.Context, request ports.MeetCommitment) (commitment.Commitment, error) {
	if s == nil || s.commitments == nil || s.businesses == nil {
		return commitment.Commitment{}, ports.ErrNoDatabase
	}

	owner, err := s.theBusiness(ctx, request.BusinessID)
	if err != nil {
		return commitment.Commitment{}, err
	}

	metOn, err := theDayOfThePromise(request.MetOn)
	if err != nil {
		return commitment.Commitment{}, err
	}

	promise, err := s.thePromise(ctx, owner.ID, commitment.ID(request.CommitmentID))
	if err != nil {
		return commitment.Commitment{}, err
	}

	met, err := promise.Met(metOn)
	if err != nil {
		return commitment.Commitment{}, err
	}

	return s.commitments.Meet(ctx, met)
}

// Read answers the calendar of the day: the promises due on it and the ones
// that are still owed.
//
// The rules are the ones of the domain: the use case reads the business whose
// promises are asked for, asks the domain whether a business of that kind
// holds promises, reads the day of the calendar, hands the promises of the
// business to the domain, and the domain answers the calendar.
//
// The reading guards what it uses: the business and the promises of the
// business. It asks for no party and for no product: the promise names them
// in the words of the owner, and the book copies neither.
func (s *CommitmentService) Read(ctx context.Context, request ports.ReadCommitments) (commitment.Calendar, error) {
	if s == nil || s.commitments == nil || s.businesses == nil {
		return commitment.Calendar{}, ports.ErrNoDatabase
	}

	owner, err := s.theBusiness(ctx, request.BusinessID)
	if err != nil {
		return commitment.Calendar{}, err
	}

	day, err := theDayOfThePromise(request.Day)
	if err != nil {
		return commitment.Calendar{}, err
	}

	held, err := s.commitments.List(ctx, owner.ID)
	if err != nil {
		return commitment.Calendar{}, err
	}

	return commitment.TheCalendar(day, held), nil
}

// theBusiness answers the business that owes the promise.
//
// The rules are the ones of the domain: the commitments belong to a business
// of its own, and the business that is not there is refused with the refusal
// of the promises: the agent is told about the business of the promise, and
// never about the personal business of a person nor about the onboarding.
//
// The error is commitment.ErrBusinessNotFound when the business is not there,
// and commitment.ErrWrongBusiness when the business does not hold promises.
func (s *CommitmentService) theBusiness(ctx context.Context, id business.ID) (business.Business, error) {
	owner, err := s.businesses.Find(ctx, id)
	if err != nil {
		if errors.Is(err, business.ErrNotFound) {
			return business.Business{}, fmt.Errorf("%w: the business %d", commitment.ErrBusinessNotFound, id)
		}

		return business.Business{}, err
	}

	if err := commitment.OwnedBy(owner.Kind); err != nil {
		return business.Business{}, err
	}

	return owner, nil
}

// thePromise answers the promise the agent named, by its own identity.
//
// The rule of the domain: the identity of a promise is its own identity, and
// never its text. Two promises that say the same thing on the same day are
// two promises, and the book never merges them.
//
// The promises of the business are the ones the business holds: the promise of
// another business is not in the list, and it is refused as a promise that is
// not there.
//
// The error is commitment.ErrNotFound when the business holds no promise with
// that identity.
func (s *CommitmentService) thePromise(ctx context.Context, own business.ID, id commitment.ID) (commitment.Commitment, error) {
	held, err := s.commitments.List(ctx, own)
	if err != nil {
		return commitment.Commitment{}, err
	}

	for _, one := range held {
		if one.ID == id {
			return one, nil
		}
	}

	return commitment.Commitment{}, fmt.Errorf("%w: the business %d holds no commitment with the identity %d",
		commitment.ErrNotFound, own, id)
}

// theDayOfThePromise reads a day of the promise: the day it is due, or the
// day it was met.
//
// The rule of the domain: the promise carries its day, and the book never
// invents one. The use case does not hold the rule and it does not
// manufacture its refusal: a day the agent names none of travels as the empty
// day, and the domain refuses it in its own words.
//
// The error is ledger.ErrInvalidDate when the day is not a day of the book.
func theDayOfThePromise(day string) (ledger.Date, error) {
	trimmed := strings.TrimSpace(day)
	if trimmed == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(trimmed)
}
