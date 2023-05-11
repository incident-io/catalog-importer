# Expressions

[cel]: https://github.com/google/cel-spec

The importer allows filtering source entries and for transforming fields from
the source when mapping them into output attributes.

These expressions are written in [CEL][cel], Google's Common Expression
Lanaguage, which is a lightweight expression syntax well suited to
transformations like this.

We explain how to read and write expressions in this doc, using snippets taken
from our example configurations.

## Where can I use expressions?

Taking an example from the Backstage config:

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
            filter: 'apiVersion == "backstage.io/v1alpha1" && kind == "API"',
            name: 'metadata.name',
            external_id: 'metadata.name',
          },
          attributes: [
            {
              id: 'description',
              name: 'Description',
              type: 'Text',
              source: 'metadata.description',
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

## How does CEL work?

CEL is very simple, and most of the time expressions will be simple references
into the source entry.

If you take an example entry of:

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

The following CEL expressions would evaluate to:

- `name` => `Website`
- `details.description` => `Marketing website`
- `runbooks` => `["https://github.com/incident-io/runbooks/blob/main/website.md"]`
- `runbooks[0]` => `https://github.com/incident-io/runbooks/blob/main/website.md`

## CEL functions

We've added some extension functions to CEL where they help solve a common use
case seen when syncing catalog data.

These are:

### `pluck`

Returns a list of elements built by mapping through objects and extracting the
value at the given selector.

On a CEL variable named `subject` of value:

```json
[
  { "key": "one", "another_key": "two" },
  { "key": "three", "another_key": "four" }
]
```

The following CEL expressions would evaluate to:

- `pluck(subject, "key")` => `["one", "three"]`
- `pluck(subject, "another_key")` => `["two", "four"]`

### `coalesce`

Like the SQL aggregate function, will remove all null values from a list.

On a CEL variable named `subject` of value:

```json
[
  "one", null, "two",
]
```

The following CEL expressions would evaluate to:

- `coalesce(subject)` => `["one", "two"]`

### `first`

Returns the first value from a list, if it has one.

On a CEL variable named `subject` of value:

```json
[
  "one", null, "two",
]
```

- `first(subject)` => `["one"]`

### `trimPrefix`

Removes the given prefix from a string.

On a CEL variable named `subject` of value:

```json
"group:engineering@example.com"
```

- `trimPrefix(subject, "something:")` => `group:engineering@example.com`
- `trimPrefix(subject, "group:")` => `engineering@example.com`
