# Config (importer.jsonnet)

The importer is powered by a configuration file which defines the pipelines that
you use to get data into the catalog. Each pipeline consists of sources and
outputs.

This file is given to the importer via the `--config` flag:

```console
$ catalog-importer validate --config=config.jsonnet
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

- [`inline`](sources#inline) for defining inside the importer config
- [`local`](sources#local) from local files
- [`backstage`](sources#backstage) for catalog data pulled from the Backstage API
- [`github`](sources#github) to load from files in GitHub repositories
- [`exec`](sources#local) from the output of a command

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

Our config examples use [Jsonnet][jsonnet], a tool which makes working with more
complex JSON files easier. The `catalog-importer` tool includes Jsonnet support,
but installing language support to your editor will make the process a lot
smoother (e.g. [VSCode extension][vscode]).

If you don't want to use Jsonnet, switch to whichever you prefer of JSON or
YAML: both will work fine.

```console
$ catalog-importer validate --config=config.json    # also works
$ catalog-importer validate --config=config.yaml    # ...as does this
$ catalog-importer validate --config=config.jsonnet # ...and this
```
