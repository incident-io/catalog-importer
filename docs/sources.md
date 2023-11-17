# Sources

We support several sources, catering for all the ways people normally store
their catalog data.

These are:

- [`inline`](#inline) for defining inside the importer config
- [`local`](#local) from local files
- [`backstage`](#backstage) for catalog data pulled from the Backstage API
- [`github`](#github) to load from files in GitHub repositories
- [`exec`](#local) from the output of a command
- [`graphql`](#graphql) for GraphQL APIs

For each of the sources, we support parsing JSON, YAML – both single and
multi-doc – and Jsonnet, where those files provide either a single source entry
or an array.

Examples of possible formats are in [Parsing](#parsing).

Some sources have config fields that support loading credentials from
environment variables. This is noted in the documentation for that source, with
more details on how this works in [Credentials](#credentials).

## `inline`

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

## `local`

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

See more details about acceptable formats in [Parsing](#parsing).

### Filename/Path

You can access the filename and the path of a local file you're importing from
in the `source` field of a config. For example:

```jsonnet
attributes: [
  {
    id: 'filename',
    name: 'Filename',
    type: 'String',
    array: false,
    source: '_local.filename',
  },
  {
    id: 'path',
    name: 'Path',
    type: 'String',
    array: false,
    source: '_local.path',
  },
]
```

These references will be replaced with the filename or path of the current file.

## `backstage`

If you already have a Backstage catalog setup, you can ask the importer to pull
directly from the Backstage API.

This looks like:

```jsonnet
// pipelines.*.sources.*
{
  backstage: {
    // This will depend on where your Backstage is located, and if it's
    // available on the same network as the importer.
    endpoint: 'https://backstage-internal.example.com/api/catalog/entities',

    // Supports environment variable substitution.
    // https://github.com/incident-io/catalog-importer/blob/master/docs/sources.md#credentials
    //
    // This token must be in base64 according to the Backstage requirements for
    // external API tokens.
    //
    // You can generate a token using something like:
    // $ node -p 'require("crypto").randomBytes(24).toString("base64")'
    //
    // https://backstage.io/docs/auth/service-to-service-auth/#usage-in-external-callers
    token: '$(BACKSTAGE_TOKEN)',

    // Some Backstage instances (e.g. Roadie) may prefer tokens to be used
    // as-is instead of signed into JWTs. If this is you, explicitly opt-out of
    // signing like so:
    sign_jwt: false,
  },
}
```

[backstage-endpoint]: https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entities

The `endpoint` should be pointing at whatever URL maps to [GET
/entries][backstage-endpoint] on your Backstage instance, and the `token` should
be a bearer token with permissions to make this call.

This will pull in all catalog entries after which you may use the source
`filter`s to separate entries into different types.

## `github`

This source can pull files matching a pattern from across repositories in a
GitHub account, and is useful when you already have catalog files such as the
catalog-info.yaml used by Backstage.

This would look like:

```jsonnet
// pipelines.*.sources.*
{
  github: {
    // Personal access token from GitHub.
    // https://github.com/incident-io/catalog-importer/blob/master/docs/sources.md#credentials
    token: "$(GITHUB_TOKEN)",
    repos: [
      "example-org/*",                    // find all repositories
      "another-example-org/example-repo", // or specific ones
    ],
    // Supports glob syntax like * for a single directory or ** for any number.
    files: [
      "**/catalog-info.yaml",
    ],
  },
}
```

The personal access token will need to have access to the organization that
contains the repos you'd like to source catalog data from.

GitHub has two types of personal access tokens, classic or fine-grained.
Depending on the type of token you create you'll need:

- Classic: `repo`, `user`, `read:org` and `read:discussion` scopes.
- Fine-grained:
  - Organization permissions:
    - Read access to members, organization administrative, team discussions.
  - Repository permissions:
    - Read access to code, discussions, metadata.

If you encounter issues, be sure to get in touch.

## `exec`

When you can't easily source catalog data from files or don't want it to be
inline, you can use the `exec` source to generate catalog entries from the
output of a command.

The command must output the same type of data as is supported by the `local`
source.

There are a few common use cases for the `exec` source:

### Complex transformations

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

### Loading from data warehouse

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

### From a remote API

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

## `graphql`

If you have a GraphQL API hosting data you'd like to import into the catalog,
you can get the importer to issue queries directly against the API and paginate
the results.

An example of generating a list of GitHub repositories using their GraphQL API
might be:

```jsonnet
// pipelines.*.sources.*
{
  graphql: {
    endpoint: 'https://api.github.com/graphql',
    headers: {
      authorization: 'Bearer $(GITHUB_TOKEN)',
    },
    query: |||
      query($cursor: String) {
        viewer {
          repositories(first: 50, after: $cursor) {
            edges {
              repository:node {
                name
                description
              }
            }
            pageInfo {
              endCursor
              hasNextPage
            }
          }
        }
      }
    |||,
    result: 'viewer.repositories.edges',
    paginate: {
      next_cursor: 'viewer.repositories.pageInfo.endCursor',
    },
  },
}
```

The example uses cursor based pagination but we support three pagination
strategies:

- No pagination, where the query has no variables.
- Use of a $page variable that is iterated once per page, or an $offset that is
  incremented by the number of results that have been seen.
- $cursor for cursor based pagination: this requires the `paginate.next_cursor`
  to specify where in the GraphQL result you should find the next cursor value.

## Credentials

For config fields that might contain sensitive values, we support substituting
values from environment variables into the value of that field.

As an example:

```jsonnet
{
  token: '$(SOME_TOKEN)',
}
```

Would be expanded into whatever the value of `SOME_TOKEN` is from the process
environment.

Wherever this is supported, it will be documented against that field.

## Parsing

All sources result in a collection of file contents. We try to parse entries
from those files, where an entry is a map of string keys to values.

For maximum flexibility, we try parsing several formats in this order:

- Jsonnet
- JSON
- YAML, split to support multidoc separators (`---`)

Once parsed, if the result of parsing is an object (map of string keys to
values) then that becomes the single result. But if the file returns an array of
objects, we'll interpret that as a list of entries and return all.

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
