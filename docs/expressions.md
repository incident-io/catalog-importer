# Expressions

The importer allows filtering source entries and for transforming fields from
the source when mapping them into output attributes.

These expressions are written in JavaScript, which allows for easy field
manipulation, and filtering of source data.

Note: Prior to version `2.0.0`, we used [CEL](https://github.com/google/cel-spec)
for our expressions. This is no longer supported, but our migration to
JavaScript should make this process simpler.

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

Given an example entry of:

```json
{
  "name": "Website",
  "details": {
    "description": "Marketing website",
    "owner": {
      "team": "Engineering"
    }
  },
  "runbooks": ["https://github.com/incident-io/runbooks/blob/main/website.md"]
}
```

The following expressions would evaluate to:

- `$.name` → `Website`
- `$.details.description` → `Marketing website`
- `$.runbooks` → `["https://github.com/incident-io/runbooks/blob/main/website.md"]`
- `$.runbooks[0]` → `https://github.com/incident-io/runbooks/blob/main/website.md`

You can also [use `get`](https://underscorejs.org/#get) to evaluate nested fields that may be null.
For the example entry above, the following expressions would evaluate to:

- `_.get($.metadata, "name")` → `null`
- `_.get($.metadata, "name", "default name")` → `default name`
- `_.get($.details, "alias")` → `null`
- `_.get($.details, "alias", "default alias")` → `default alias`
- `_.get($.details, "description")` → `Marketing website`
- `_.get($.details, ["description", "owner", "team"])` → Engineering`

## Migrating from CEL

As shown above, the main difference between CEL and JavaScript is that your
data is in scope as the variable `$`. This means that where previously your
source field referred to `name`, it now needs to refer to `$.name`. For most
cases, the migration is as simple as prepending `$.`.

You will also need to update the properties of the `source` in the `output` of
your pipelines.

## Outdated CEL functions

Prior to version `2.0.0`, we had implemented a handful of functions ourselves
to make the adoption of CEL a bit easier. Below is a description of each of
the functions that we removed, along with their JavaScript equivalent:

### `coalesce`

```json
{ "subject": ["one", null, "two"] }
```

- Before: `coalesce(subject)`
- After: `$.subject.filter(function(v){return v})`

Both would result in `["one", "two"]`.

### `first`

```json
{ "subject": ["one", null, "two"] }
```

- Before: `first(subject)` => `["one"]`
- After: `$.subject.slice(0, 1)`

Both would result in `["one"]`. You can pull out the first object from the
array by using an index, such as `$.subject[0]`.

### `trimPrefix` and `replace`

```json
{ "subject": "group:engineering@example.com" }
```

- Before: `trimPrefix(subject, "group:")` (gives `engineering@example.com`)
- After:
  - `$.subject.replace(/^(group\:)/,"")` → `engineering@example.com`
  - `$.subject.slice(6)` → `engineering@example.com`
  - `$.subject.replace(/group:|@example.com/g, "")` → `engineering`
