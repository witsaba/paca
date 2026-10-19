package server_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp/internal/server"
	"github.com/witsaba/paca/packages/core/adapters/sqlite"
)

// The database is migrated at startup: the file that did not exist carries
// the schema of the initial migration.
func TestTheDatabaseIsMigratedAtStartup(t *testing.T) {
	cfg := testConfig(t)
	srv, err := server.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(cfg.DBPath); err != nil {
		t.Fatalf("the database file was not created: %v", err)
	}

	db, err := sqlite.Open(context.Background(), cfg.DBPath)
	if err != nil {
		t.Fatalf("open the database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("the database was not closed: %v", err)
		}
	}()

	rows, err := db.QueryContext(context.Background(), "PRAGMA user_version")
	if err != nil {
		t.Fatalf("read the schema version: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("the rows were not closed: %v", err)
		}
	}()

	var version int
	if !rows.Next() {
		t.Fatalf("no row: %v", rows.Err())
	}
	if err := rows.Scan(&version); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if version != 1 {
		t.Errorf("schema version = %d, want 1", version)
	}
}

func TestServerNewRefusesAnUnusableDatabase(t *testing.T) {
	cfg := testConfig(t)
	cfg.DBPath = filepath.Join(t.TempDir(), "missing-directory", "paca.db")

	if _, err := server.New(context.Background(), cfg); err == nil {
		t.Fatal("server.New: no error, want the failure of the open")
	}
}

// The server listens, answers and stops when the context is cancelled.

func TestRunServesAndStops(t *testing.T) {
	cfg := testConfig(t)
	cfg.Port = freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx, cfg, logger)
	}()

	waitForServer(t, "http://"+net.JoinHostPort(cfg.Bind, portString(cfg.Port))+server.HealthPath)

	res, err := http.Get("http://" + net.JoinHostPort(cfg.Bind, portString(cfg.Port)) + server.HealthPath)
	if err != nil {
		t.Fatalf("get the health endpoint: %v", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Errorf("the body was not closed: %v", err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not stop")
	}
}

// The server serves TLS when it has the material, which is what the
// deployment on the local network needs.

func TestRunServesTLS(t *testing.T) {
	certFile, keyFile := selfSignedCertificate(t)
	cfg := testConfig(t)
	cfg.Port = freePort(t)
	cfg.TLSCert = certFile
	cfg.TLSKey = keyFile

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx, cfg, logger)
	}()

	address := "https://" + net.JoinHostPort(cfg.Bind, portString(cfg.Port)) + server.HealthPath
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	var res *http.Response
	var err error
	for range 50 {
		res, err = client.Get(address)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("get the health endpoint over TLS: %v", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Errorf("the body was not closed: %v", err)
		}
	}()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not stop")
	}
}

func TestRunRefusesAnUnusableDatabase(t *testing.T) {
	cfg := testConfig(t)
	cfg.DBPath = filepath.Join(t.TempDir(), "missing-directory", "paca.db")

	err := server.Run(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("Run: no error, want the failure of the open")
	}
}

// freePort returns a port that is free at the moment of the call.

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("the listener was not closed: %v", err)
		}
	}()

	return listener.Addr().(*net.TCPAddr).Port
}

// portString is the port as the string an address carries.

func portString(port int) string {
	return strconv.Itoa(port)
}

// waitForServer waits until the server answers at the given URL.

func waitForServer(t *testing.T, url string) {
	t.Helper()

	for range 50 {
		res, err := http.Get(url)
		if err == nil {
			_ = res.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("the server did not answer at %s", url)
}

// selfSignedCertificate writes a self-signed certificate and its key into a
// temporary directory and returns the paths of both.

func selfSignedCertificate(t *testing.T) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate the key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "paca-mcp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create the certificate: %v", err)
	}

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatalf("write the certificate: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal the key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("write the key: %v", err)
	}

	return certFile, keyFile
}
