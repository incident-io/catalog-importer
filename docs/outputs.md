# Outputs

Outputs define the resulting catalog types and entries, and are best understood
by example. Take this pipeline:

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
          categories: ['service'],
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
syntax and what is possible in these fields, and our guide on [external IDs and
aliases](aliases-and-external-ids.md) to understand how these special fields
work.

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
- The source is a Javascript expression (see [docs](expressions.md)) that is evaluated
  against the source entry to find the value for this attribute.

For more information on how to use filter expressions, read [Using
expressions](expressions.md) or look at the [Backstage](backstage) example for
real-life use cases.

### Enum attribute

Enums are useful when you have an attribute of 'String' type (both array and non-array), that you'd like to have as as separate catalog type, such as tags. Using the above example of `BackstageAPIType`, we can instead generate it from `BackstageAPI`

```jsonnet
{
  sync_id: 'example-org/example-repo',
  pipelines: [
    // Backstage API
    {
      sources: [
        {
          inline: {
            entries: [
              {
                name: "Payments API",
                external_id: "payments"
                type: "grpc",
              }
            ],
          },
        },
      ],
      outputs: [
        {
          name: 'Backstage API',
          description: 'APIs that we have',
          type_name: 'Custom["BackstageAPI"]',
          source: {
            name: 'name',
            external_id: 'external_id',
          },
          categories: ['service'],
          attributes: [
            {
              id: "type",
              name: "API type",
              array: false,
              source: "$.type"
              enum: {
                name: 'Backstage API Type',
                description: 'Type or format of the API.',
                type_name: 'Custom["BackstageAPIType"]',
                enable_backlink: true,
              },
            },
          ],
        },
      ],
    },
  ],
}
```

The above we generate the following catalog types:

- `BackstageAPI` with attributes:
  - `Name`
  - `API type`
- `BackstageAPIType` with attributes:
  - `Name`
  - `Backstage API`

The `enable_backlink` option allows you to specify if the created enum should have an attribute pointing back to the 
attribute that created it. If disabled, the `BackstageAPIType` above will not have a `Backstage API` attribute.


See [simple/importer.jsonnet](https://github.com/incident-io/catalog-importer/blob/bbb659c312af7c45a626a68643e1cd4e890376d5/docs/simple/importer.jsonnet#L161-L166) for a working example