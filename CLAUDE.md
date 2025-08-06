# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Development
- `make` - Build the binary (output: `bin/catalog-importer`)
- `make test` - Run all tests using go test
- `ginkgo` - Use for running tests (NOT `go test` - this project uses Ginkgo)
- `go fmt` - Format Go files after making changes
- `goimports -w <files>` - Fix imports in Go files
- `make generate` - Generate code
- `make tools` - Install development tools from tools.go

### Platform-specific builds
- `make darwin` - Build for macOS
- `make linux` - Build for Linux

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
```bash
export INCIDENT_API_KEY="inc_development_YOUR_API_KEY"
export INCIDENT_ENDPOINT="http://localhost:3001/api/public"
```

## Architecture

The catalog-importer is a CLI tool for syncing catalog data from various sources into the incident.io catalog.

### Core Components

**CLI Structure**: Uses kingpin for command-line parsing. Main commands:
- `sync` - Primary command to sync data from sources
- `import` - Import catalog data directly or generate config
- `init` - Initialize new config from template
- `types` - Show available types for account

**Configuration Pipeline**: 
- Jsonnet-based configuration files define data pipelines
- Each pipeline consists of sources → transformations → outputs
- Config validation happens at boot time via `config/reference.jsonnet`

**Key Packages**:
- `config/` - Configuration parsing, validation, and pipeline management
- `source/` - Data source implementations (GitHub, local files, Backstage, GraphQL, etc.)
- `output/` - Output formatting and API client integration
- `reconcile/` - Entry reconciliation logic
- `expr/` - JavaScript expression evaluation for transformations
- `client/` - Auto-generated API client from OpenAPI spec

**Source Types**: Supports multiple source types including GitHub repos, local files, Backstage catalogs, GraphQL APIs, and inline data.

**Data Flow**: Sources → Jsonnet transformations → Validation → API sync with incident.io catalog

### Code Generation
- `client/client.gen.go` - Auto-generated from `client/openapi3.json` using oapi-codegen
- Regenerate with `make client/client.gen.go`