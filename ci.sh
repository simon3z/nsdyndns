#!/usr/bin/env bash
# CI quality gate for nsdyndns.
# Usage: ./ci.sh
set -euo pipefail
cd "$(dirname "$0")"

echo "--- gofmt ---"
gofmt -l .

echo "--- go vet ---"
go vet ./...

echo "--- go build ---"
go build ./...

echo "--- go mod verify ---"
go mod verify

echo "--- go test ---"
go test ./...

echo "✓ All checks passed."
