package nsdyndns

// cspell:ignore nsdyndns ipify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetExternalIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("203.0.113.42\n"))
	}))

	defer srv.Close()

	origURLs := externalIPServiceURLs

	externalIPServiceURLs = []url.URL{{Scheme: "http", Host: srv.Listener.Addr().String()}}

	defer func() { externalIPServiceURLs = origURLs }()

	ip, err := GetExternalIP(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if ip.String() != "203.0.113.42" {
		t.Fatalf("expected ip 203.0.113.42, got %s", ip)
	}
}

func TestGetExternalIPFallback(t *testing.T) {
	// First service fails, second service succeeds.
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("198.51.100.7"))
	}))

	defer srv1.Close()
	defer srv2.Close()

	origURLs := externalIPServiceURLs

	externalIPServiceURLs = []url.URL{
		{Scheme: "http", Host: srv1.Listener.Addr().String()},
		{Scheme: "http", Host: srv2.Listener.Addr().String()},
	}

	defer func() { externalIPServiceURLs = origURLs }()

	ip, err := GetExternalIP(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if ip.String() != "198.51.100.7" {
		t.Fatalf("expected ip 198.51.100.7, got %s", ip)
	}
}

func TestGetExternalIPInvalidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-an-ip"))
	}))

	defer srv.Close()

	origURLs := externalIPServiceURLs

	externalIPServiceURLs = []url.URL{{Scheme: "http", Host: srv.Listener.Addr().String()}}

	defer func() { externalIPServiceURLs = origURLs }()

	_, err := GetExternalIP(context.Background())

	if err == nil {
		t.Fatal("expected error for invalid response body")
	}
}
