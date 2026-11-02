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

See [simple/importer.jsonnet](https://github.com/incident-io/catalog-importer/blob/8d0f02c57598330defe14ed2ce783427d290450c/docs/simple/importer.jsonnet#L161-L166) for a working example

### Schema-only attributes

By default, once a catalog type is managed from catalog-importer, it cannot be
edited from the incident.io dashboard. This prevents ensures that your catalog
always reflects what is defined in code.

However, you may want to have some attributes on a catalog type that aren't
managed in code; we call these "schema-only" attributes in catalog-importer, since
the schema is managed in code but the data is managed in the dashboard.

For example, you might want to have an 'escalation path' attribute on your
BackstageGroup type, which is managed in the dashboard:

```jsonnet
{
  sync_id: 'example-org/example-repo',
  pipelines: [
    // Backstage API
    {
      name: 'Backstage Group',
      description: 'Groups synced from Backstage.',
      type_name: 'Custom["BackstageGroup"]',
      categories: ['team'],
      source: {
        filter: '$.apiVersion == "backstage.io/v1alpha1" && $.kind == "Group"',
        name: '$.metadata.name',
        external_id: '$.metadata.namespace + "/" + $.metadata.name',
        aliases: [
          '$.metadata.name',
        ],
      },
      attributes: [
        // Attributes set from Backstage:
        {
          id: 'parent',
          name: 'Parent',
          type: 'Custom["BackstageGroup"]',
          source: '$.spec.parent',
        },
        ...

        // Schema-only attributes:
        {
          id: 'escalation-path',
          name: 'Escalation path',
          type: "EscalationPath",
          schema_only: true,
        },
      ],
    },
  ],
}
```

### Team ownership

Set `owning_team_ids` on an output to control which teams own the resulting
catalog type. The value is a list of catalog entry IDs for the teams:

```jsonnet
{
  name: 'Team',
  type_name: 'Custom["Team"]',
  owning_team_ids: ['01FCNDV6P870EA6S7TK1DSYDG0'],
  ...
}
```

Ownership is only managed when the field is present:

- Omit `owning_team_ids` to leave ownership untouched. Owners set in the
  dashboard are preserved across syncs.
- Set it to a list of team IDs to own the type by exactly those teams.
- Set it to `[]` to clear all owners.

When team catalog permissions are set up for your organisation, the API key's
team scope matters, and the simplest option is a key without team restrictions,
which is always allowed:

- Creating a type with owners requires the key to have catalog type create
  permission on every team you reference.
- Changing the owners of a type that is already owned is a global action: a
  team-scoped key cannot reassign ownership even when it controls the teams
  involved. Syncs that leave ownership unchanged are unaffected.

If team permissions aren't set up, none of these restrictions apply.

## Synced types

Some catalog types are synced by incident.io from an integration, such as
`Schedule`, `EscalationPath` and `User`. You can't create or edit their entries
with a normal output, but you can add to them with `synced_type_outputs`:

- **Custom aliases**, so other catalog entries and alert attributes can refer to
  an entry by a name you choose, such as a Slack handle for a schedule.
- **Attributes you added to the type**, such as a runbook URL on each
  escalation path.

```jsonnet
{
  sync_id: 'example-org/example-repo',
  pipelines: [
    {
      sources: [
        {
          inline: {
            entries: [
              {
                schedule: 'Primary on-call',
                handles: ['payments-oncall', 'payments'],
                runbook: 'https://example.com/runbooks/payments',
              },
            ],
          },
        },
      ],
      synced_type_outputs: [
        {
          type_name: 'Schedule',
          source: {
            match: '$.schedule',
            custom_aliases: ['$.handles'],
          },
          attributes: [
            { id: 'runbook_url', source: '$.runbook' },
          ],
        },
      ],
    },
  ],
}
```

Run `catalog-importer types` to find the type name of a synced type. A pipeline
can have outputs, synced type outputs, or both.

### Matching entries

The integration creates and deletes entries in a synced type, so the importer
never does. Instead, `source.match` finds the entry that already exists for each
source entry. We try the value as the entry's ID, then its external ID, then its
name, then one of the aliases the integration syncs. Custom aliases are not used
for matching, because the importer sets them.

If a source entry matches no entry (for example, the schedule was deleted), or
matches more than one (for example, two schedules have the same name), the
importer reports it and skips it. It still syncs every other entry and output,
then exits with an error, so a scheduled run fails and you can fix the source.
If two source entries match the same entry, the first one wins and the second is
reported in the same way.

Entries that no source entry matches are left alone.

### What the importer owns

For each entry it matches, the importer only writes:

- The whole list of custom aliases, when `source.custom_aliases` is set. Any
  custom alias it doesn't produce is removed, and an empty result clears them.
  Omit `source.custom_aliases` to leave custom aliases alone. An alias the
  integration already syncs onto the entry is skipped, as the entry already
  answers to it.
- The value of each attribute in `attributes`. If the source has no value, the
  importer removes it.

It never changes the entry's name, external ID, synced aliases or synced
attributes, and the entries stay editable in the dashboard. Some types, such as
alert priorities, manage their aliases elsewhere in incident.io and take no
custom aliases: the importer reports an error if you set `source.custom_aliases`
for one of them.

### Attributes

Each attribute must already exist on the synced type. Add it in the incident.io
dashboard first, then refer to it by its ID. The importer takes the attribute's
type from the catalog type, so you only give `id` and, optionally, `source`
(which defaults to the field with the same name as the ID). You can't set an
attribute that the integration syncs.

### Users and team membership

incident.io finds the members of a team through the entries of some types, such
as `User`. A custom alias on one of those entries can change who belongs to a
team, so changing them needs an API key with the `team.memberships_update`
scope (the "Can manage team memberships and edit catalog entries" role). The
importer's usual API key doesn't have it. Without it, the sync fails with an
error that says which scope is missing.
