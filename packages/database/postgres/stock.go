package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// StockRepository is the adapter of the storage of the layers of the stock.
//
// The purchase is the fact of what the business bought, and the layer is
// derived from it: the adapter writes the layer with the purchase it comes
// from, and it reads the layers back in the order they leave the stock.
//
// The row never decides what a layer is: the domain checks it, and a row
// that another writer broke is refused.
type StockRepository struct {
	pool *Pool
}

// NewStockRepository returns the repository of the layers, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewStockRepository(pool *Pool) (*StockRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &StockRepository{pool: pool}, nil
}

// Record writes the layers of a purchase, and answers them with the
// identities that the database generated.
//
// The position of a layer is the position of the line of the purchase it
// comes from: a layer is derived from one line and from no other, and the
// unique index of the table holds the rule as a defense in depth.
//
// The error is purchase.ErrInvalidValue when the row breaks a rule of the
// table, purchase.ErrNotFound when the purchase or the ingredient of the
// layer is not there, and purchase.ErrNoStorage when the storage is not
// open.
func (r *StockRepository) Record(ctx context.Context, layers []purchase.Layer) ([]purchase.Layer, error) {
	if r == nil || r.pool == nil {
		return nil, purchase.ErrNoStorage
	}

	written := make([]purchase.Layer, 0, len(layers))

	for _, one := range layers {
		position, err := positionOf(one.Position)
		if err != nil {
			return nil, err
		}

		row, err := r.pool.queriesFor(ctx).CreateStockLayer(ctx, db.CreateStockLayerParams{
			BusinessID:   int64(one.BusinessID),
			IngredientID: int64(one.IngredientID),
			PurchaseID:   int64(one.PurchaseID),
			Position:     position,
			Name:         one.Name,
			Day:          dateOf(one.Day),
			Quantity:     one.Bought.Quantity,
			Unit:         one.Bought.Unit,
			Remaining:    one.Left.Quantity,
			Cost:         int64(one.Cost),
		})
		if err != nil {
			return nil, writeLayerError(err)
		}

		layer, err := layerFromRow(row)
		if err != nil {
			return nil, err
		}

		written = append(written, layer)
	}

	return written, nil
}

// List returns the layers of a business, in the order they leave the
// stock: the oldest layer is the first.
//
// The error is purchase.ErrNoStorage when the storage is not open.
func (r *StockRepository) List(ctx context.Context, own business.ID) ([]purchase.Layer, error) {
	if r == nil || r.pool == nil {
		return nil, purchase.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListStockLayersByBusiness(ctx, int64(own))
	if err != nil {
		return nil, writeLayerError(err)
	}

	layers := make([]purchase.Layer, 0, len(rows))
	for _, row := range rows {
		layer, err := layerFromRow(row)
		if err != nil {
			return nil, err
		}

		layers = append(layers, layer)
	}

	return layers, nil
}

// layerFromRow maps a row of the layers onto the model of the domain.
//
// The layer is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with
// a layer it does not know.
func layerFromRow(row db.StockLayer) (purchase.Layer, error) {
	layer := purchase.Layer{
		ID:           purchase.LayerID(row.ID),
		BusinessID:   business.ID(row.BusinessID),
		IngredientID: ingredient.ID(row.IngredientID),
		Name:         row.Name,
		PurchaseID:   purchase.ID(row.PurchaseID),
		Position:     int(row.Position),
		Day:          ledger.NewDate(row.Day.Time),
		Bought:       purchase.Measure{Quantity: row.Quantity, Unit: row.Unit},
		Left:         purchase.Measure{Quantity: row.Remaining, Unit: row.Unit},
		Cost:         ledger.Amount(row.Cost),
	}

	return layer.Checked()
}

// positionOf returns the position of the line of a layer as the column of
// the table holds it.
//
// The column is an integer of thirty-two bits, and the position of a layer
// is the number of the line it comes from: a position that does not fit
// the column is a position the table cannot hold.
func positionOf(position int) (int32, error) {
	if position <= 0 || position > math.MaxInt32 {
		return 0, fmt.Errorf("%w: the position of the layer is %d", purchase.ErrInvalidValue, position)
	}

	return int32(position), nil
}

// writeLayerError translates the truth of the database into the errors of
// the port of the layers.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the purchase or the ingredient that is not there is the
// refusal of the foreign key.
func writeLayerError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return purchase.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		return fmt.Errorf("%w: the purchase already has a layer for that line", purchase.ErrInvalidValue)
	case foreignKeyViolation:
		return fmt.Errorf("%w: the purchase or the ingredient of the layer is not there", purchase.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", purchase.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
