# Deploying using CI

Most people run the catalog from their CI pipelines, where they either sync on
merge or trigger syncs periodically depending on their needs.

## CircleCI

If you run on CircleCI, an example config is below.

> You can configure a [scheduled pipeline](https://circleci.com/docs/scheduled-pipelines/)
> to run the sync on a regular cadence. This is recommended if your importer
> config uses sources other than local files.

```yaml
# .circleci/config.yml
---
version: 2.1

jobs:
  sync:
    docker:
      - image: cimg/base:2023.04
    working_directory: ~/app
    steps:
      - checkout
      - run:
          name: Install catalog-importer
          command: |
            VERSION="0.13.0"

            echo "Installing importer v${VERSION}..."
            curl -L \
              -o "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" \
              "https://github.com/incident-io/catalog-importer/releases/download/v${VERSION}/catalog-importer_${VERSION}_linux_amd64.tar.gz"
            tar zxf "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" -C /tmp
      - run:
          name: Sync
          command: |
            if [[ "${$CIRCLE_BRANCH}" == "master" ]]; then
              /tmp/catalog-importer sync --config config.jsonnet
            else
              /tmp/catalog-importer sync --config config.jsonnet --dry-run
            fi

workflows:
  version: 2
  sync:
    jobs:
      - sync
```

## GitHub Actions

If you run on GitHub Actions, an example config is:

```yaml
name: Sync

# Run on every push.
on: [push]

# Alternatively, run on a schedule.
# on:
#   schedule:
#     - cron: "55 * * * *" # hourly, on the 55th minute

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        with:
          fetch-depth: 0
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.20"
      - name: Install catalog-importer
        run: |
          go install github.com/incident-io/catalog-importer/cmd/catalog-importer@latest
      - name: Sync
        run: |
          catalog-importer sync --config=config.jsonnet --prune
```

