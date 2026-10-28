package server_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/platform/server"
)

// discardLogger is the logger of the test.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTheServerServesTheRequestsAndShutsDown(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("served"))
	})

	srv := server.New("127.0.0.1:0", handler, discardLogger(), 5*time.Second)
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen() = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() {
		served <- srv.Serve(ctx)
	}()

	response, err := http.Get("http://" + srv.Addr() + "/")
	if err != nil {
		t.Fatalf("the request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	cancel()

	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("Serve() = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not shut down")
	}
}

func TestTheServerFailsToListenOnABusyPort(t *testing.T) {
	t.Parallel()

	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("the test could not bind a port: %v", err)
	}
	defer busy.Close()

	srv := server.New(busy.Addr().String(), http.NewServeMux(), discardLogger(), time.Second)
	if err := srv.Listen(); err == nil {
		t.Fatal("Listen() = nil, want an error")
	}
}
