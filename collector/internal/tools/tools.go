//go:build tools
// +build tools

package tools

// based on https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/internal/tools/tools.go
import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/addlicense"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/ory/go-acc"
	_ "github.com/pavius/impi/cmd/impi"
	_ "github.com/tcnksm/ghr"
	_ "go.opentelemetry.io/collector/cmd/mdatagen"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
