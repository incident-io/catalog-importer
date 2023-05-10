# Config (importer.jsonnet)

The importer is powered by a JSON configuration file, which defines the
pipelines that you use to get data into the catalog. Each pipeline consists of
sources and outputs.

This file is given to the importer via the `--config` flag:

```console
$ catalog-importer validate --config=config.jsonnet
```

We explain how the configuration works below, but if you're already familiar or
want to just try things out, be sure to check the reference.jsonnet that
documents all possible configuration options:

- https://github.com/incident-io/catalog-importer/blob/master/config/reference.jsonnet

Note that the config can be JSON, YAML or Jsonnet: see [File
format](#file-format).

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

These are:

- [`inline`](#inline) for defining inside the importer config
- [`local`](#local) from local files
- [`exec`](#local) from the output of a command

### `inline`

When you have a small number of entries or want to define the entry contents
directly in the config file, you can use the inline source.

This looks like:

```jsonnet
// pipelines.*.sources.*
{
  inline: {
    // These entries are passed directly into the outputs.
    //
    // Note that outputs still need to map the entry fields to the
    // output attributes, but can do so by referencing the keys (e.g.
    // name, or description).
    entries: [
      {
        external_id: 'some-external-id',
        name: 'Name',
        description: 'Entry description',
      },
    ],
  },
}
```

If you use Jsonnet for your importer configuration, we recommend splitting your
config into catalog data and the importer pipelines.

Then you can import the data into your importer config like so:

```jsonnet
// importer.jsonnet
local catalog = import 'catalog.jsonnet';

{
  pipelines: [
    {
      sources: [
        {
          inline: {
            entries: catalog.teams, // loaded from catalog.jsonnet
          },
        },
      ],
    },
  ],
}
```

### `local`

When you store catalog data in files, such as for tools like Backstage with
`catalog-info.yaml`s, you can use the `local` source to load those files into
your pipeline.

This looks like:

```jsonnet
// pipelines.*.sources.*
{
  'local': {
    // List of file glob patterns to apply from the current directory.
    //
    // Files can be either YAML, JSON or Jsonnet form, and output either a
    // single entry or an array of entries.
    files: [
      'catalog-info.yaml',
      'pkg/integrations/*/config.yaml',
    ],
  },
}
```

This is often used when running an importer sync from a monorepo CI pipeline,
where that monorepo contains all the catalog files. We support YAML, JSON or
Jsonnet, and will flatten the resulting entries into a single array.

An example YAML file might be:

```yaml
---
description: This is a multi-doc YAML
---
description: And would be parsed as two entries
```

Or you want to store things in JSON:

```json
{
  "description": "Happily parsed as a single entry"
}
```

And if needed, a JSON file that contains many entries:

```json
[
  { "description": "But maybe you want" },
  { "description": "more than one entry per-file?" },
  { "description": "That's fine too!" }
]
```

And finally, support for Jsonnet:

```jsonnet
std.map(function(desc) { description: desc }, [
  "Your Jsonnet files",
  "will also load fine",
])
```

### `backstage`

If you already have a Backstage catalog setup, you can ask the importer to pull
directly from the Backstage API.

This looks like:

```jsonnet
// pipelines.*.sources.*
{
  backstage: {
    endpoint: 'https://backstage-internal.example.com/api/catalog/entities',
    token: '<bearer-token>',
  },
}
```

[backstage-endpoint]: https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entities

The `endpoint` should be pointing at whatever URL maps to [GET
/entries][backstage-endpoint] on your Backstage instance, and the `token` should
be a bearer token with permissions to make this call.

This will pull in all catalog entries after which you may use the source
`filter`s to separate entries into different types.

### `github`

This source can pull files matching a pattern from across repositories in a
GitHub account, and is useful when you already have catalog files such as the
catalog-info.yaml used by Backstage.

> This is not yet built, but please contact us if this would be your preferred
> way of sourcing catalog data.

This would look like:

```jsonnet
// pipelines.*.sources.*
{
  github: {
    repos: [
      "example-org/example-repo",
    ],
    files: [
      "catalog-info.yaml",
    ],
  },
}
```

### `exec`

When you can't easily source catalog data from files or don't want it to be
inline, you can use the `exec` source to generate catalog entries from the
output of a command.

The command must output the same type of data as is supported by the `local`
source.

There are a few common use cases for the `exec` source:

#### Complex transformations

While we support [expressions](expressions.md) for filtering and transforming
source data, you might have some more complex transformation better suited to
other tools.

For this, we'd advise using something like jq to perform the translation:

```jsonnet
// pipelines.*.sources.*
{
  exec: {
    // Use jq to turn an object of key to entry into a list of entries.
    command: [
      'jq',
      'to_entries | map(.value)',
      'catalog.json',
    ],
  },
}
```

#### Loading from data warehouse

Often you may want to sync data from your company's warehouse into the catalog,
such as a list of customers and who owns that customer relationship (great for
building automations to notify the right people when customers are impacted by
an incident).

Many warehouses will have tools that can execute queries and return the results
in JSON form. As an example, we (incident.io) load our customer list from
BigQuery like so:

```jsonnet
// pipelines.*.sources.*
{
  exec: {
    command: [
      'bq',
      'query',
      '--format=json',
      '--max_rows=10000',
      '--use_legacy_sql=false',
      '--headless',
      '--project_id=incident-io-catalog',
      (importstr 'customer.sql'),
    ],
  },
}
```

Where we've written a SQL query that finds all our customers in `customer.sql`,
then provided that as an argument to the `bq` tool.

#### From a remote API

If your catalog is available over an API on the same network as the importer,
you can use curl or a similar tool to pull that data.

```jsonnet
// pipelines.*.sources.*
{
  exec: {
    command: [
      'curl', 'https://internal-catalog.example.com/catalog_entries',
    ],
  },
}
```

## Outputs

Having defined sources for your data, you now need to specify the outputs.

Each output maps to a catalog type, and will 'own' all entries it syncs into
that type. This means the importer will remove any entries it finds in that
catalog type that are not present in the source, and the importer will refuse to
sync catalog type that were created from an importer with a different `sync_id`.

Outputs are best understood by example:

```jsonnet
{
  sync_id: 'example-org/example-repo',
  pipelines: [
    // Backstage API Type
    {
      sources: [
        {
          inline: {
            entries: [
              {
                name: 'OpenAPI',
                external_id: 'openapi',
                description:
                  'An API definition in YAML or JSON format based on the OpenAPI version 2 or version 3 spec.',
              },
              {
                name: 'gRPC',
                external_id: 'grpc',
                description:
                  'An API definition based on Protocol Buffers to use with gRPC.',
              },
            ],
          },
        },
      ],
      outputs: [
        {
          name: 'Backstage API Type',
          description: 'Type or format of the API.',
          type_name: 'Custom["BackstageAPIType"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              array: false,
              source: 'description',
            },
          ],
        },
      ],
    },
  ],
}
```

In this pipeline, we load entries from the `inline` source. Each entry will have
a `name`, `external_id` and `description` field.

The output will be a catalog type called `Backstage API Type`, with the unique
type name of `Custom["BackstageAPIType"]`. All custom outputs must have type
names of the form `Custom["<name of type>"]` and they must be unique: this type
name is how you reference this type for catalog attributes.

The `source` config for the output:

- Specifies that the name of the resulting catalog entry should be from the
  `name` field of the source.
- And similar that `external_id` should come from the `external_id` field.

It's worth reading our guide on [expressions](expressions.md) to understand the
syntax and what is possible in these fields.

The Backstage API Type (the catalog type we're creating for this output) will be
given a single attribute:

- Which will be called `Description`.
- The ID of this attribute is `description`, which is how we uniquely identify
  this attribute in the catalog type. This field makes it possible to rename
  this attribute without breaking references.
- It's type will be `Text`, supporting rich text. This could also point to any
  other catalog type by referencing it's type name, such as `Custom["Service"]`
  or even `Custom["BackstageAPIType"]` to reference itself.
- It is a single value attribute, because `array` is false.
- The source is a CEL expression (see [docs](expressions.md)) that is evaluated
  against the source entry to find the value for this attribute.

For more information on how to use filter expressions, read [Using
expressions](expressions.md) or look at the [Backstage](backstage) example for
real-life use cases.

## File format

Each of our examples uses [Jsonnet][jsonnet], a tool which makes working with
more complex JSON files easier. The `catalog-importer` tool includes Jsonnet
support, but installing language support to your editor will make the process
a lot smoother. (e.g. [VSCode extension][vscode])

If you don't want to use Jsonnet, switch to whichever you prefer of JSON or
YAML: both will work fine.

```console
$ catalog-importer validate --config=config.json # also works
```
