FROM alpine:latest AS runtime

# Update package index and add certificates with latest versions
RUN apk update && apk add --no-cache ca-certificates

# Copy the catalog-importer binary (you'll need to extract this from the original image)
COPY catalog-importer /usr/local/bin

ENTRYPOINT ["/usr/local/bin/catalog-importer"]