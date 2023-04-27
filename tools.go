//go:build tools

package tools

import (
	// Generate the OpenAPI client
	_ "github.com/deepmap/oapi-codegen/cmd/oapi-codegen"
	// Test runner
	_ "github.com/onsi/ginkgo/v2/ginkgo"
)
