package application

import (
	"context"
	"errors"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/store"
	"github.com/witsaba/paca/packages/domain/user"
)

// OnboardingService is the use case that completes the onboarding of a
// person.
//
// The onboarding is one operation: the person is registered, their personal
// business is opened with its accounts, and they move to the state of the
// people who use the product. The agent makes one call, and the service
// writes it in one transaction: a person whose book is open and whose
// accounts are half of them is a person PACA cannot tell the truth about.
//
// The use case knows the rules of the two contexts it orchestrates, and it
// calls them: the rule of the complete onboarding lives in the domain of the
// person, and the rule of the personal business and of its accounts lives in
// the domain of the book.
type OnboardingService struct {
	users    user.Repository
	business business.Repository
	accounts account.Repository
	work     store.UnitOfWork
}

// NewOnboardingService builds the use case of the onboarding.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
func NewOnboardingService(
	users user.Repository,
	businesses business.Repository,
	accounts account.Repository,
	work store.UnitOfWork,
) *OnboardingService {
	return &OnboardingService{
		users:    users,
		business: businesses,
		accounts: accounts,
		work:     work,
	}
}

// Complete registers the person, opens their personal business with its
// accounts, and moves them to the state of the people who use the product.
//
// The document ID and the name are required: the use case refuses a call
// that does not bring them. The phone and the email are optional.
//
// The call is safe to repeat: the person that is already registered is
// completed and not registered again, the business that is already open is
// kept and not opened again, and the accounts that are already there are
// read and not written again.
func (s *OnboardingService) Complete(ctx context.Context, documentID, fullName, phone, email string) (ports.Onboarding, error) {
	if s == nil || s.users == nil || s.business == nil || s.accounts == nil || s.work == nil {
		return ports.Onboarding{}, ports.ErrNoDatabase
	}

	id, err := user.ParseID(documentID)
	if err != nil {
		return ports.Onboarding{}, err
	}

	registration, err := user.NewRegistration(fullName, phone, email)
	if err != nil {
		return ports.Onboarding{}, err
	}

	// The person and the business of the person are read inside the
	// transaction, and the call is tried twice. Two calls of the same
	// person may arrive together: the one that loses the race finds the
	// business that the winner committed and completes the onboarding with
	// it, instead of failing with a refusal the person can do nothing
	// about.
	var outcome ports.Onboarding
	var failure error
	for attempt := 0; attempt < 2; attempt++ {
		failure = s.work.Do(ctx, func(ctx context.Context) error {
			// The state of the person is read before anything is written:
			// a transaction that PostgreSQL has aborted reads nothing.
			registered, err := s.users.Find(ctx, id)
			known := err == nil
			if err != nil && !errors.Is(err, user.ErrNotFound) {
				return err
			}

			opened, err := s.business.FindPersonalByOwner(ctx, id)
			hasBusiness := err == nil
			if err != nil && !errors.Is(err, business.ErrNotFound) {
				return err
			}

			person := registered
			if !known {
				created, err := s.users.Create(ctx, id, registration)
				if err != nil {
					return err
				}
				person = created
			}

			own := opened
			if !hasBusiness {
				personal, err := business.Personal(id)
				if err != nil {
					return err
				}

				created, err := s.business.Create(ctx, id, personal)
				if err != nil {
					return err
				}
				own = created
			}

			written, err := s.accountsOf(ctx, own, hasBusiness)
			if err != nil {
				return err
			}

			active := person
			if person.Status != user.StatusActive {
				moved, err := s.users.ChangeStatus(ctx, id, user.StatusActive, "")
				if err != nil {
					return err
				}
				active = moved
			}

			outcome = ports.Onboarding{User: active, Business: own, Accounts: written}
			return nil
		})
		if failure == nil {
			break
		}
		if !alreadyThere(failure) {
			break
		}
	}
	if failure != nil {
		return ports.Onboarding{}, failure
	}

	return outcome, nil
}

// alreadyThere reports whether the storage refused because the row is
// already there: another call of the same person won the race, and the
// onboarding completes with what that call wrote.
func alreadyThere(err error) bool {
	return errors.Is(err, business.ErrAlreadyExists) ||
		errors.Is(err, user.ErrAlreadyExists) ||
		errors.Is(err, account.ErrAlreadyExists)
}

// accountsOf answers the accounts of the business: the default set of the
// personal business when the business was just opened, and the accounts that
// are already there when it was open before.
func (s *OnboardingService) accountsOf(ctx context.Context, own business.Business, existed bool) ([]account.Account, error) {
	if existed {
		return s.accounts.List(ctx, own.ID)
	}

	seed := account.DefaultPersonalAccounts()
	accounts := make([]account.Account, 0, len(seed))
	for _, one := range seed {
		value, err := account.New(own.ID, one.Kind, one.Name, "", nil)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, value)
	}

	return s.accounts.CreateMany(ctx, accounts)
}
