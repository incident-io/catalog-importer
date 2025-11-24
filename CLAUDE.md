# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Development
- `make` or `make prog` - Build the binary (output: `bin/catalog-importer`)
- `ginkgo` - Run tests (IMPORTANT: Always use Ginkgo, NOT `go test`)
- `ginkgo -r` - Run all tests recursively
- `ginkgo -v` - Run tests with verbose output
- `ginkgo --focus="<pattern>"` - Run only tests matching pattern
- `go fmt` - Format Go files after making changes
- `goimports -w <files>` - Fix imports in Go files
- `make generate` - Generate code (runs `go generate ./...`)
- `make tools` - Install development tools from tools.go (oapi-codegen, ginkgo)
- `make clean` - Remove built binaries

### Platform-specific Builds
- `make darwin` - Build for macOS AMD64 (output: `bin/catalog-importer.darwin_amd64`)
- `make linux` - Build for Linux AMD64 (output: `bin/catalog-importer.linux_amd64`)

### Docker
```bash
docker run \
    -v $(pwd)/docs/simple:/config --workdir /config \
    -e 'INCIDENT_API_KEY=<key>' \
    --rm -it \
    incidentio/catalog-importer:latest \
    sync --config /config/importer.jsonnet
```

### Local Development Environment Variables
For incident.io staff developing against a local instance:
```bash
export INCIDENT_API_KEY="inc_development_YOUR_API_KEY"
export INCIDENT_ENDPOINT="http://localhost:3001/api/public"
```

## Architecture

The catalog-importer is a CLI tool for syncing catalog data from various sources into the incident.io catalog.

### Core Components

**CLI Structure**: Uses kingpin for command-line parsing. Main commands defined in `cmd/catalog-importer/cmd/`:
- `sync` - Primary command to sync data from sources (cmd/sync.go)
- `import` - Import catalog data directly or generate config (cmd/import.go)
- `init` - Initialize new config from template (cmd/init.go)
- `types` - Show available types for account (cmd/types.go)
- `validate` - Validate configuration files (cmd/validate.go)
- `backstage` - Backstage-specific commands (cmd/backstage.go)

**Configuration Pipeline**:
- Jsonnet-based configuration files define data pipelines
- Each pipeline consists of: sources → transformations → outputs
- Config validation happens at boot time via `config/reference.jsonnet`
- Configuration parsing is in `config/parser.go`
- Config loading (with caching support) is in `config/loader.go`

**Key Packages**:
- `config/` - Configuration parsing, validation, and pipeline management
  - `parser.go` - Jsonnet configuration parsing
  - `loader.go` - Config loading with caching support
  - `config.go` - Core config structures
- `source/` - Data source implementations
  - `source_github.go` - GitHub repository source
  - `source_local.go` - Local file system source
  - `source_backstage.go` - Backstage catalog source
  - `source_graphql.go` - GraphQL API source
  - `source_inline.go` - Inline data source
  - `source_exec.go` - Execute external commands as source
  - `credential.go` - Credential management
- `output/` - Output formatting and API client integration
  - `output.go` - Output handling
  - `collect.go` - Collecting output from sources
  - `marshal.go` - Marshaling output data
- `reconcile/` - Entry reconciliation logic
  - `entries.go` - Entry reconciliation between sources and API
- `expr/` - JavaScript expression evaluation for transformations
  - `js_eval.go` - JavaScript expression evaluation using Otto
- `client/` - Auto-generated API client from OpenAPI spec
  - `client.go` - Custom client wrapper and utilities
  - `client.gen.go` - Auto-generated types and client (DO NOT EDIT)

**Source Types**: Supports multiple source types including:
- GitHub repositories (recursive file discovery)
- Local file system (with glob patterns)
- Backstage catalogs
- GraphQL APIs
- Inline data (defined in config)
- External command execution

**Data Flow**:
1. Sources fetch raw data
2. Jsonnet transformations process and shape data
3. Validation against catalog schema
4. API sync with incident.io catalog via generated client
5. Reconciliation to determine create/update/delete operations

### Code Generation

**Generated Client**:
- `client/client.gen.go` - Auto-generated from `client/openapi3.json` using oapi-codegen
- Source: OpenAPI schema from the incident.io API (including secret endpoints for catalog operations)
- Regenerate with: `make client/client.gen.go`

**Update Process** (for incident.io staff):
```bash
# Copy the public schema with secret endpoints from your core repository
cp ../core/server/lib/openapi/public-schema-v3-including-secret-endpoints.json client/openapi3.json

# Regenerate the client
make client/client.gen.go

# Format the generated code
go fmt ./client/...

# Verify it compiles
make
```

Note: The schema should be copied from `server/lib/openapi/`, NOT from `server/api/gen/http/` (which contains the full internal API).

### Testing

**Test Framework**: Uses Ginkgo (BDD testing framework) + Gomega (matcher library)
- All test suites use `suite_test.go` pattern for Ginkgo setup
- Tests are in `*_test.go` files alongside production code
- IMPORTANT: Always run tests with `ginkgo`, NOT `go test`

**Test Packages**:
- `config/` - Config parsing and validation tests
- `source/` - Source implementation tests (including Backstage)
- `output/` - Output marshaling tests
- `reconcile/` - Entry reconciliation tests
- `expr/` - JavaScript evaluation tests
- `client/` - Client wrapper tests

**Running Tests**:
```bash
ginkgo -r              # Run all tests recursively
ginkgo ./config        # Run tests in specific package
ginkgo -v              # Verbose output
ginkgo --focus="regex" # Run tests matching pattern
```

### Development Tools

Defined in `tools.go` and installed with `make tools`:
- `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` - OpenAPI client generator
- `github.com/onsi/ginkgo/v2/ginkgo` - BDD test framework CLI

### Project Structure

```
catalog-importer/
├── cmd/catalog-importer/     # CLI entry point
│   ├── main.go              # Main entry point
│   └── cmd/                 # Command implementations
├── config/                  # Configuration parsing and management
├── source/                  # Data source implementations
├── output/                  # Output formatting and API integration
├── reconcile/              # Entry reconciliation logic
├── expr/                   # JavaScript expression evaluation
├── client/                 # API client (generated + custom)
├── docs/                   # Documentation and examples
├── bin/                    # Built binaries (gitignored)
├── Makefile               # Build targets
├── tools.go               # Development tool dependencies
└── CLAUDE.md              # This file
```

### Common Tasks

**Adding a new source type**:
1. Create `source/source_<type>.go`
2. Implement the source interface
3. Add tests in `source/source_<type>_test.go`
4. Register in config parser
5. Run `ginkgo ./source` to verify tests pass

**Updating API client**:
1. Copy latest schema from core repo: `cp ../core/server/lib/openapi/public-schema-v3-including-secret-endpoints.json client/openapi3.json`
2. Run `make client/client.gen.go`
3. Run `go fmt ./client/...`
4. Run `ginkgo -r` to verify compatibility
5. Update `client/client.go` if custom wrappers need changes

**Making code changes**:
1. Make your changes
2. Run `go fmt` on modified files
3. Run `goimports -w <files>` if imports changed
4. Run `ginkgo` in the relevant package to test
5. Run `make` to verify it compiles