FROM alpine:3.22.2 AS runtime

# Add certificates so we can make HTTPS requests.
RUN apk add --no-cache ca-certificates

# goreleaser supplies this for us.
COPY catalog-importer /usr/local/bin

ENTRYPOINT ["/usr/local/bin/catalog-importer"]
