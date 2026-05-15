#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

go test ./...
pushd web && npm run test && npm run build && popd  # outputs to internal/api/static/
go build -o letrvu ./cmd/letrvu
./letrvu -addr :8080
