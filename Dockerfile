FROM alpine:20230329 AS runtime

# goreleaser supplies this for us
COPY catalog-importer /usr/local/bin

ENTRYPOINT ["/usr/local/bin/catalog-importer"]
