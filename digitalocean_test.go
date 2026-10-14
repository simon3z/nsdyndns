package nsdyndns

// cspell:ignore nsdyndns digitalocean

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// doAPI mocks the DigitalOcean v2 API, serving a record list (with
// pagination), accepting PUT updates and POST creations.
type doAPI struct {
	t       *testing.T
	records []DomainRecord
	putBody []byte
	created []DomainRecord
}

func (m *doAPI) handler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/v2/domains/example.com/records" && r.Method == http.MethodGet:
		page, _ := parsePageParam(m.t, r.URL.Query().Get("page"))
		start := page - 1
		end := start + 1
		if end > len(m.records) {
			end = len(m.records)
		}

		body := struct {
			DomainRecords []DomainRecord `json:"domain_records"`
			Links         struct {
				Next string `json:"next"`
			} `json:"links"`
			Total int `json:"total"`
		}{
			DomainRecords: m.records[start:end],
			Total:         len(m.records),
		}

		if end < len(m.records) {
			body.Links.Next = fmt.Sprintf("/v2/domains/example.com/records?page=%d", page+1)
		}

		writeJSON(m.t, w, http.StatusOK, body)

	case r.URL.Path == "/v2/domains/example.com/records" && r.Method == http.MethodPost:
		var rec DomainRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			writeJSON(m.t, w, http.StatusBadRequest, map[string]string{"message": "invalid body"})
			return
		}

		rec.Id = int64(len(m.created) + 1000)
		m.created = append(m.created, rec)

		writeJSON(m.t, w, http.StatusCreated, struct {
			DomainRecord DomainRecord `json:"domain_record"`
		}{DomainRecord: rec})

	case strings.HasPrefix(r.URL.Path, "/v2/domains/example.com/records/") && r.Method == http.MethodPut:
		var rec DomainRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			writeJSON(m.t, w, http.StatusBadRequest, map[string]string{"message": "invalid body"})
			return
		}

		m.putBody, _ = json.Marshal(rec)

		writeJSON(m.t, w, http.StatusOK, struct {
			DomainRecord DomainRecord `json:"domain_record"`
		}{DomainRecord: rec})

	default:
		writeJSON(m.t, w, http.StatusNotFound, map[string]string{"message": "not found"})
	}
}

func parsePageParam(t *testing.T, page string) (int, error) {
	if page == "" {
		return 1, nil
	}

	var p int

	_, err := fmt.Sscanf(page, "%d", &p)

	return p, err
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body interface{}) {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		t.Fatalf("failed to encode response: %s", err)
	}
}

func TestDigitalOceanSetAddressUpdate(t *testing.T) {
	m := &doAPI{t: t, records: []DomainRecord{{Id: 42, Type: "A", Name: "sub", Data: "198.51.100.1", Ttl: 60}}}
	srv := httptest.NewServer(http.HandlerFunc(m.handler))
	defer srv.Close()

	s := NewDigitalOceanService("token").(*DigitalOceanService)
	s.apiURL = srv.URL + "/v2"

	addr := net.ParseIP("203.0.113.42")

	if err := s.SetAddress(context.Background(), "example.com", "sub", addr); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var putRec DomainRecord

	if err := json.Unmarshal(m.putBody, &putRec); err != nil {
		t.Fatalf("failed to unmarshal put body: %s", err)
	}

	if putRec.Data != "203.0.113.42" || putRec.Type != "A" || putRec.Name != "sub" || putRec.Ttl != DigitalOceanDefaultTtl {
		t.Fatalf("unexpected put body: %#v", putRec)
	}
}

func TestDigitalOceanSetAddressTypeFilter(t *testing.T) {
	// Only a CNAME record with the matching name exists: an A record must
	// be created instead of updating the CNAME.
	m := &doAPI{t: t, records: []DomainRecord{{Id: 7, Type: "CNAME", Name: "sub", Data: "elsewhere.example.com", Ttl: 60}}}
	srv := httptest.NewServer(http.HandlerFunc(m.handler))
	defer srv.Close()

	s := NewDigitalOceanService("token").(*DigitalOceanService)
	s.apiURL = srv.URL + "/v2"

	addr := net.ParseIP("203.0.113.42")

	if err := s.SetAddress(context.Background(), "example.com", "sub", addr); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(m.created) != 1 || m.created[0].Type != "A" {
		t.Fatalf("expected A record creation, got %#v", m.created)
	}
}

func TestDigitalOceanSetAddressCreate(t *testing.T) {
	m := &doAPI{t: t}
	srv := httptest.NewServer(http.HandlerFunc(m.handler))
	defer srv.Close()

	s := NewDigitalOceanService("token").(*DigitalOceanService)
	s.apiURL = srv.URL + "/v2"

	addr := net.ParseIP("203.0.113.42")

	if err := s.SetAddress(context.Background(), "example.com", "sub", addr); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(m.created) != 1 || m.created[0].Data != "203.0.113.42" {
		t.Fatalf("expected record creation, got %#v", m.created)
	}
}

func TestDigitalOceanSetAddressPagination(t *testing.T) {
	m := &doAPI{t: t, records: []DomainRecord{
		{Id: 1, Type: "A", Name: "other", Data: "198.51.100.1", Ttl: 60},
		{Id: 2, Type: "A", Name: "sub", Data: "198.51.100.2", Ttl: 60},
	}}
	srv := httptest.NewServer(http.HandlerFunc(m.handler))
	defer srv.Close()

	s := NewDigitalOceanService("token").(*DigitalOceanService)
	s.apiURL = srv.URL + "/v2"

	addr := net.ParseIP("203.0.113.42")

	if err := s.SetAddress(context.Background(), "example.com", "sub", addr); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var putRec DomainRecord

	if err := json.Unmarshal(m.putBody, &putRec); err != nil {
		t.Fatalf("failed to unmarshal put body: %s", err)
	}

	if putRec.Data != "203.0.113.42" {
		t.Fatalf("unexpected put body: %#v", putRec)
	}
}

func TestDigitalOceanSetAddressVerificationFailure(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v2/domains/example.com/records", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, struct {
			DomainRecords []DomainRecord `json:"domain_records"`
			Total         int            `json:"total"`
		}{DomainRecords: []DomainRecord{{Id: 42, Type: "A", Name: "sub", Data: "198.51.100.1", Ttl: 60}}, Total: 1})
	})

	mux.HandleFunc("/v2/domains/example.com/records/42", func(w http.ResponseWriter, r *http.Request) {
		// Server echoes back a record that does not match the requested update.
		writeJSON(t, w, http.StatusOK, struct {
			DomainRecord DomainRecord `json:"domain_record"`
		}{DomainRecord: DomainRecord{Id: 42, Type: "A", Name: "sub", Data: "198.51.100.1", Ttl: 60}})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := NewDigitalOceanService("token").(*DigitalOceanService)
	s.apiURL = srv.URL + "/v2"

	addr := net.ParseIP("203.0.113.42")

	if err := s.SetAddress(context.Background(), "example.com", "sub", addr); err == nil {
		t.Fatal("expected verification failure error")
	}
}

func TestDoRequestRetriesOn429(t *testing.T) {
	attempts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	defer srv.Close()

	body, err := doRequest(context.Background(), func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	})

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if string(body) != "ok" {
		t.Fatalf("unexpected body: %s", body)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDoRequestNoRetryOn4xx(t *testing.T) {
	attempts := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusForbidden)
	}))

	defer srv.Close()

	_, err := doRequest(context.Background(), func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	})

	if err == nil {
		t.Fatal("expected error for 403 response")
	}

	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
