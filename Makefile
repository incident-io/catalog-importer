PROG=bin/catalog-importer
VERSION=$(shell git rev-parse --short HEAD)-dev
BUILD_COMMAND=go build -ldflags "-s -w -X main.Version=$(VERSION)"

################################################################################
# Build
################################################################################

.PHONY: prog darwin linux generate clean test

prog: $(PROG)
darwin: $(PROG:=.darwin_amd64)
linux: $(PROG:=.linux_amd64)

bin/%.linux_amd64:
	CGO_ENABLED=0 GOOS=linux $(BUILD_COMMAND) -a -o $@ cmd/$*/*.go

bin/%.darwin_amd64:
	CGO_ENABLED=0 GOOS=darwin $(BUILD_COMMAND) -a -o $@ cmd/$*/*.go

bin/%:
	$(BUILD_COMMAND) -o $@ cmd/$*/*.go

generate:
	go generate ./...

test:
	go test -v ./...

clean:
	rm -rfv $(PROG)

################################################################################
# Development
################################################################################

# Installs development tools from tools.go
tools:
	go mod download \
		&& cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %
################################################################################
# Clients
################################################################################

.PHONY: client/client.gen.go

client/client.gen.go:
	rm -rf $@
	oapi-codegen \
		--generate types,client \
		--package client \
		--o $@ \
		client/openapi3.json
