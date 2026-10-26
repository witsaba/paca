// Package healthcheck asks the server whether it is alive.
//
// The container health check runs this command inside the container, so it
// always dials the loopback interface of the container: the server is not
// published to anything, and the check needs no network to reach it.
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Timeout bounds the wait of the health check.
const Timeout = 3 * time.Second

// Check asks the server on the given port whether it is alive.
//
// It returns nil when the server answers that it is healthy, and an error
// when it does not answer, or when it answers that it is not healthy.
func Check(ctx context.Context, port int) error {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build the health request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("the server did not answer: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the server answered %d, want %d", response.StatusCode, http.StatusOK)
	}

	return nil
}
