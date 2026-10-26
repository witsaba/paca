package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adapter "github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
)

// BenchmarkTheHealthEndpoint measures the cost of a health answer.
func BenchmarkTheHealthEndpoint(b *testing.B) {
	querier := &fakeQuerier{report: health.Report{
		Identity:  health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
		Status:    health.StatusOK,
		StartedAt: time.Now(),
		CheckedAt: time.Now(),
		Uptime:    time.Minute,
		Checks: []health.Check{{
			Name:   "mcp-transport",
			Status: health.StatusOK,
		}},
	}}

	handler := adapter.NewHealthHandler(querier)

	b.ReportAllocs()
	for b.Loop() {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, adapter.HealthPath, nil)
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			b.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
	}
}
