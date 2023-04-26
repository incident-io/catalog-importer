# Simple

This is an example of configuring a catalog from scratch, designed to be
centralised in a single repo.

It uses an example from the incident.io team where we sync three catalog types:

- Feature, for all product features.
- Integration, all third-party product integrations.
- Team, list all Product Development teams.

The root `config.jsonnet` file loads each pipeline, and is what the `sync`
command should be run with.
