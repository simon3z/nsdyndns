package nsdyndns

// cspell:ignore nsdyndns digitalocean

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
)

type DigitalOceanService struct {
	Token  string
	apiURL string
}

type DomainRecord struct {
	Id   int64  `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	Ttl  int    `json:"ttl"`
}

const (
	digitalOceanDefaultAPIURL = "https://api.digitalocean.com/v2"
	DigitalOceanDefaultTtl    = 60
)

// ErrRecordNotFound indicates that no matching domain record exists.
var ErrRecordNotFound = errors.New("domain record not found")

func NewDigitalOceanService(token string) NameService {
	return &DigitalOceanService{Token: token, apiURL: digitalOceanDefaultAPIURL}
}

func (s *DigitalOceanService) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	return doRequest(ctx, func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, method, s.apiURL+"/"+path, bytes.NewReader(body))

		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+s.Token)
		req.Header.Set("Content-Type", "application/json")

		return req, nil
	})
}

// findRecordId looks up the id of the A record for host, following
// pagination, and returns ErrRecordNotFound when there is no match.
func (s *DigitalOceanService) findRecordId(ctx context.Context, domain, host string) (int64, error) {
	page := 1

	for {
		body, err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("domains/%s/records?per_page=200&page=%d", domain, page), nil)

		if err != nil {
			return 0, fmt.Errorf("couldn't list domain records: %w", err)
		}

		resp := struct {
			DomainRecords []DomainRecord `json:"domain_records"`
			Links         struct {
				Next string `json:"next"`
			} `json:"links"`
		}{}

		if err := json.Unmarshal(body, &resp); err != nil {
			return 0, err
		}

		for i := range resp.DomainRecords {
			record := resp.DomainRecords[i]

			if record.Name == recordName(host) && record.Type == "A" {
				return record.Id, nil
			}
		}

		if resp.Links.Next == "" {
			return 0, ErrRecordNotFound
		}

		page++
	}
}

func (s *DigitalOceanService) SetAddress(ctx context.Context, domain, host string, address net.IP) error {
	record := DomainRecord{
		Type: "A",
		Name: recordName(host),
		Data: address.String(),
		Ttl:  DigitalOceanDefaultTtl,
	}

	body, err := json.Marshal(record)

	if err != nil {
		return err
	}

	id, err := s.findRecordId(ctx, domain, host)

	if errors.Is(err, ErrRecordNotFound) {
		_, err := s.doRequest(ctx, http.MethodPost, fmt.Sprintf("domains/%s/records", domain), body)

		if err != nil {
			return fmt.Errorf("couldn't create ip address: %w", err)
		}

		return nil
	}

	if err != nil {
		return err
	}

	resBody, err := s.doRequest(ctx, http.MethodPut, fmt.Sprintf("domains/%s/records/%d", domain, id), body)

	if err != nil {
		return fmt.Errorf("couldn't update ip address: %w", err)
	}

	resAddress := struct {
		DomainRecord DomainRecord `json:"domain_record"`
	}{}

	if err := json.Unmarshal(resBody, &resAddress); err != nil {
		return err
	}

	if resAddress.DomainRecord.Data != address.String() || resAddress.DomainRecord.Ttl != DigitalOceanDefaultTtl {
		return fmt.Errorf("possible failure in domain update: %#v", resAddress.DomainRecord)
	}

	return nil
}
