package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/domain/schema"
)

// MigrateService is the use case that applies the pending migrations of the
// schema.
//
// It owns the rule of the idempotence: applying the migrations twice is not
// an error, the second run finds nothing to do and says so.
type MigrateService struct {
	migrator ports.Migrator
}

// NewMigrateService builds the migration use case.
func NewMigrateService(migrator ports.Migrator) (*MigrateService, error) {
	if migrator == nil {
		return nil, errors.New("the migration service needs a migrator")
	}
	return &MigrateService{migrator: migrator}, nil
}

// Migrate applies the pending migrations and returns the state of the schema.
//
// The state carries the migrations that were applied by this run: an empty
// list means the schema was already up to date, which is not an error.
func (s *MigrateService) Migrate(ctx context.Context) (schema.State, error) {
	applied, err := s.migrator.Up(ctx)
	if err != nil {
		return schema.State{}, fmt.Errorf("the pending migrations could not be applied: %w", err)
	}

	version, err := s.migrator.Version(ctx)
	if err != nil {
		return schema.State{}, fmt.Errorf("the version of the schema could not be read: %w", err)
	}

	pending, err := s.migrator.Pending(ctx)
	if err != nil {
		return schema.State{}, fmt.Errorf("the pending migrations could not be read: %w", err)
	}

	return schema.State{
		Version: version,
		Pending: pending,
		Applied: applied,
	}, nil
}

// Status returns the state of every migration of the schema, and changes
// nothing.
func (s *MigrateService) Status(ctx context.Context) ([]schema.Migration, error) {
	migrations, err := s.migrator.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("the state of the migrations could not be read: %w", err)
	}
	return migrations, nil
}
