# Docs

The easiest way to get started is to copy one of the examples and tweak it to
match your needs, which will depend on whether you already use a catalog and the
type of catalog data you have available.

Choose from:

- [Simple](simple), for anyone starting from scratch and wanting to load catalog
  data into incident.io directly from importer config.
- [Backstage](backstage), for those already using Backstage as a service catalog
  and want to import existing `catalog-info.yaml` files.

Once you've created a config.jsonnet, visit your incident dashboard to create an
API key with permission to:

- View data
- Manage organisation settings

Then you can run a sync with:

```console
$ catalog-importer sync --config=config.jsonnet

✔ Loaded config (3 pipelines, 3 sources, 3 outputs)
✔ Connected to incident.io API (https://api.incident.io)
✔ Found 16 catalog types, with 3 that match our sync ID (incident-io/catalog)
```

<details>
  <summary>
    Where this will be followed by progress output as entries are synced into
    the catalog:
  </summary>

```
↻ Creating catalog types that don't yet exist...
  ✔ Custom["Feature"] (id=01GYZMPSJPBE1ZFDF1ESEWFYZF)
  ✔ Custom["Integration"] (id=01GYZMPSV08SYE4RF49C3JZT76)
  ✔ Custom["Team"] (id=01GYZMPT7C692DXCEVHFHVKZAQ)

↻ Syncing catalog type schemas...
  ✔ Custom["Feature"] (id=01GYZMPSJPBE1ZFDF1ESEWFYZF)
  ✔ Custom["Integration"] (id=01GYZMPSV08SYE4RF49C3JZT76)
  ✔ Custom["Team"] (id=01GYZMPT7C692DXCEVHFHVKZAQ)

↻ Syncing pipeline... (Custom["Feature"])

  ↻ Loading data from sources...
    ✔ inline (found 30 entries)

  ↻ Syncing entries...

    ↻ Custom["Feature"]
      ✔ Building entries... (found 30 entries matching filters)
      ✔ No entries to delete
      ✔ Creating new entries in catalog... (30 entries to create)
      ✔ No existing entries to update

↻ Syncing pipeline... (Custom["Integration"])

  ↻ Loading data from sources...
    ✔ inline (found 21 entries)

  ↻ Syncing entries...

    ↻ Custom["Integration"]
      ✔ Building entries... (found 21 entries matching filters)
      ✔ No entries to delete
      ✔ Creating new entries in catalog... (21 entries to create)
      ✔ No existing entries to update

↻ Syncing pipeline... (Custom["Team"])

  ↻ Loading data from sources...
    ✔ inline (found 3 entries)

  ↻ Syncing entries...

    ↻ Custom["Team"]
      ✔ Building entries... (found 3 entries matching filters)
      ✔ No entries to delete
      ✔ Creating new entries in catalog... (3 entries to create)
      ✔ No existing entries to update
```

</details>

## Continuous integration (CI)

Most people run the catalog from their CI pipelines, where they either sync on
merge or trigger syncs periodically depending on their needs.

### CircleCI

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
            VERSION="0.4.0"

            echo "Installing importer v${VERSION}..."
            curl -L \
              -o "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" \
              "https://github.com/incident-io/catalog-importer/releases/download/v${VERSION}/catalog-importer_${VERSION}_linux_amd64.tar.gz"
            tar zxf "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" -C /tmp
      - run:
          name: Sync
          command: |
            /tmp/catalog-importer sync --config config.jsonnet

workflows:
  version: 2
  sync:
    jobs:
      - sync:
          filters:
            branches:
              only: master # or main
```

### GitHub Actions

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
      -
        name: Checkout
        uses: actions/checkout@v3
        with:
          fetch-depth: 0
      -
        name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      -
        name: Install catalog-importer
        run: |
          go install github.com/incident-io/catalog-importer/cmd/catalog-importer@latest
      -
        name: Sync
        run: |
          catalog-importer sync --config=config.jsonnet --prune
```
