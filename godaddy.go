package nsdyndns

// cspell:ignore nsdyndns godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type GoDaddyService struct {
	Key    string
	Secret string
	apiURL string
}

type HostResponse struct {
	Data string `json:"data"`
	Name string `json:"name"`
}

const goDaddyDefaultAPIURL = "https://api.godaddy.com/v1"

func NewGoDaddyService(key, secret string) NameService {
	return &GoDaddyService{Key: key, Secret: secret, apiURL: goDaddyDefaultAPIURL}
}

func (s *GoDaddyService) SetAddress(ctx context.Context, domain, host string, address net.IP) error {
	body, err := json.Marshal([]HostResponse{{Data: address.String()}})

	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/domains/%s/records/A/%s", s.apiURL, domain, recordName(host))

	_, err = doRequest(ctx, func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))

		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", goDaddyAuthorization(s.Key, s.Secret))
		req.Header.Set("Content-Type", "application/json")

		return req, nil
	})

	if err != nil {
		return fmt.Errorf("couldn't update ip address: %w", err)
	}

	return nil
}

func goDaddyAuthorization(key, secret string) string {
	return fmt.Sprintf("sso-key %s:%s", key, secret)
}
