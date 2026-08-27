FROM alpine:3.24.1 AS runtime

# Add certificates so we can make HTTPS requests.
#
# Use ca-certificates-bundle rather than ca-certificates. The latter depends on
# libcrypto3 (for the update-ca-certificates script), which pulls OpenSSL into
# the image. We build with CGO_ENABLED=0, so TLS comes from crypto/tls in the Go
# standard library and never links OpenSSL. The bundle package has no
# dependencies and provides the same certificate file.
RUN apk add --no-cache ca-certificates-bundle

# goreleaser supplies this for us.
COPY catalog-importer /usr/local/bin

ENTRYPOINT ["/usr/local/bin/catalog-importer"]
