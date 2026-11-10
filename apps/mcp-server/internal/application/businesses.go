package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/store"
	"github.com/witsaba/paca/packages/domain/user"
)

// BusinessService is the use case of the businesses of a person.
//
// A person may own more than one business: the Personal Business, that the
// onboarding opens, and the others, that the owner opens when they open
// theirs. The bakery of the owner is one of those, and it has a book of its
// own.
//
// The use case knows the rules of the two contexts it orchestrates, and it
// calls them: the rule of the name of a business lives in the domain of the
// business, and the rule of the accounts of the book lives in the domain of
// the book.
type BusinessService struct {
	businesses business.Repository
	accounts   account.Repository
	work       store.UnitOfWork
}

// NewBusinessService builds the use case of the businesses.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
func NewBusinessService(
	businesses business.Repository,
	accounts account.Repository,
	work store.UnitOfWork,
) *BusinessService {
	return &BusinessService{
		businesses: businesses,
		accounts:   accounts,
		work:       work,
	}
}

// Open opens a business of the owner and writes the accounts of its book.
//
// The document ID of the owner is required, and the name of the business is
// required: a business without a name has nothing to show in the book of
// its owner. The currency is optional, and it is the currency of the domain
// -- the peso -- when the caller has no opinion about it.
//
// The kind of the business is always the kind of a business that is not the
// personal one: the Personal Business of a person is opened by the
// onboarding, and never here.
//
// The name of a business is taken inside the businesses of its owner: the
// use case reads the businesses of the owner and refuses the duplicate. The
// table of the businesses does not hold that rule yet, and the domain holds
// it.
//
// The business and its accounts are written in one transaction: a book that
// starts with half of its accounts lies about what the owner has.
func (s *BusinessService) Open(ctx context.Context, documentID, name, currency string) (ports.RegisteredBusiness, error) {
	if s == nil || s.businesses == nil || s.accounts == nil || s.work == nil {
		return ports.RegisteredBusiness{}, ports.ErrNoDatabase
	}

	owner, err := user.ParseID(documentID)
	if err != nil {
		return ports.RegisteredBusiness{}, err
	}

	opened, err := business.New(owner, business.KindBusiness, name, s.currencyOf(currency))
	if err != nil {
		return ports.RegisteredBusiness{}, err
	}

	// The businesses of the owner are read before the transaction opens,
	// because a transaction that PostgreSQL has aborted reads nothing.
	owned, err := s.businesses.ListByOwner(ctx, owner)
	if err != nil {
		return ports.RegisteredBusiness{}, err
	}
	if s.nameIsTaken(owned, opened.Name) {
		return ports.RegisteredBusiness{}, fmt.Errorf("%w: the name %q is taken inside the businesses of its owner",
			business.ErrAlreadyExists, opened.Name)
	}

	var outcome ports.RegisteredBusiness
	err = s.work.Do(ctx, func(ctx context.Context) error {
		created, err := s.businesses.Create(ctx, owner, opened)
		if err != nil {
			return err
		}

		written, err := s.accountsOf(ctx, created)
		if err != nil {
			return err
		}

		outcome = ports.RegisteredBusiness{Business: created, Accounts: written}
		return nil
	})
	if err != nil {
		return ports.RegisteredBusiness{}, err
	}

	return outcome, nil
}

// List returns the businesses of the owner, in the order of their identity.
//
// The document ID of the owner is required. The list is empty when the
// person holds no business at all, and the personal business of the person
// is one of them when they hold one.
func (s *BusinessService) List(ctx context.Context, documentID string) ([]business.Business, error) {
	if s == nil || s.businesses == nil {
		return nil, ports.ErrNoDatabase
	}

	owner, err := user.ParseID(documentID)
	if err != nil {
		return nil, err
	}

	return s.businesses.ListByOwner(ctx, owner)
}

// currencyOf answers the currency of the book of the business: the one the
// caller names, and the currency of the domain when the caller names none.
func (s *BusinessService) currencyOf(currency string) string {
	if strings.TrimSpace(currency) == "" {
		return business.DefaultCurrency
	}

	return currency
}

// nameIsTaken reports whether the name of the business is already one of the
// businesses of the owner.
//
// The rule is the one of the domain: the name of a business is taken inside
// the businesses of its owner, and the same name in the businesses of
// another owner is free.
func (s *BusinessService) nameIsTaken(owned []business.Business, name string) bool {
	for _, one := range owned {
		if one.Name == name {
			return true
		}
	}

	return false
}

// accountsOf answers the accounts of the book of the business: the set the
// domain owns for a business of the kind business.
//
// The set is written in the transaction of the caller: the book of a
// business starts whole or not at all.
func (s *BusinessService) accountsOf(ctx context.Context, own business.Business) ([]account.Account, error) {
	seed := account.DefaultBusinessAccounts()

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
