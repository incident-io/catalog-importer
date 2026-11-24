# Sources

We support several sources, catering for all the ways people normally store
their catalog data.

These are:

- [`inline`](#inline) for defining inside the importer config
- [`local`](#local) from local files
- [`backstage`](#backstage) for catalog data pulled from the Backstage API
- [`github`](#github) to load from files in GitHub repositories
- [`exec`](#exec) from the output of a command
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

## `backstage`

Pulls entities from the Backstage catalog API. For a complete config that maps
Backstage kinds to catalog types, start from the [Backstage
template](backstage/README.md).

```jsonnet
// pipelines.*.sources.*
{
  backstage: {
    // Your Backstage's GET /api/catalog/entities/by-query endpoint. It must be
    // reachable from wherever the importer runs.
    endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',

    // Optional. Passed to Backstage, so only matching entities are fetched.
    filter: 'kind=component,kind=system,kind=domain,kind=group,kind=user',

    // Optional. Supports environment variable substitution, see Credentials.
    token: '$(BACKSTAGE_TOKEN)',

    // Optional. Whether to sign the token into a JWT, defaults to true.
    sign_jwt: false,
  },
}
```

[by-query]: https://backstage.io/docs/features/software-catalog/software-catalog-api/#get-entitiesby-query
[external-access]: https://backstage.io/docs/auth/service-to-service-auth/#static-tokens

The `endpoint` should point at [GET /entities/by-query][by-query]. The older
`/api/catalog/entities` endpoint still works, but is deprecated.

### Authentication

How you configure `token` depends on how your Backstage accepts [external
callers][external-access]:

- **Static token** (`type: static` in `backend.auth.externalAccess`): set
  `token` to the token and `sign_jwt: false`. The importer sends
  `Authorization: Bearer <token>`.
- **Legacy secret** (`type: legacy`, or the older `backend.auth.keys`): set
  `token` to the base64 secret and leave `sign_jwt` out. The importer signs a
  JWT with the secret on every run, with `sub: backstage-server` and a one hour
  expiry.
- **Roadie**: use `https://api.roadie.so/api/catalog/entities/by-query` as the
  `endpoint`, a Roadie API token, and `sign_jwt: false`.
- **No auth**: leave `token` out, and the importer sends no `Authorization`
  header.

If something in front of Backstage needs other headers, such as an auth proxy
or API gateway, use these options alongside `token`, or instead of it:

```jsonnet
backstage: {
  endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',

  // Sent as-is with every request. Values support environment variable
  // substitution. Needs version 2.15.0 or later.
  headers: {
    'X-Proxy-Client-Id': 'catalog-importer',
    'X-Proxy-Secret': '$(BACKSTAGE_PROXY_SECRET)',
  },

  // Sends "Bearer <token>" in this header instead of Authorization.
  // header: 'X-Backstage-Authorization',
},
```

### Filtering

`filter` uses the [Backstage filter syntax][by-query], and Backstage applies it
before returning anything, so it's the cheapest way to leave entities out:

- Conditions are comma-separated, such as `kind=group,spec.type=team`.
- A key repeated in one filter matches any of its values, so
  `kind=component,kind=system` fetches both kinds.
- Different keys must all match, so `kind=component,spec.lifecycle=production`
  fetches production components only.
- A key with no value matches entities that have it, such as
  `metadata.annotations.pagerduty.com/service-id`.

Because different keys must all match, a condition that only some kinds have
filters out the rest. To combine conditions like that, use one `backstage`
source per filter in the same pipeline: the importer loads them all.

To filter on something Backstage can't, use the `filter` expression on each
output instead. See [Expressions](expressions.md).

### Other parameters

- `page_size`: how many entities to fetch per request. Defaults to 100.

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
    // Optional: excludeArchived, if true, will cause the source to exclude files from
    // repositories that have been archived. Defaults to false (archived repositories are included).
    // excludeArchived: true,
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
