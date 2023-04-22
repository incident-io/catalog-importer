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
