FROM alpine:3.24.2 AS runtime

# Add certificates so we can make HTTPS requests, and upgrade the base
# packages: the versions in the base image have vulnerabilities that an
# update addresses. curl and jq are there for exec sources, which our docs
# show using them, and the image runs as nobody so users can't install them.
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates curl jq

# goreleaser supplies this for us.
COPY catalog-importer /usr/local/bin

# Run as nobody:nobody. The numeric form needs no passwd entry, so it keeps
# working if the base image changes.
USER 65534:65534

ENTRYPOINT ["/usr/local/bin/catalog-importer"]
