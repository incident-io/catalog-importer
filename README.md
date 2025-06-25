# Catalog Importer

**Sync your service catalog data into incident.io from any source**

The catalog importer is the official CLI tool for syncing catalog data into [incident.io](https://incident.io/). It connects your existing catalog sources (GitHub repos, Backstage, local files, APIs) to incident.io's catalog, keeping your service information automatically up-to-date.

![Catalog dashboard](dashboard.png)

## Why use the catalog importer?

- **Single source of truth**: Keep your catalog data in your existing tools and workflows
- **Automatic synchronization**: Run in CI/CD to keep incident.io's catalog always current
- **Flexible data sources**: Support for GitHub, Backstage, local files, APIs, and more
- **Rich data transformation**: Filter, transform, and enrich your catalog data during import
- **Team ownership**: Maintain catalog data alongside your code where teams can easily update it

## Quick start

### 1. Install the importer

**macOS (recommended):**
```console
brew tap incident-io/homebrew-taps
brew install catalog-importer
```

**Other platforms:**
```console
go install -v github.com/incident-io/catalog-importer/v2/cmd/catalog-importer@latest
```

### 2. Choose your starting template

Get a working configuration in seconds:

```console
catalog-importer init
```

Choose from:
- **Simple**: Start from scratch with inline data (perfect for learning)
- **Backstage**: Import existing Backstage catalog-info.yaml files

### 3. Set up your API key

Create an API key at [app.incident.io/settings/api-keys](https://app.incident.io/settings/api-keys) with these permissions:
- View catalog types and entries
- Manage catalog types and edit catalog data

```console
export INCIDENT_API_KEY="your-api-key-here"
```

### 4. Test your setup

```console
catalog-importer validate --config=importer.jsonnet
catalog-importer sync --config=importer.jsonnet --dry-run
```

### 5. Go live

```console
catalog-importer sync --config=importer.jsonnet
```

## What's next?

- **[Complete documentation](docs)** - Comprehensive guides and examples
- **[Configuration reference](config/reference.jsonnet)** - All available options
- **[CI/CD setup](docs/deploying.md)** - Run automatically in your pipelines
- **[Data sources](docs/sources.md)** - Connect to GitHub, Backstage, APIs, and more

## Using Docker

A Docker image is available for containerised environments; see [Docker
Hub][hub] for more details of the image and available tags.

[hub]: https://hub.docker.com/r/incidentio/catalog-importer/tags

You may wish to deploy this on a scheduled basis to sync your catalog data. To do
that, you'll need to ensure that the necessary configuration is mounted into the
container and that the API key is supplied as an environment variable.

```console
docker run \
    -v $(pwd)/docs/simple:/config --workdir /config \
    -e 'INCIDENT_API_KEY=<key>' \
    --rm -it \
    incidentio/catalog-importer:latest \
    sync --config /config/importer.jsonnet
```

## Linking the Catalog UI to your importer repository

If you manage some of your catalog types through the catalog importer, you can
now pass a `--source-repo-url` parameter when running the catalog importer to
associate the URL of the repository where you're storing your catalog types
with those types.

This will prevent the catalog types you're syncing from being edited in the
catalog UI, and add a link in the UI from those types to your repository.

## Contributing

We're happy to accept open-source contributions or feedback. Just open a
PR/issue and we'll get back to you. This repo contains details on
[how to get started with local development](./development.md), and [how to publish a new release](./RELEASE.md).
