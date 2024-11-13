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
            VERSION="2.4.3"

            echo "Installing importer v${VERSION}..."
            curl -L \
              -o "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" \
              "https://github.com/incident-io/catalog-importer/releases/download/v${VERSION}/catalog-importer_${VERSION}_linux_amd64.tar.gz"
            tar zxf "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" -C /tmp
      - run:
          name: Sync
          command: |
            if [[ "${$CIRCLE_BRANCH}" == "master" ]]; then
              /tmp/catalog-importer sync --config importer.jsonnet
            else
              /tmp/catalog-importer sync --config importer.jsonnet --dry-run
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
          VERSION="2.4.3"

          echo "Installing importer v${VERSION}..."
          curl -L \
            -o "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" \
            "https://github.com/incident-io/catalog-importer/releases/download/v${VERSION}/catalog-importer_${VERSION}_linux_amd64.tar.gz"
          tar zxf "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" -C /tmp
      - name: Sync
        run: |
          /tmp/catalog-importer sync --config=importer.jsonnet --prune

```

## GitLab CI

If you run on GitLab CI, an example config is:

```yaml
# .gitlab-ci.yml

variables:
  IMPORTER_VERSION: "2.4.3"

sync:
  image: ubuntu:latest

  # You can use rules to control when the job runs
  rules:
    # Run on every push to any branch
    - if: $CI_COMMIT_BRANCH
      when: always

    # Alternatively, you can use scheduled pipelines
    # Configure this in GitLab UI: Settings > CI/CD > Pipeline schedules
    # - if: $CI_PIPELINE_SOURCE == "schedule"
    #   when: always

  before_script:
    # Install curl and other dependencies
    - apt-get update && apt-get install -y curl

    # Install catalog-importer
    - |
      echo "Installing importer v${IMPORTER_VERSION}..."
      curl -L \
        -o "/tmp/catalog-importer_${IMPORTER_VERSION}_linux_amd64.tar.gz" \
        "https://github.com/incident-io/catalog-importer/releases/download/v${IMPORTER_VERSION}/catalog-importer_${IMPORTER_VERSION}_linux_amd64.tar.gz"
      tar zxf "/tmp/catalog-importer_${IMPORTER_VERSION}_linux_amd64.tar.gz" -C /tmp

  script:
    # Run sync with different options based on branch
    - |
      if [ "$CI_COMMIT_BRANCH" = "master" ] || [ "$CI_COMMIT_BRANCH" = "main" ]; then
        /tmp/catalog-importer sync --config importer.jsonnet --prune
      else
        /tmp/catalog-importer sync --config importer.jsonnet --dry-run
      fi
```
