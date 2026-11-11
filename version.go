package nsdyndns

// Version of the nsdyndnsd daemon. It can be overridden at build time
// with:
//
//	go build -ldflags "-X github.com/simon3z/nsdyndns.Version=..."
//
// The release tooling (rpm-build.sh) keeps this in sync with the
// release version, falling back to git tags.
var Version = "0.1.1"
