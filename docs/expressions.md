# Expressions

The importer allows filtering source entries and for transforming fields from
the source when mapping them into output attributes.

These expressions are written in plain Javascript, which allows for easy field
manipulation.

Note: Prior to version `2.0.0`, we expected [CEL](https://github.com/google/cel-spec)
for our expressions. This is no longer supported, but our migration
to plain Javascript should make this process massively simpler!

We explain how to read and write expressions in this doc, using snippets taken
from our example configurations.

## Where can I use expressions?

Taking an example from the Backstage config (extended slightly):

```jsonnet
{
  pipelines: [
    {
      outputs: [
        // Backstage API
        {
          name: 'Backstage API',
          description: 'APIs synced from Backstage.',
          type_name: 'Custom["BackstageAPI"]',
          source: {
            filter: '$.apiVersion == "backstage.io/v1alpha1" && $.kind == "API"',
            name: '$.metadata.name',
            external_id: '$.metadata.name',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: '$.metadata.description',
            },
            {
              id: 'google_group',
              name: 'Google Group',
              type: 'String',
              source: '$.metadata.google_group.replace("group:", "")',
            },
          ]
        }
      ]
    }
  ]
}
```

The fields that use expressions are:

- `pipelines.*.outputs.*.source.filter` which is an expression controlling which
  of the source entries are provided to this output. Only when this expression
  evaluates to true for an entry will it be synced into the Backstage API
  catalog type.
- `pipelines.*.outputs.*.source.{name,external_id}` which are evaluated to
  provide the values for the name and external ID of the resulting catalog
  entry.
- `pipelines.*.outputs.*.attributes.*.source` as above, used to determine the
  resulting value of the attribute for this catalog entry.


## Further examples

Given the example entry of:

```json
{
  "name": "Website",
  "details": {
    "description": "Marketing website"
  },
  "runbooks": [
    "https://github.com/incident-io/runbooks/blob/main/website.md"
  ]
}
```

The following JS expressions would evaluate to:

- `$.name` => `Website`
- `$.details.description` => `Marketing website`
- `$.runbooks` => `["https://github.com/incident-io/runbooks/blob/main/website.md"]`
- `$.runbooks[0]` => `https://github.com/incident-io/runbooks/blob/main/website.md`

## Previously implemented CEL expressions

Prior to v. `2.0.0`, we had implemented a handful of functions ourselves
to make the adoption of CEL a bit easier. The migration to plain JS should
make their replacements both user-friendly and flexible, here are some examples:

### `first`
```json
[
  "one", null, "two"
]
```
Previously:
`first(subject)` => `["one"]`

Using JS:
`$.subject.slice(0, 1)`

### `trimPrefix` and `replace`
```json
"group:engineering@example.com"
```
Previously:
`trimPrefix(subject, "group:")` => `engineering@example.com`

Using JS:
`$.subject.replace(/^(group\:)/,"")` => `engineering@example.com`
`$.subject.slice(6)` => `engineering@example.com`
`$.subject.replace(/group:|@example.com/g, "")` => `engineering`
