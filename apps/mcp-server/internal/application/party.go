package application

import (
	"context"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/party"
)

// PartyService is the use case that opens a party of the business and that
// reads the parties it knows.
//
// The party is the other side of the bread: the person, the store or the
// farm that buys from the bakery or that the bakery buys from. The use case
// carries what the agent said to the domain, and the answer of the domain
// back to the agent.
//
// The use case never decides what a party is: the kind, the roles, the
// legal human that represents it and the way of reaching it are the rules of
// the domain, and the use case only asks the domain.
type PartyService struct {
	parties    party.Repository
	businesses business.Repository
}

// NewPartyService builds the use case of the parties.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The storage of the
// businesses is needed too: the parties belong to a business of its own,
// and the use case reads the business before it writes the party.
func NewPartyService(parties party.Repository, businesses business.Repository) *PartyService {
	return &PartyService{parties: parties, businesses: businesses}
}

// Open writes the party of the business and answers it with it.
//
// The rules are the ones of the domain: the use case reads the business
// that owns the party, asks the domain whether a business of that kind owns
// parties, asks the domain to read the kind, the roles and the legal human
// of the party, and writes what the domain built. It never decides what a
// party is.
func (s *PartyService) Open(ctx context.Context, request ports.OpenParty) (party.Party, error) {
	if s == nil || s.parties == nil || s.businesses == nil {
		return party.Party{}, ports.ErrNoDatabase
	}

	owner, err := s.businesses.Find(ctx, business.ID(request.BusinessID))
	if err != nil {
		return party.Party{}, err
	}

	if err := party.OwnedBy(owner.Kind); err != nil {
		return party.Party{}, err
	}

	kind, err := party.ParseKind(request.Kind)
	if err != nil {
		return party.Party{}, err
	}

	roles, err := party.ParseRoles(request.Roles)
	if err != nil {
		return party.Party{}, err
	}

	// The legal human is a fact of the party, and it is optional: the farm
	// may carry none and the person refuses one. The domain decides which
	// of them it is, and it is the party that refuses what breaks its
	// rule.
	human := party.LegalHuman{}
	if request.LegalHumanName != "" || request.LegalFigure != "" {
		human, err = party.NewLegalHuman(request.LegalHumanName, request.LegalFigure)
		if err != nil {
			return party.Party{}, err
		}
	}

	built, err := party.New(
		owner.ID,
		kind,
		request.Name,
		roles,
		request.Phone,
		request.Email,
		request.Address,
		human,
	)
	if err != nil {
		return party.Party{}, err
	}

	return s.parties.Create(ctx, built)
}

// Read answers the parties of the business: the person, the store and the
// farm that it knows, with their roles and their facts.
//
// The rules are the ones of the domain: the use case reads the parties of
// the business and hands them to the agent. The name of a party is not
// unique inside the business, and the answer holds every one of them.
func (s *PartyService) Read(ctx context.Context, request ports.ReadParties) ([]party.Party, error) {
	if s == nil || s.parties == nil {
		return nil, ports.ErrNoDatabase
	}

	return s.parties.List(ctx, business.ID(request.BusinessID))
}
