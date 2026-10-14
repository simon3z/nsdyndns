package nsdyndns

import (
	"context"
	"net"
)

type NameService interface {
	SetAddress(ctx context.Context, domain, host string, address net.IP) error
}

// recordName maps an empty host to "@", the conventional name for the
// apex (root) domain record.
func recordName(host string) string {
	if host == "" {
		return "@"
	}

	return host
}
