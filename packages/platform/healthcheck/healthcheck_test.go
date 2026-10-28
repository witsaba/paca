package healthcheck_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/witsaba/paca/packages/platform/healthcheck"
)

func TestTheCheckFailsWhenTheServerDoesNotAnswer(t *testing.T) {
	t.Parallel()

	if err := healthcheck.Check(context.Background(), 1); err == nil {
		t.Fatal("Check() = nil, want an error")
	}
}

func TestTheCheckFailsWhenTheServerIsNotHealthy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	if err := healthcheck.Check(context.Background(), portOf(t, server.URL)); err == nil {
		t.Fatal("Check() = nil, want an error")
	}
}

// portOf returns the port a test server listens on.
func portOf(t *testing.T, url string) int {
	t.Helper()

	_, port, err := net.SplitHostPort(url[len("http://"):])
	if err != nil {
		t.Fatalf("read the port of %q: %v", url, err)
	}

	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("read the port of %q: %v", url, err)
	}

	return number
}
