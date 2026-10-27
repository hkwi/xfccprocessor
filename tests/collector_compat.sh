#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT HUP INT TERM

cd "$test_dir"
go mod init example.com/xfccprocessor-collector-compat
go mod edit \
    -go=1.26.0 \
    -require=github.com/hkwi/xfccprocessor@v0.0.0 \
    -replace="github.com/hkwi/xfccprocessor=$repo_dir" \
    -require=go.opentelemetry.io/collector/exporter/otlpexporter@v0.164.0

cat > collector_test.go <<'EOF'
package compat

import (
    "testing"

    "github.com/hkwi/xfccprocessor"
    "go.opentelemetry.io/collector/exporter/otlpexporter"
)

func TestFactories(t *testing.T) {
    if xfccprocessor.NewFactory() == nil || otlpexporter.NewFactory() == nil {
        t.Fatal("component factory is nil")
    }
}
EOF

# Collector v0.164.0 requires Go 1.26. Resolve the consumer graph without excludes
# or Collector replacements; standalone processor tests do not expose overlaps.
go mod tidy
go test -mod=readonly ./...
