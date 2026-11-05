package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
)

// IngredientService is the use case that matches a name the owner said
// against the catalog of the names the business buys.
//
// The agent reads the sentence of the owner and asks the catalog what the
// name she said is: the ingredient it names, the options it offers when
// several answer it, or the news that the catalog does not know it.
//
// The use case never decides which ingredient the owner meant: the rule of
// the domain decides, and the use case only carries the names of the
// business to the catalog and the answer back to the agent.
type IngredientService struct {
	ingredients ingredient.Repository
}

// NewIngredientService builds the use case of the catalog.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
func NewIngredientService(ingredients ingredient.Repository) *IngredientService {
	return &IngredientService{ingredients: ingredients}
}

// Match answers the name the owner said against the catalog of the
// business.
//
// The rules are the ones of the domain: the use case reads the catalog and
// asks the domain to match the name. It never decides which ingredient the
// owner meant, and it never refuses a name the catalog does not know: a
// name the catalog does not know is not an error, it is a name that is
// new.
func (s *IngredientService) Match(ctx context.Context, request ports.MatchItems) (ingredient.Match, error) {
	if s == nil || s.ingredients == nil {
		return ingredient.Match{}, ports.ErrNoDatabase
	}

	items, err := s.ingredients.List(ctx, request.BusinessID)
	if err != nil {
		return ingredient.Match{}, err
	}

	catalog, err := ingredient.NewCatalog(request.BusinessID, items)
	if err != nil {
		return ingredient.Match{}, err
	}

	return catalog.Match(request.Name)
}

// Read answers the catalog of a business: what is known of every ingredient
// and what is still missing of each one.
//
// The catalog begins empty and grows with what the owner buys: the list is
// empty when the business bought nothing yet.
//
// The rules are the ones of the domain: the use case reads the catalog and
// hands it to the agent, and the domain decides what an ingredient is.
func (s *IngredientService) Read(ctx context.Context, request ports.ReadIngredients) ([]ingredient.Ingredient, error) {
	if s == nil || s.ingredients == nil {
		return nil, ports.ErrNoDatabase
	}

	return s.ingredients.List(ctx, request.BusinessID)
}

// Complete writes what the owner declared about an ingredient of the
// catalog, and answers with the ingredient as it is now.
//
// The rules are the ones of the domain: the use case reads the catalog of
// the business, asks the domain what the ingredient is, asks the domain
// what the declaration of the owner makes of it, and writes what the
// domain built. It never decides what an ingredient carries.
//
// The ingredient belongs to the business: the completion of an ingredient
// the catalog of the business does not hold is refused, and nothing is
// written.
func (s *IngredientService) Complete(ctx context.Context, request ports.CompleteIngredient) (ingredient.Ingredient, error) {
	if s == nil || s.ingredients == nil {
		return ingredient.Ingredient{}, ports.ErrNoDatabase
	}

	current, err := s.find(ctx, request.BusinessID, ingredient.ID(request.IngredientID))
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	declaration, err := declarationOf(request)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	completed, err := ingredient.Complete(current, declaration)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	return s.ingredients.Update(ctx, completed)
}

// find answers the ingredient of the catalog of the business.
//
// The rule of the domain: an ingredient of another business is not in the
// catalog of this one, and an ingredient that was never bought is not in
// it either. The error is ingredient.ErrNotFound, and the answer tells the
// agent to match the name against the catalog first.
func (s *IngredientService) find(ctx context.Context, own business.ID, id ingredient.ID) (ingredient.Ingredient, error) {
	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	catalog, err := ingredient.NewCatalog(own, items)
	if err != nil {
		return ingredient.Ingredient{}, err
	}

	found, held := catalog.Find(id)
	if !held {
		return ingredient.Ingredient{}, fmt.Errorf("%w: the ingredient %d is not one of the catalog of the business %d",
			ingredient.ErrNotFound, id, own)
	}

	return found, nil
}

// declarationOf turns what the agent sent into the declaration of the
// domain.
//
// The rules are the ones of the domain: the use case asks the domain to
// read the unit, to build the certification and to build the conversion,
// and it never builds them by itself.
//
// A field the owner did not declare stays absent: the ingredient keeps
// what it had.
func declarationOf(request ports.CompleteIngredient) (ingredient.Declaration, error) {
	declaration := ingredient.Declaration{
		PackageGrams: request.PackageGrams,
		Allergens:    request.Allergens,
		Aliases:      request.Aliases,
	}

	if unit := strings.TrimSpace(request.Unit); unit != "" {
		parsed, err := ingredient.ParseUnit(unit)
		if err != nil {
			return ingredient.Declaration{}, err
		}

		declaration.Unit = parsed
	}

	if request.CertificationName != "" || request.CertificationEvidence != "" {
		certification, err := ingredient.NewCertification(request.CertificationName, request.CertificationEvidence)
		if err != nil {
			return ingredient.Declaration{}, err
		}

		declaration.Certification = certification
	}

	if request.Conversion != 0 {
		conversion, err := ingredient.NewConversion(request.Conversion)
		if err != nil {
			return ingredient.Declaration{}, err
		}

		declaration.Conversion = conversion
	}

	return declaration, nil
}
