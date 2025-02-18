//go:build tools

package tools

import (
	// Generate the OpenAPI client
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	// Test runner
	_ "github.com/onsi/ginkgo/v2/ginkgo"
)
