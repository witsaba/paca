package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/witsaba/paca/apps/mcp/internal/auth"
)

// okHandler answers every request with 200 and names the method it received.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.Method))
	})
}

// request sends a request with the given authorization header through the
// middleware and returns the recorded response.
func request(t *testing.T, token, header string) *http.Response {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler := auth.Middleware(token)(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	handler.ServeHTTP(recorder, req)

	return recorder.Result()
}

func TestNoTokenConfiguredPassesEveryRequest(t *testing.T) {
	res := request(t, "", "")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestTheCorrectTokenIsAccepted(t *testing.T) {
	res := request(t, "s3cret", "Bearer s3cret")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestTheSchemeIsCaseInsensitive(t *testing.T) {
	res := request(t, "s3cret", "bearer s3cret")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestSurroundingSpacesAreIgnored(t *testing.T) {
	res := request(t, "s3cret", "Bearer   s3cret  ")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestARefusedRequest(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		header string
	}{
		{name: "no header", token: "s3cret"},
		{name: "wrong token", token: "s3cret", header: "Bearer wrong"},
		{name: "empty credentials", token: "s3cret", header: "Bearer "},
		{name: "another scheme", token: "s3cret", header: "Basic s3cret"},
		{name: "token without a scheme", token: "s3cret", header: "s3cret"},
		{name: "the token in the wrong header", token: "s3cret", header: "Bearer s3cre"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := request(t, tc.token, tc.header)

			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
			}
			if got := res.Header.Get("WWW-Authenticate"); got == "" {
				t.Error("the challenge header is missing")
			}
		})
	}
}

func TestValid(t *testing.T) {
	cases := []struct {
		name   string
		header string
		token  string
		want   bool
	}{
		{name: "the token", header: "Bearer s3cret", token: "s3cret", want: true},
		{name: "no header", header: "", token: "s3cret", want: false},
		{name: "wrong token", header: "Bearer nope", token: "s3cret", want: false},
		{name: "short scheme", header: "Beare", token: "s3cret", want: false},
		{name: "empty credentials", header: "Bearer", token: "s3cret", want: false},
		{name: "an empty token never matches", header: "Bearer ", token: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := auth.Valid(tc.header, tc.token); got != tc.want {
				t.Errorf("Valid(%q, %q) = %v, want %v", tc.header, tc.token, got, tc.want)
			}
		})
	}
}
