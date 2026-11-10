package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// RotationService is the use case that reads the rotation of the book of a
// business: the order in which the units that are still in the book leave
// it.
//
// The rotation follows the earliest expiry and the valuation follows the
// FIFO: the two rules are apart, and the use case never mixes them. The
// answer of the rotation carries no money.
//
// The units that are read are the ones that are still in the book: the
// units that left it - the ones that were sold, wasted, consumed or given
// away - are never offered again.
//
// The use case never decides the order: the domain builds the rotation, and
// the use case carries the units it read to it.
type RotationService struct {
	units   recipe.UnitInTheBookReader
	recipes recipe.Reader
	clock   ports.Clock
}

// NewRotationService builds the use case of the rotation.
//
// The reader of the units is the port of the storage that reads the units
// that are still in the book: the rotation holds the units the bakery
// still has. The recipes are the port of the storage of the recipes: the
// product the owner asks about is named with the name she reads, and the
// book never guesses which one she meant. The clock is the port of the
// time: the days that are left of a unit are counted from the day of the
// answer.
func NewRotationService(
	units recipe.UnitInTheBookReader,
	recipes recipe.Reader,
	clock ports.Clock,
) *RotationService {
	return &RotationService{units: units, recipes: recipes, clock: clock}
}

// Read answers the rotation of the book of the business.
//
// The units are the ones that are still in the book, and the answer carries
// the day every one of them expires, the days that are left of it and the
// product it belongs to.
//
// The business is required: the rotation belongs to a business, and the use
// case refuses the business that is not there.
//
// The product is optional: the zero identity with the empty name answer the
// rotation of the whole book of the business, and a product answers the
// rotation of that product alone.
//
// The rules are the ones of the domain: the use case reads the units of the
// book and it asks the domain to build the rotation. It never decides the
// order, and it never values anything.
func (s *RotationService) Read(ctx context.Context, request ports.ReadRotation) (recipe.Rotation, error) {
	if s == nil || s.units == nil || s.recipes == nil || s.clock == nil {
		return recipe.Rotation{}, ports.ErrNoDatabase
	}

	if request.BusinessID <= 0 {
		return recipe.Rotation{}, fmt.Errorf("%w: the rotation is of the business %d",
			recipe.ErrMissingBusiness, request.BusinessID)
	}

	product := recipe.ID(0)
	name := ""

	if request.RecipeID != 0 || strings.TrimSpace(request.RecipeName) != "" {
		baked, err := theRecipeOf(ctx, s.recipes, request.BusinessID, request.RecipeID, request.RecipeName)
		if err != nil {
			return recipe.Rotation{}, err
		}

		product = baked.ID
		name = baked.Name
	}

	units, err := s.units.ListInTheBook(ctx, request.BusinessID)
	if err != nil {
		return recipe.Rotation{}, err
	}

	return recipe.RotationOf(units, product, name, s.today()), nil
}

// today answers the day of today.
func (s *RotationService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
