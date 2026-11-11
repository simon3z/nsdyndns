package main

// cspell:ignore nsdyndns

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simon3z/nsdyndns/cmd"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()

	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config: %s", err)
	}

	return path
}

type stubService struct {
	calls   int
	lastIP  net.IP
	failErr error
}

func (s *stubService) SetAddress(ctx context.Context, domain, host string, address net.IP) error {
	s.calls++
	s.lastIP = address

	return s.failErr
}

func configForTest(t *testing.T) *cmd.Config {
	t.Helper()

	path := writeConfig(t, `
domain: example.com
host: sub
digitalocean:
  token: tok
`)

	config, err := cmd.LoadConfiguration(path)

	if err != nil {
		t.Fatalf("failed to load config: %s", err)
	}

	return config
}

// Regression: the flags registered in init survive the binary build.
func TestFlagsRegistered(t *testing.T) {
	for _, name := range []string{"c", "i", "version"} {
		if flag.Lookup(name) == nil {
			t.Fatalf("flag %s is not registered", name)
		}
	}
}

func TestCheckAndUpdateNoChange(t *testing.T) {
	// DNS already points at the external ip: no provider is called.
	config := configForTest(t)

	stub := &stubService{}
	services := []*cmd.ConfigNameService{{Name: "digitalocean", Service: stub}}

	err := CheckAndUpdate(context.Background(), config, services,
		func(ctx context.Context) (net.IP, error) { return net.ParseIP("203.0.113.42"), nil },
		func(host string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.42")}, nil },
	)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if stub.calls != 0 {
		t.Fatalf("expected no provider calls, got %d", stub.calls)
	}
}

func TestCheckAndUpdateUpdatesAllServices(t *testing.T) {
	config := configForTest(t)

	stubA := &stubService{}
	stubB := &stubService{}
	services := []*cmd.ConfigNameService{{Name: "a", Service: stubA}, {Name: "b", Service: stubB}}

	newIP := net.ParseIP("203.0.113.42")

	err := CheckAndUpdate(context.Background(), config, services,
		func(ctx context.Context) (net.IP, error) { return newIP, nil },
		func(host string) ([]net.IP, error) { return []net.IP{net.ParseIP("198.51.100.7")}, nil },
	)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if stubA.calls != 1 || stubB.calls != 1 {
		t.Fatalf("expected one call per service, got %d and %d", stubA.calls, stubB.calls)
	}

	if !stubA.lastIP.Equal(newIP) || !stubB.lastIP.Equal(newIP) {
		t.Fatalf("providers not updated to %s", newIP)
	}
}

// Regression: one failing service does not stop the others, and both
// failures are aggregated into the returned error.
func TestCheckAndUpdateAggregatesErrors(t *testing.T) {
	config := configForTest(t)

	failA := &stubService{failErr: errors.New("a failed")}
	stubB := &stubService{}
	services := []*cmd.ConfigNameService{{Name: "a", Service: failA}, {Name: "b", Service: stubB}}

	err := CheckAndUpdate(context.Background(), config, services,
		func(ctx context.Context) (net.IP, error) { return net.ParseIP("203.0.113.42"), nil },
		func(host string) ([]net.IP, error) { return []net.IP{net.ParseIP("198.51.100.7")}, nil },
	)

	if err == nil {
		t.Fatal("expected error for failing service")
	}

	if !strings.Contains(err.Error(), "a failed") || !strings.Contains(err.Error(), "a: ") {
		t.Fatalf("expected aggregated error naming service a: %s", err)
	}

	if stubB.calls != 1 {
		t.Fatalf("expected service b to still be updated, got %d calls", stubB.calls)
	}
}

func TestCheckAndUpdateIPFetcherError(t *testing.T) {
	config := configForTest(t)

	stub := &stubService{}
	services := []*cmd.ConfigNameService{{Name: "digitalocean", Service: stub}}

	err := CheckAndUpdate(context.Background(), config, services,
		func(ctx context.Context) (net.IP, error) { return nil, errors.New("fetch failed") },
		func(host string) ([]net.IP, error) { return []net.IP{net.ParseIP("198.51.100.7")}, nil },
	)

	if err == nil || err.Error() != "fetch failed" {
		t.Fatalf("expected ip fetcher error to propagate, got: %v", err)
	}

	if stub.calls != 0 {
		t.Fatalf("expected no provider calls, got %d", stub.calls)
	}
}

func TestCheckAndUpdateDnsLookupError(t *testing.T) {
	config := configForTest(t)

	stub := &stubService{}
	services := []*cmd.ConfigNameService{{Name: "digitalocean", Service: stub}}

	err := CheckAndUpdate(context.Background(), config, services,
		func(ctx context.Context) (net.IP, error) { return net.ParseIP("203.0.113.42"), nil },
		func(host string) ([]net.IP, error) {
			return nil, fmt.Errorf("lookup failed: %w", errors.New("dns error"))
		},
	)

	if err == nil || !strings.Contains(err.Error(), "address lookup for sub.example.com failed") {
		t.Fatalf("expected wrapped dns lookup error, got: %v", err)
	}

	if stub.calls != 0 {
		t.Fatalf("expected no provider calls, got %d", stub.calls)
	}
}
