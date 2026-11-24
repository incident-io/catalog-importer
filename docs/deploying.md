# Deploying

Each run of the importer syncs once and exits. Most people run it from CI,
either when the config changes or on a schedule. If it reads from a system that
changes on its own, such as the Backstage API, run it on a schedule so
incident.io keeps up.

The run needs to reach both incident.io and your source. Hosted CI runners
usually can't reach an internal Backstage, so for that you can use a
self-hosted runner inside your network, or run it in your own infrastructure
with a [Kubernetes CronJob](#kubernetes-cronjob) or [cron and
Docker](#cron-and-docker).

If your catalog types already exist in incident.io, the **Manage in GitHub**
button on the Catalog page sets up a repository for you, including GitHub
Actions workflows like the ones below.

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
            VERSION="2.12.6"

            echo "Installing importer v${VERSION}..."
            curl -L \
              -o "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" \
              "https://github.com/incident-io/catalog-importer/releases/download/v${VERSION}/catalog-importer_${VERSION}_linux_amd64.tar.gz"
            tar zxf "/tmp/catalog-importer_${VERSION}_linux_amd64.tar.gz" -C /tmp
      - run:
          name: Sync
          command: |
            if [[ "${CIRCLE_BRANCH}" == "master" || "${CIRCLE_BRANCH}" == "main" ]]; then
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

If you run on GitHub Actions, use two workflows: one that syncs when changes
reach your main branch, and one that runs a dry-run on pull requests so you can
review the changes before they merge. Both use the Docker image, and expect an
`INCIDENT_API_KEY` repository secret.

```yaml
# .github/workflows/import-catalog.yml
name: Import catalog

on:
  push:
    branches: ["main", "master"]
  workflow_dispatch: # allow manual runs

  # Alternatively, run on a schedule. This is recommended if your importer
  # config uses sources other than local files.
  # schedule:
  #   - cron: "55 * * * *" # hourly, on the 55th minute

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - name: Sync
        run: |
          docker run \
            -v "$(pwd)":/config --workdir /config \
            -e 'INCIDENT_API_KEY=${{ secrets.INCIDENT_API_KEY }}' \
            -e "SOURCE_REPO_URL=${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}" \
            --rm \
            incidentio/catalog-importer:v2 \
            sync --config importer.jsonnet
```

```yaml
# .github/workflows/import-catalog-dry-run.yml
name: Import catalog (dry run)

on:
  pull_request:
    branches: ["main", "master"]

jobs:
  dry-run:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - name: Dry run
        run: |
          docker run \
            -v "$(pwd)":/config --workdir /config \
            -e 'INCIDENT_API_KEY=${{ secrets.INCIDENT_API_KEY }}' \
            --rm \
            incidentio/catalog-importer:v2 \
            sync --config importer.jsonnet --dry-run
```

The `v2` tag follows the latest 2.x release. To control when you upgrade, pin a
specific version such as `v2.12.6`; the available tags are on
[Docker Hub](https://hub.docker.com/r/incidentio/catalog-importer/tags).

The image is based on Alpine and runs as the unprivileged `nobody` user, so
your repository needs to be readable by that user. It includes `curl` and `jq`
for use in [`exec` sources](sources.md#exec).

## GitLab CI

If you run on GitLab CI, an example config is:

```yaml
# .gitlab-ci.yml

variables:
  IMPORTER_VERSION: "2.12.6"

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
    # Install curl on this Ubuntu image. Our Docker image is Alpine-based and
    # already includes curl and jq.
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

## Kubernetes CronJob

If Backstage runs in Kubernetes, run the importer next to it. Build an image
that adds your config to ours:

```dockerfile
# Dockerfile, next to importer.jsonnet
FROM incidentio/catalog-importer:v2
COPY . /config
WORKDIR /config
```

```console
docker build -t registry.example.com/catalog-importer-config:1 .
docker push registry.example.com/catalog-importer-config:1
```

Put the credentials in a Secret, then schedule the sync:

```console
kubectl create secret generic catalog-importer \
  --from-literal=INCIDENT_API_KEY='<incident.io API key>' \
  --from-literal=BACKSTAGE_TOKEN='<Backstage token>'
```

```yaml
# cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: catalog-importer
spec:
  schedule: "0 * * * *" # hourly
  concurrencyPolicy: Forbid # never start a sync while the last one is running
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: catalog-importer
              image: registry.example.com/catalog-importer-config:1
              args: ["sync", "--config", "importer.jsonnet", "--no-progress"]
              envFrom:
                - secretRef:
                    name: catalog-importer
```

```console
kubectl apply -f cronjob.yaml
kubectl create job --from=cronjob/catalog-importer catalog-importer-now # run one now
```

## Cron and Docker

On any machine with Docker, a crontab entry is enough. Keep the credentials in
an env file outside the config directory:

```console
$ cat /etc/catalog-importer.env
INCIDENT_API_KEY=<incident.io API key>
BACKSTAGE_TOKEN=<Backstage token>
```

```crontab
# Sync hourly, at 15 minutes past.
15 * * * * docker run --rm -v /opt/catalog-importer:/config --workdir /config --env-file /etc/catalog-importer.env incidentio/catalog-importer:v2 sync --config importer.jsonnet --no-progress
```

The image runs as the unprivileged `nobody` user, so the config directory must
be readable by everyone.
