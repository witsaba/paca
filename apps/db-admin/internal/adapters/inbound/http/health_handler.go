// Package http adapts the use cases of the service to the HTTP protocol.
//
// The adapters translate: they read a request, they call the use case, and
// they write a response. They hold no rule of their own.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/domain/health"
	"github.com/witsaba/paca/apps/db-admin/internal/ports"
)

// HealthPath is the path of the health endpoint. It is the only route of the
// service.
const HealthPath = "/healthz"

// HealthHandler serves the health endpoint of the service.
type HealthHandler struct {
	query ports.HealthQuerier
}

// NewHealthHandler builds the handler of the health endpoint.
func NewHealthHandler(query ports.HealthQuerier) *HealthHandler {
	return &HealthHandler{query: query}
}

// healthResponse is the body of the health endpoint.
type healthResponse struct {
	Status        string          `json:"status"`
	Service       serviceResponse `json:"service"`
	UptimeSeconds float64         `json:"uptime_seconds"`
	CheckedAt     string          `json:"checked_at"`
	Checks        []checkResponse `json:"checks"`
	Error         string          `json:"error,omitempty"`
}

// serviceResponse identifies the service in the health endpoint.
type serviceResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// checkResponse is the outcome of a single check in the health endpoint.
type checkResponse struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Detail    string `json:"detail,omitempty"`
}

// ServeHTTP implements http.Handler.
//
// The endpoint answers 200 while the service can serve, and 503 when it
// cannot. A degraded service still answers 200: it works, and the body says
// that something is wrong.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	report, err := h.query.Health(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Status: health.StatusUnhealthy.String(),
			Error:  err.Error(),
		})
		return
	}

	status := http.StatusOK
	if !report.Status.IsServing() {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, newHealthResponse(report))
}

// newHealthResponse maps a report to the body of the health endpoint.
func newHealthResponse(report health.Report) healthResponse {
	response := healthResponse{
		Status:        report.Status.String(),
		Service:       serviceResponse{Name: report.Identity.Name, Version: report.Identity.Version},
		UptimeSeconds: report.Uptime.Seconds(),
		CheckedAt:     report.CheckedAt.UTC().Format(time.RFC3339Nano),
		Checks:        make([]checkResponse, 0, len(report.Checks)),
	}

	for _, check := range report.Checks {
		response.Checks = append(response.Checks, checkResponse{
			Name:      check.Name,
			Status:    check.Status.String(),
			LatencyMS: check.Latency.Milliseconds(),
			Detail:    check.Detail,
		})
	}

	return response
}

// writeJSON writes a JSON body with the given status.
func writeJSON(w http.ResponseWriter, status int, body healthResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	// The body is written after the status, so an encoding failure can only
	// be logged by the server, never turned into a second status.
	_ = json.NewEncoder(w).Encode(body)
}
