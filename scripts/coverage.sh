#!/usr/bin/env bash
#
# coverage.sh - Go test coverage report for nsdyndns.
#
# Usage: ./scripts/coverage.sh

set -euo pipefail
cd "$(dirname "$0")/.."

go test -coverprofile=coverage.out ./...
go tool cover -func coverage.out

go tool cover -html coverage.out -o coverage.html
echo "HTML report written to coverage.html"
