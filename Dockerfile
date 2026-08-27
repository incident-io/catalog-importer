FROM alpine:3.24.1 AS runtime

# Bring the base image up to the current patch versions, then add certificates
# so we can make HTTPS requests.
#
# The upgrade is doing the security work here. OpenSSL is present in the base
# image regardless of what we install, because busybox's ssl_client depends on
# it, so pinning the base alone leaves us on whatever OpenSSL that base release
# happened to ship. Alpine publishes fixes into the 3.24 repository well before
# it cuts a new base image, and this is how we pick them up.
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates

# goreleaser supplies this for us.
COPY catalog-importer /usr/local/bin

ENTRYPOINT ["/usr/local/bin/catalog-importer"]
