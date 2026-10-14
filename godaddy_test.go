package nsdyndns

// cspell:ignore nsdyndns godaddy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoDaddySetAddress(t *testing.T) {
	var gotAuth string
	var gotBody []HostResponse

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")

		err := json.NewDecoder(r.Body).Decode(&gotBody)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	s := NewGoDaddyService("key", "secret").(*GoDaddyService)
	s.apiURL = srv.URL

	addr := net.ParseIP("203.0.113.42")

	err := s.SetAddress(context.Background(), "example.com", "sub", addr)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if gotAuth != "sso-key key:secret" {
		t.Fatalf("unexpected authorization header: %s", gotAuth)
	}

	if len(gotBody) != 1 || gotBody[0].Data != "203.0.113.42" {
		t.Fatalf("unexpected request body: %#v", gotBody)
	}
}

func TestGoDaddySetAddressApex(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	s := NewGoDaddyService("key", "secret").(*GoDaddyService)
	s.apiURL = srv.URL

	addr := net.ParseIP("203.0.113.42")

	err := s.SetAddress(context.Background(), "example.com", "", addr)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if gotPath != "/domains/example.com/records/A/@" {
		t.Fatalf("unexpected path %s", gotPath)
	}
}

func TestGoDaddySetAddressError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"record-not-found"}`))
	}))

	defer srv.Close()

	s := NewGoDaddyService("key", "secret").(*GoDaddyService)
	s.apiURL = srv.URL

	addr := net.ParseIP("203.0.113.42")

	if err := s.SetAddress(context.Background(), "example.com", "sub", addr); err == nil {
		t.Fatal("expected error for 404 response")
	}
}
