// Package auth protects the MCP surface with a bearer token.
//
// The token is checked on every request that reaches the MCP surface, and
// the comparison is constant time, so the answer of the server does not leak
// how much of the token the caller guessed. The token itself never leaves
// this package: it is compared, never logged.
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// bearerPrefix is the scheme of the authorization header, followed by a
// space. The scheme is compared without case, as RFC 9110 allows.
const bearerPrefix = "Bearer "

// Unauthorized is the message the server answers a refused request with. It
// names no reason: a caller must not learn whether the token was missing or
// wrong.
const Unauthorized = "unauthorized"

// Middleware returns a handler that demands the bearer token.
//
// An empty token means that no authentication is configured, and the handler
// is the identity: the caller has already decided that the surface is open,
// which only a loopback bind allows. A token that is configured is checked on
// every request, and a request that does not carry it is refused with 401.
func Middleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if token == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !Valid(r.Header.Get("Authorization"), token) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="paca-mcp"`)
				http.Error(w, Unauthorized, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Valid reports whether the authorization header carries the token.
//
// The header must use the bearer scheme, and the credentials that follow it
// must be exactly the token. The comparison is constant time.
func Valid(header, token string) bool {
	if len(header) < len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return false
	}
	presented := strings.TrimSpace(header[len(bearerPrefix):])
	if presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(token)) == 1
}
