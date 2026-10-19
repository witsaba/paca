package server

import "net/http"

// HealthBody is the answer of the liveness endpoint.
const HealthBody = "ok"

// HealthHandler returns the handler of the liveness endpoint.
//
// The endpoint answers 200 with the plain text "ok". It touches no database
// and no MCP session: it says that the process is alive, and nothing about
// the schema or the uptime, which belong to the health tool of a later
// session.
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(HealthBody))
	})
}
