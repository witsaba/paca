package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	adapter "github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
)

// fakeQuerier is the use case of the test: it returns the report the test
// decided.
type fakeQuerier struct {
	report health.Report
	err    error
}

// Health returns the report of the test.
func (q *fakeQuerier) Health(context.Context) (health.Report, error) {
	return q.report, q.err
}

func TestTheHealthEndpointAnswersTheReport(t *testing.T) {
	t.Parallel()

	checkedAt := time.Date(2026, time.October, 27, 10, 0, 0, 0, time.UTC)
	querier := &fakeQuerier{report: health.Report{
		Identity:  health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
		Status:    health.StatusOK,
		StartedAt: checkedAt.Add(-time.Minute),
		CheckedAt: checkedAt,
		Uptime:    time.Minute,
		Checks: []health.Check{{
			Name:    "mcp-transport",
			Status:  health.StatusOK,
			Latency: 1500 * time.Microsecond,
		}},
	}}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil)

	adapter.NewHealthHandler(querier).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q, want a JSON body", contentType)
	}

	var body struct {
		Status  string `json:"status"`
		Service struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"service"`
		UptimeSeconds float64 `json:"uptime_seconds"`
		CheckedAt     string  `json:"checked_at"`
		Checks        []struct {
			Name      string `json:"name"`
			Status    string `json:"status"`
			LatencyMS int64  `json:"latency_ms"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("status = %q, want %q", body.Status, "ok")
	}
	if body.Service.Name != "paca-mcp-server" || body.Service.Version != "0.1.0" {
		t.Fatalf("service = %+v, want the name and the version of the server", body.Service)
	}
	if body.UptimeSeconds != 60 {
		t.Fatalf("uptime = %v, want 60", body.UptimeSeconds)
	}
	if body.CheckedAt != "2026-10-27T10:00:00Z" {
		t.Fatalf("checked at = %q, want %q", body.CheckedAt, "2026-10-27T10:00:00Z")
	}
	if len(body.Checks) != 1 {
		t.Fatalf("checks = %d, want 1", len(body.Checks))
	}
	if body.Checks[0].Name != "mcp-transport" || body.Checks[0].Status != "ok" {
		t.Fatalf("check = %+v, want the transport to be ok", body.Checks[0])
	}
}

func TestTheHealthEndpointAnswersUnavailableWhenTheServiceIsUnhealthy(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{report: health.Report{
		Identity: health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
		Status:   health.StatusUnhealthy,
		Checks: []health.Check{{
			Name:   "mcp-transport",
			Status: health.StatusUnhealthy,
			Detail: "the transport is not serving yet",
		}},
	}}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil)

	adapter.NewHealthHandler(querier).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), "unhealthy") {
		t.Fatalf("body = %q, want it to report an unhealthy service", recorder.Body.String())
	}
}

func TestTheHealthEndpointAnswersUnavailableWhenTheReportFails(t *testing.T) {
	t.Parallel()

	querier := &fakeQuerier{err: errors.New("the request was cancelled")}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil)

	adapter.NewHealthHandler(querier).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), "the request was cancelled") {
		t.Fatalf("body = %q, want it to explain the failure", recorder.Body.String())
	}
}
