# Catalog importer

Jump to [Getting started](#getting-started) if you want to begin from an example
configuration or just want to experiment.

Otherwise check-out the rest of our documentation for details on how the
importer works:

- [Understanding importer configuration](config.md)
- [Sources for catalog data such as files, GitHub, Backstage and more](sources.md)
- [Outputs that produce catalog types and entries](outputs.md)
- [Using expressions to filter and adjust source data](expressions.md)
- [Run the importer from CI tools like CircleCI or GitHub Actions](deploying.md)

[open-issue]: https://github.com/incident-io/catalog-importer/issues/new

If you can't find an answer to your question, please [open an issue][open-issue]
with your request and we'll be happy to help out.

## Getting started

[api-keys]: https://app.incident.io/settings/api-keys
[jsonnet]: https://jsonnet.org/
[vscode]: https://marketplace.visualstudio.com/items?itemName=Grafana.vscode-jsonnet
[cel]: https://github.com/google/cel-spec

The easiest way to get started is to copy one of the examples and tweak it to
match your needs, which will depend on whether you already use a catalog and the
type of catalog data you have available.

You can run `catalog-importer init`, which will give you a fresh copy of one of
the templates in a local directory.

Choose from:

- [Simple](simple), for anyone starting from scratch and wanting to load catalog
  data into incident.io directly from importer config.
- [Backstage](backstage), for those already using Backstage as a service catalog
  and want to import existing `catalog-info.yaml` files.

Once you've created a `importer.jsonnet`, visit your [incident dashboard][api-keys]
to create an API key with permission to:

- View catalog types and entries
- Manage catalog types and edit catalog data

Then set that token as your `INCIDENT_API_KEY` environment variable.

You can check your config is valid by running:

```
$ catalog-importer validate --config=importer.jsonnet
```

Then you can run a sync with:

```console
$ catalog-importer sync --config=importer.jsonnet

✔ Loaded config (3 pipelines, 3 sources, 3 outputs)
✔ Connected to incident.io API (https://api.incident.io)
✔ Found 16 catalog types, with 3 that match our sync ID (incident-io/catalog)
```

<details>
  <summary>
    Where this will be followed by progress output as entries are synced into
    the catalog:
  </summary>

```
↻ Creating catalog types that don't yet exist...
  ✔ Custom["Feature"] (id=01GYZMPSJPBE1ZFDF1ESEWFYZF)
  ✔ Custom["Integration"] (id=01GYZMPSV08SYE4RF49C3JZT76)
  ✔ Custom["Team"] (id=01GYZMPT7C692DXCEVHFHVKZAQ)

↻ Syncing catalog type schemas...
  ✔ Custom["Feature"] (id=01GYZMPSJPBE1ZFDF1ESEWFYZF)
  ✔ Custom["Integration"] (id=01GYZMPSV08SYE4RF49C3JZT76)
  ✔ Custom["Team"] (id=01GYZMPT7C692DXCEVHFHVKZAQ)

↻ Syncing pipeline... (Custom["Feature"])

  ↻ Loading data from sources...
    ✔ inline (found 30 entries)

  ↻ Syncing entries...

    ↻ Custom["Feature"]
      ✔ Building entries... (found 30 entries matching filters)
      ✔ No entries to delete
      ✔ Creating new entries in catalog... (30 entries to create)
      ✔ No existing entries to update

↻ Syncing pipeline... (Custom["Integration"])

  ↻ Loading data from sources...
    ✔ inline (found 21 entries)

  ↻ Syncing entries...

    ↻ Custom["Integration"]
      ✔ Building entries... (found 21 entries matching filters)
      ✔ No entries to delete
      ✔ Creating new entries in catalog... (21 entries to create)
      ✔ No existing entries to update

↻ Syncing pipeline... (Custom["Team"])

  ↻ Loading data from sources...
    ✔ inline (found 3 entries)

  ↻ Syncing entries...

    ↻ Custom["Team"]
      ✔ Building entries... (found 3 entries matching filters)
      ✔ No entries to delete
      ✔ Creating new entries in catalog... (3 entries to create)
      ✔ No existing entries to update
```

</details>

And that's it! Your data will now be loaded into your catalog.

Note that we only support up to 50,000 entries for each catalog type. Please contact us if you'd like to explore options for larger lists.
