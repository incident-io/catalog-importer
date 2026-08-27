FROM alpine:3.24.1 AS runtime

# Add certificates so we can make HTTPS requests, and upgrade the base
# packages: the versions in the base image have vulnerabilities that an
# update addresses.
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates

# goreleaser supplies this for us.
COPY catalog-importer /usr/local/bin

ENTRYPOINT ["/usr/local/bin/catalog-importer"]
