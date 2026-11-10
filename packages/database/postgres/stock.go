package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

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
// The layer is asked before it is written: the domain decides what a layer
// is, and a layer that claims another business than its purchase is
// refused before the storage hears of it.
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
		if _, err := one.Checked(); err != nil {
			return nil, err
		}

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

		layer, err := layerFromWrittenRow(row)
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
// The query reads the purchase of every layer with it: a layer belongs to
// the business of its purchase, and the stock of a business never holds
// the layer of a purchase of another one.
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
		layer, err := layerFromListedRow(row)
		if err != nil {
			return nil, err
		}

		layers = append(layers, layer)
	}

	return layers, nil
}

// WriteRemaining writes the quantity that is left of the layers of the
// stock after the production took from them.
//
// The rule of the domain: the quantity that is left of a layer is the
// quantity it held minus the quantity the production took from it, and the
// quantity and its unit travel together: the remaining carries the unit of
// the layer it belongs to, and the storage never subtracts a gram from a
// kilogram. The layers leave the stock in the order of the FIFO, and the
// storage never takes more of a layer than it holds.
//
// The row is written only when the layer still holds what the production
// read: a layer that changed under the production is refused.
//
// The error is purchase.ErrInvalidValue when the row breaks a rule of the
// table, purchase.ErrNotFound when the layer is not there, and
// purchase.ErrNoStorage when the storage is not open.
func (r *StockRepository) WriteRemaining(ctx context.Context, remaining []purchase.Remaining) error {
	if r == nil || r.pool == nil {
		return purchase.ErrNoStorage
	}

	for _, one := range remaining {
		row, err := r.pool.queriesFor(ctx).TakeStockLayer(ctx, db.TakeStockLayerParams{
			Remaining: one.Left.Quantity,
			ID:        int64(one.LayerID),
			Unit:      one.Left.Unit,
			Held:      one.Held.Quantity,
		})
		if err != nil {
			return writeRemainingError(err)
		}

		if row.Remaining != one.Left.Quantity {
			return fmt.Errorf("%w: the layer %d holds %v and not %v",
				purchase.ErrInvalidValue, one.LayerID, row.Remaining, one.Left.Quantity)
		}
	}

	return nil
}

// rowOfALayer is what a row of the layers carries, however it was read.
type rowOfALayer struct {
	ID                 int64
	BusinessID         int64
	IngredientID       int64
	PurchaseID         int64
	Position           int32
	Name               string
	Day                pgtype.Date
	Quantity           float64
	Unit               string
	Remaining          float64
	Cost               int64
	PurchaseBusinessID int64
}

// layerFromWrittenRow maps the row of a layer that was just written onto
// the model of the domain.
func layerFromWrittenRow(row db.CreateStockLayerRow) (purchase.Layer, error) {
	return layerOf(rowOfALayer{
		ID:                 row.ID,
		BusinessID:         row.BusinessID,
		IngredientID:       row.IngredientID,
		PurchaseID:         row.PurchaseID,
		Position:           row.Position,
		Name:               row.Name,
		Day:                row.Day,
		Quantity:           row.Quantity,
		Unit:               row.Unit,
		Remaining:          row.Remaining,
		Cost:               row.Cost,
		PurchaseBusinessID: row.PurchaseBusinessID,
	})
}

// layerFromListedRow maps a row of the layers that was read back onto the
// model of the domain.
//
// The line of the purchase is read with the layer, and the domain asks
// whether the layer is the layer of that line: the money of the layer is
// the money of the line of the purchase, and a row that another writer
// broke is refused.
func layerFromListedRow(row db.ListStockLayersByBusinessRow) (purchase.Layer, error) {
	layer, err := layerOf(rowOfALayer{
		ID:                 row.ID,
		BusinessID:         row.BusinessID,
		IngredientID:       row.IngredientID,
		PurchaseID:         row.PurchaseID,
		Position:           row.Position,
		Name:               row.Name,
		Day:                row.Day,
		Quantity:           row.Quantity,
		Unit:               row.Unit,
		Remaining:          row.Remaining,
		Cost:               row.Cost,
		PurchaseBusinessID: row.PurchaseBusinessID,
	})
	if err != nil {
		return purchase.Layer{}, err
	}

	line, err := lineOfThePurchase(row)
	if err != nil {
		return purchase.Layer{}, err
	}

	bought := purchase.Purchase{
		ID:         purchase.ID(row.PurchaseID),
		BusinessID: business.ID(row.PurchaseBusinessID),
		Day:        ledger.NewDate(row.Day.Time),
		Lines:      []purchase.Line{line},
	}

	if err := layer.CheckedAgainstTheLine(bought, line, int(row.Position)); err != nil {
		return purchase.Layer{}, err
	}

	return layer, nil
}

// lineOfThePurchase builds the line of the purchase the layer comes from.
//
// The line is read with the layer: the money of the layer is the money of
// the line, and a layer that comes from no line of its purchase is
// refused instead of being read as a layer of the stock.
func lineOfThePurchase(row db.ListStockLayersByBusinessRow) (purchase.Line, error) {
	if !row.LineIngredientID.Valid || !row.LineName.Valid || !row.LineQuantity.Valid ||
		!row.LineUnit.Valid || !row.LineTotal.Valid || !row.LineUnitPrice.Valid {
		return purchase.Line{}, fmt.Errorf("%w: the layer of the purchase %d comes from no line of it",
			purchase.ErrLayerNotTied, row.PurchaseID)
	}

	line, err := purchase.NewLine(
		row.LineName.String,
		purchase.Measure{Quantity: row.LineQuantity.Float64, Unit: row.LineUnit.String},
		purchase.Price{
			Total: ledger.Amount(row.LineTotal.Int64),
			Unit:  ledger.Amount(row.LineUnitPrice.Int64),
		},
	)
	if err != nil {
		return purchase.Line{}, err
	}

	return line.WithIngredient(ingredient.ID(row.LineIngredientID.Int64)), nil
}

// layerOf builds the layer of the domain out of a row, and asks the domain
// whether it is whole.
//
// The layer is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with
// a layer it does not know.
func layerOf(row rowOfALayer) (purchase.Layer, error) {
	layer := purchase.Layer{
		ID:                 purchase.LayerID(row.ID),
		BusinessID:         business.ID(row.BusinessID),
		PurchaseBusinessID: business.ID(row.PurchaseBusinessID),
		IngredientID:       ingredient.ID(row.IngredientID),
		Name:               row.Name,
		PurchaseID:         purchase.ID(row.PurchaseID),
		Position:           int(row.Position),
		Day:                ledger.NewDate(row.Day.Time),
		Bought:             purchase.Measure{Quantity: row.Quantity, Unit: row.Unit},
		Left:               purchase.Measure{Quantity: row.Remaining, Unit: row.Unit},
		Cost:               ledger.Amount(row.Cost),
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

// writeRemainingError translates the truth of the database into the errors
// of the port of the layers.
//
// The layer that is not there is a layer the storage does not hold, and a
// layer that changed under the production is a row the database refused.
func writeRemainingError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return purchase.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code == checkViolation {
		return fmt.Errorf("%w: the constraint %q refused the row",
			purchase.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
