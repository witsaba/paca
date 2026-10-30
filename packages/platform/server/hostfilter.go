package server

import (
	"net/http"
	"strings"
)

// The Host header of a request names the host the client believes it is
// talking to.
//
// A server that answers a foreign host answers a request that was meant for
// another service: the browser of a person, or a proxy that forwards the
// request to the wrong place. The services of the monorepository that are
// reachable from outside name the hosts they answer for.

// HostFilter refuses the requests that carry a Host header the service does
// not answer for.
//
// The names of the allowlist are the hosts of the service: the port of the
// request is not part of the name of the host. The comparison ignores the
// case, because the name of a host does not carry one.
//
// A request with no Host header is refused: a request that does not say who
// it is talking to is a request that may be talking to anybody.
func HostFilter(allowed []string) func(http.Handler) http.Handler {
	names := make([]string, 0, len(allowed))
	for _, name := range allowed {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, strings.ToLower(name))
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !hostIsAllowed(r.Host, names) {
				http.Error(w,
					"the host of the request is not one of the hosts of the service",
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hostIsAllowed tells whether the Host header of the request names one of
// the hosts of the service.
func hostIsAllowed(host string, allowed []string) bool {
	if host == "" {
		return false
	}

	name := strings.ToLower(host)
	if index := strings.LastIndex(name, ":"); index != -1 && !strings.Contains(name[index:], "]") {
		name = name[:index]
	}

	for _, candidate := range allowed {
		if candidate == name {
			return true
		}
	}

	return false
}
