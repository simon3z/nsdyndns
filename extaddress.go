package nsdyndns

// cspell:ignore nsdyndns ipify

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// externalIPServiceURLs are the external IP detection services, in order of
// preference. Each is tried until one returns a valid IP address.
var externalIPServiceURLs = []url.URL{
	{Scheme: "https", Host: "api.ipify.org"},
	{Scheme: "https", Host: "api.ip.sb", Path: "/ip"},
	{Scheme: "https", Host: "ifconfig.me", Path: "/ip"},
}

// GetExternalIP determines the host's external IP address by querying the
// configured IP detection services in order until one succeeds.
func GetExternalIP(ctx context.Context) (net.IP, error) {
	var lastErr error

	for _, u := range externalIPServiceURLs {
		body, err := doRequest(ctx, func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		})

		if err != nil {
			lastErr = err
			continue
		}

		ip := net.ParseIP(strings.TrimSpace(string(body)))

		if ip == nil {
			lastErr = fmt.Errorf("couldn't parse ip address from %s: %s", u.Host, strings.TrimSpace(string(body)))
			continue
		}

		return ip, nil
	}

	return nil, lastErr
}

// GetExternalIPService returns the host of the preferred external IP
// detection service, for logging purposes.
func GetExternalIPService() string {
	return externalIPServiceURLs[0].Host
}
