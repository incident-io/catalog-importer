# catalog-importer

This is the official catalog importer for the incident.io catalog.

You can use the importer to manage your catalog types, and sync entries from
various sources into the catalog on a regular basis. People often run this in a
cron or from their CI pipeline.

## How does this work?

The importer is controlled with a configuration file which specifies:

- Sources: a list of sources such as GitHub, GitLab, or local files that can be
  used to load catalog entries.
- Outputs: corresponding to catalog types, an output defines the catalog type
  schema and source filters to load catalog entries into that type.

The importer has a `sync` command that will use the configuration file to:

- Create catalog types
- Sync catalog type schemas
- Extract catalog data from sources
- For each output:
  - Filter and build entries from source
  - Delete any entries from incident.io that could not be found in source
  - Create or update remaining sourced entries

An example output is:

```
✔ Loaded config (1 source, 7 outputs)
✔ Connected to incident.io API (https://api.incident.io/)
✔ Found 14 catalog types, with 3 that match our sync ID (circleci-catalog)

↻ Prune enabled (--prune), removing types that are no longer in config...
  ✔ Nothing to prune!

↻ Creating catalog types that don't yet exist...
  ✔ Custom["BackstageAPI"] (id=01GXP4RGMZ6MCBT4S4T911K325)
  ✔ Custom["BackstageComponent"] (id=01GXP4TPZ5N0E4E94N6V42KM5T)
  ✔ Custom["BackstageGroup"] (id=01GX61MSWSRSE48YKCR99CK91Y)
  ✔ Custom["BackstageUser"] (id=01GVQD57QXPK5KC1NC5W1R4FV1)

↻ Removing types that are no longer in config...
  ✔ Custom["Service"] (id=01GXP4RGMZ6MCBT4S4T911K325)

↻ Syncing catalog type schemas...
  ✔ Custom["BackstageAPI"]
  ✔ Custom["BackstageComponent"]
  ✔ Custom["BackstageGroup"]
  ✔ Custom["BackstageUser"]
  ✔ Custom["Feature"]
  ✔ Custom["Integration"]
  ✔ Custom["Team"]

↻ Loading data from sources...
  ✔ github (found 3128 entries from 13 repositories)
  ✔ local-files (found 42 entries from 3 files)

↻ Syncing entries...

  ↻ Custom["BackstageAPI"]
    ✔ Building entries... (found 103 entries matching filters)
    ✔ Deleting unmanaged entries... (found 7 entries in catalog not in source)
        100% |████████████████████████████████████████| (7/7, 8 it/s)
    ✔ Syncing entries into catalog...
        100% |████████████████████████████████████████| (103/103, 17 it/s)

  ↻ Custom["BackstageComponent"]
    ✔ Building entries... (found 312 entries matching filters)
    ✔ Deleting unmanaged entries... (found 18 entries in catalog not in source)
        100% |████████████████████████████████████████| (18/18, 13 it/s)
    ✔ Syncing entries into catalog...
        100% |████████████████████████████████████████| (312/312, 26 it/s)
```
