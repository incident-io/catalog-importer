# Synced types

This example adds to catalog types that incident.io syncs, instead of creating
types of its own. For each team in `on-call.jsonnet`, it:

- Adds custom aliases to the team's schedule, such as its Slack handle.
- Adds the same handle as a custom alias of the team's escalation path, and sets
  the escalation path's runbook URL.

The importer matches each team to a schedule and an escalation path that already
exist, by name here. It never creates, deletes or renames them. If a name
matches nothing, the sync reports it and exits with an error once everything
else has synced.

Before you run it, add a `runbook_url` attribute to the Escalation path type in
the incident.io dashboard.

```console
catalog-importer sync --config importer.jsonnet --dry-run
```

See [synced types](../outputs.md#synced-types) for how this works.
