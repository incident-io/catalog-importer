# Simple

This is an example of configuring a catalog from scratch, designed to be
centralised in a single repo.

It uses an example from the incident.io team where we sync three catalog types:

- Feature, for all product features.
- Integration, all third-party product integrations.
- Team, list all Product Development teams.

The root `catalog.jsonnet` file specifies all the catalog data, and
`importer.jsonnet` defines pipelines and is what the `sync` command should be
run with.

```console
$ catalog-importer sync --config=importer.jsonnet
```

![Backstage catalog types created by this config](dashboard.png)
