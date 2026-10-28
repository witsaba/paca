// Package application holds the use cases of the database admin service: the
// rules that decide what the service does, with no protocol and no
// infrastructure in sight.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/domain/health"
)

// The health use case is the inbound port of the adapters that expose the
// health of the service.
var _ ports.HealthQuerier = (*HealthService)(nil)

// HealthService is the use case that reports the health of the service.
//
// It owns the rule that matters: one failing component makes the whole
// service unhealthy, one degraded component makes it degraded, and a service
// with no checks at all is healthy.
type HealthService struct {
	identity  health.Identity
	checkers  []ports.Checker
	clock     ports.Clock
	startedAt time.Time
	timeout   time.Duration
}

// NewHealthService builds the health use case.
//
// The identity names the service in the report, the checkers are the
// components it watches, and the timeout bounds how long a single check may
// take.
func NewHealthService(
	identity health.Identity,
	checkers []ports.Checker,
	clock ports.Clock,
	startedAt time.Time,
	timeout time.Duration,
) (*HealthService, error) {
	if identity.Name == "" {
		return nil, errors.New("the health service needs the name of the service")
	}
	if clock == nil {
		return nil, errors.New("the health service needs a clock")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("the health service needs a positive check timeout, got %s", timeout)
	}
	return &HealthService{
		identity:  identity,
		checkers:  checkers,
		clock:     clock,
		startedAt: startedAt,
		timeout:   timeout,
	}, nil
}

// Health returns the health of the service. It implements the inbound port
// ports.HealthQuerier.
//
// A report is returned even when a component fails: the failure is part of
// the report, not an error of the call. The error is reserved for the calls
// that cannot produce a report at all, such as a cancelled context.
func (s *HealthService) Health(ctx context.Context) (health.Report, error) {
	if err := ctx.Err(); err != nil {
		return health.Report{}, err
	}

	now := s.clock.Now()
	report := health.Report{
		Identity:  s.identity,
		Status:    health.StatusOK,
		StartedAt: s.startedAt,
		CheckedAt: now,
		Uptime:    now.Sub(s.startedAt),
		Checks:    make([]health.Check, 0, len(s.checkers)),
	}

	for _, checker := range s.checkers {
		check := s.runCheck(ctx, checker)
		report.Checks = append(report.Checks, check)
		report.Status = report.Status.Worst(check.Status)
	}

	return report, nil
}

// runCheck probes one component and turns its outcome into a check.
func (s *HealthService) runCheck(ctx context.Context, checker ports.Checker) health.Check {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	started := s.clock.Now()
	err := checker.Check(ctx)
	latency := s.clock.Now().Sub(started)

	check := health.Check{
		Name:    checker.Name(),
		Status:  health.StatusOK,
		Latency: latency,
	}

	switch {
	case err == nil:
	case errors.Is(err, health.ErrDegraded):
		check.Status = health.StatusDegraded
		check.Detail = err.Error()
	default:
		check.Status = health.StatusUnhealthy
		check.Detail = err.Error()
	}

	return check
}
