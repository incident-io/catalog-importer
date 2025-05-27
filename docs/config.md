# Config (importer.jsonnet)

The importer is powered by a configuration file which defines the pipelines that
you use to get data into the catalog. Each pipeline consists of sources and
outputs.

This file is given to the importer via the `--config` flag:

```console
$ catalog-importer validate --config=importer.jsonnet
```

We explain how the configuration works below, but if you're already familiar or
want to just try things out, be sure to check the reference.jsonnet that
documents all possible configuration options:

- https://github.com/incident-io/catalog-importer/blob/master/config/reference.jsonnet

Note that the config can be JSON, YAML or Jsonnet: see [File format](#file-format).

## What is config?

The configuration file defines a number of pipelines, where a pipeline
specifies:

- **Sources** define where your data comes from. This might be inline data, local
  files, files from GitHub, or the output of a command.

- **Outputs** define the resulting catalog types and entries, including the
  attributes that the type should have, and how their values should be populated
  from the source data.

Sources define where the importer will find the data it will try loading into
the catalog, while the outputs are about the catalog types you want to push data
into.

## Sources

We support several sources, catering for all the ways people normally store
their catalog data.

These are, with details in the links:

- [`inline`](sources.md#inline) for defining inside the importer config
- [`local`](sources.md#local) from local files
- [`backstage`](sources.md#backstage) for catalog data pulled from the Backstage API
- [`github`](sources.md#github) to load from files in GitHub repositories
- [`exec`](sources.md#local) from the output of a command

View more details in [Sources](sources.md).

## Outputs

Having defined sources for your data, you now need to specify the outputs.

Each output maps to a catalog type, and will 'own' all entries it syncs into
that type. This means the importer will remove any entries it finds in that
catalog type that are not present in the source, and the importer will refuse to
sync catalog type that were created from an importer with a different `sync_id`.

View more details in [Outputs](outputs.md).

## File format

Our config examples use [Jsonnet](https://jsonnet.org/), a tool which makes working with more
complex JSON files easier. The `catalog-importer` tool includes Jsonnet support,
but installing language support to your editor will make the process a lot
smoother (e.g. [VSCode extension](https://github.com/grafana/vscode-jsonnet)).

If you don't want to use Jsonnet, switch to whichever you prefer of JSON or
YAML: both will work fine.

```console
$ catalog-importer validate --config=config.json    # also works
$ catalog-importer validate --config=config.yaml    # ...as does this
$ catalog-importer validate --config=importer.jsonnet # ...and this
```
