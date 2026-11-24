# Backstage

This template imports your Backstage catalog into incident.io: components,
systems and domains, and the groups and users that own them. It reads from the
Backstage API, or from catalog-info.yaml files.

Out the box, it creates these catalog types:

| Catalog type             | From Backstage                                                 |
| ------------------------ | -------------------------------------------------------------- |
| Backstage Component      | `Component` entities, with owner, system and dependencies      |
| Backstage System         | `System` entities, with owner and domain                       |
| Backstage Domain         | `Domain` entities, with owner                                  |
| Backstage Group          | `Group` entities, with parent, members and escalation paths    |
| Backstage User           | `User` entities, linked to incident.io users by email          |
| Backstage Component Type | Service, website or library                                    |
| Backstage Lifecycle      | Experimental, production or deprecated                         |
| Backstage Tag            | Component tags                                                 |
| Backstage Group Type     | The group types in your catalog                                |

`API` and `Resource` entities are off by default. Turn them on in
[`kinds.libsonnet`](kinds.libsonnet).

For how this fits into incident.io, including making Backstage Groups your
teams, see [Backstage](https://docs.incident.io/catalog/backstage) in the
incident.io docs.

## Files

| File                              | What it holds                                                |
| --------------------------------- | ------------------------------------------------------------ |
| `importer.jsonnet`                | Where to load entities from, and how to authenticate          |
| `kinds.libsonnet`                 | Which Backstage kinds to import                               |
| `pipelines/entries.jsonnet`       | How each kind maps to a catalog type and its attributes       |
| `pipelines/component-type.jsonnet`, `pipelines/lifecycle.jsonnet`, `pipelines/api-type.jsonnet` | Fixed values for component types, lifecycles and API types |
| `catalog-info.yaml`               | Example entities, for a first sync before you connect Backstage |

## Get started

### 1. Install the importer

```console
brew tap incident-io/homebrew-taps
brew install catalog-importer
```

Or download a binary from the [releases](https://github.com/incident-io/catalog-importer/releases),
or use the `incidentio/catalog-importer` Docker image.

### 2. Copy this template

```console
catalog-importer init
```

Choose **Backstage**, and where to put it. The importer copies the template
into a `backstage` directory there. Run the commands below from inside it.

### 3. Create an API key

Create an API key in [Settings → API keys](https://app.incident.io/~/settings/api-keys)
with these permissions:

- **View catalog types and entries**
- **Manage catalog types and edit catalog data**

Then export it:

```console
export INCIDENT_API_KEY="<your API key>"
```

### 4. Try the example

From the template directory, sync the example entities in `catalog-info.yaml`:

```console
catalog-importer sync --config importer.jsonnet
```

This creates the catalog types above, with one entry in most of them. Open
**Catalog** in incident.io to see them.

### 5. Connect to Backstage

In `importer.jsonnet`, replace the `local` source with the `backstage` source
that's commented out below it, and set the endpoint to your Backstage's
`/api/catalog/entities/by-query`:

```jsonnet
local sources = [
  {
    backstage: {
      endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',
      filter: kinds.filter,
      token: '$(BACKSTAGE_TOKEN)',
      sign_jwt: false,
    },
  },
];
```

`filter: kinds.filter` means Backstage only sends the kinds you import, so
syncs stay quick on a big catalog. See [Authentication](#authentication) for
the token.

Check the config, then preview the changes:

```console
export BACKSTAGE_TOKEN="<your Backstage token>"
catalog-importer validate --config importer.jsonnet
catalog-importer sync --config importer.jsonnet --dry-run
```

When the preview looks right, run it for real:

```console
catalog-importer sync --config importer.jsonnet
```

The example entries go away, replaced by your Backstage entities.

### 6. Run it on a schedule

Each run syncs once, so run it on a schedule to keep incident.io in step with
Backstage. The importer needs to reach your Backstage, which usually means
running it inside your network. [Deploying](../deploying.md) has examples for
CI, a Kubernetes CronJob, and cron with Docker.

## Authentication

Anything calling Backstage's API from outside needs a token. Pick the option
that matches your Backstage config.

**Static token.** Backstage's `backend.auth.externalAccess` config can accept a
fixed token:

```yaml
# Backstage app-config.yaml
backend:
  auth:
    externalAccess:
      - type: static
        options:
          token: ${INCIDENT_IO_IMPORTER_TOKEN}
          subject: incident-io-catalog-importer
```

Send it as-is with `sign_jwt: false`:

```jsonnet
backstage: {
  endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',
  token: '$(BACKSTAGE_TOKEN)',
  sign_jwt: false,
},
```

**Legacy secret.** With `type: legacy` external access (or the older
`backend.auth.keys`), Backstage expects a JWT signed with a shared base64
secret. Give the importer the secret and leave `sign_jwt` out: it signs a
short-lived token on every run.

```jsonnet
backstage: {
  endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',
  token: '$(BACKSTAGE_SECRET)',
},
```

**Behind a proxy.** If an auth proxy or gateway sits in front of Backstage and
wants its own credentials, send them with `headers` (version 2.15.0 or later).
Values can come from environment variables:

```jsonnet
backstage: {
  endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',
  token: '$(BACKSTAGE_TOKEN)',
  sign_jwt: false,
  headers: {
    'X-Proxy-Client-Id': 'catalog-importer',
    'X-Proxy-Secret': '$(BACKSTAGE_PROXY_SECRET)',
  },
},
```

`header` changes which header carries the `Bearer <token>`, for a gateway that
moves the `Authorization` header elsewhere. See
[Sources](../sources.md#backstage) for every option.

**Roadie.** Use `https://api.roadie.so/api/catalog/entities/by-query` as the
endpoint, a Roadie API token, and `sign_jwt: false`.

To check a static or Roadie token works before you sync, ask for a single
entity:

```console
curl -H "Authorization: Bearer $BACKSTAGE_TOKEN" \
  'https://backstage.example.com/api/catalog/entities/by-query?limit=1'
```

This won't work with a legacy secret, which the importer signs into a JWT
first. Run `catalog-importer sync --dry-run` to check one of those instead.

## Choose what to import

### Kinds

[`kinds.libsonnet`](kinds.libsonnet) lists the kinds to import. Component,
System, Domain, Group and User are on, which is everything you need to route
alerts and incidents to the team that owns a service. To add API or Resource
entities, uncomment them:

```jsonnet
local kinds = [
  'Component',
  'System',
  'Domain',
  'Group',
  'User',
  'API',
  // 'Resource',
];
```

This list drives the Backstage filter, which catalog types get created, and
the attributes that link them. Turning on API adds the
**Backstage API** and **Backstage API Type** types, and **Provides APIs** and
**Consumes APIs** on components.

Only turn on a kind you'll use. Large catalogs can hold hundreds of thousands
of API entities, and each one is fetched and synced on every run.

### Filter in Backstage

The `filter` on the `backstage` source is passed to the Backstage API, so
entities it excludes are never fetched. `kinds.filter` gives
`kind=component,kind=system,kind=domain,kind=group,kind=user`: a key repeated
in one filter matches any of its values.

Different keys in one filter must all match. Because one filter applies to
every kind, a condition only some kinds have (like `spec.lifecycle`) would drop
the others. Use one source per condition instead, and the importer combines
them:

```jsonnet
local backstage = {
  endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',
  token: '$(BACKSTAGE_TOKEN)',
  sign_jwt: false,
};

local sources = [
  // Production components only
  { backstage: backstage { filter: 'kind=component,spec.lifecycle=production' } },
  // Everything else, from one namespace
  { backstage: backstage { filter: 'kind=system,kind=domain,kind=group,kind=user,metadata.namespace=default' } },
];
```

### Filter in an output

Each output in `pipelines/entries.jsonnet` has a `source.filter`, a JavaScript
expression run on every entity. Use it for conditions Backstage filters can't
express:

```jsonnet
source: {
  // Teams only, not departments or other group types
  filter: '$.kind == "Group" && $.spec.type == "team"',
  ...
},
```

Filters here run after entities are fetched, so prefer a Backstage filter for
anything that removes a lot of entities.

## Make Backstage Groups your teams

The **Backstage Group** type has what incident.io needs to use it as your
teams:

- **Members**: the incident.io users in each group, worked out from each
  Backstage user's groups and email address.
- **Escalation paths**: left for you to fill in from incident.io.
- **Parent**: the group above it, for team hierarchies.

After a sync, set Backstage Group as your team type in incident.io. The steps
are in [Use Backstage Groups as your teams](https://docs.incident.io/catalog/backstage#use-backstage-groups-as-your-teams).

Keep `members` and `escalation_paths` in the config once Backstage Group is
your team type: incident.io stops a sync from removing or changing an attribute
that powers teams.

## Add your own attributes

Annotations are where most teams keep everything else they know about an
entity. To bring one across, add an attribute to the output in
`pipelines/entries.jsonnet`. For example, to put each user's GitHub handle on
the **Backstage User** type:

```jsonnet
{
  id: 'github',
  name: 'GitHub',
  type: 'String',
  source: '$.metadata.annotations["github.com/user-login"]',
},
```

An attribute can link to another catalog type too. This puts an **On-call team**
on each component, from an annotation that names a Group:

```jsonnet
{
  id: 'on_call_team',
  name: 'On-call team',
  type: 'Custom["BackstageGroup"]',
  source: ref('$.metadata.annotations["example.com/on-call-team"]', 'group'),
},
```

`ref` (and `refs`, for lists) are helpers at the top of
`pipelines/entries.jsonnet`. They turn a Backstage entity reference like
`payments` or `group:default/Payments` into `default/payments`, which every
entry has as an alias. References match whatever the case, as they do in
Backstage.

## Upgrade from the old template

If you set this up from an earlier version of the template, two things changed
that a sync would otherwise undo.

**The escalation path attribute.** The old template had one escalation path per
group, in an attribute with the ID `escalation-path`. This one has a list, in
`escalation_paths`. If you've filled in the old one, or chosen it as your team
type's escalation paths attribute, keep it: in `pipelines/entries.jsonnet`, set
the attribute's `id` back to `escalation-path` and remove `array: true`.
Otherwise the sync drops the values you set, or fails because incident.io won't
change an attribute that powers teams.

**API entities.** The old template imported API entities. This one doesn't, so
your Backstage API type stops updating. Turn API on in `kinds.libsonnet` to keep
it (it no longer has a Definition attribute), or
[remove it](#remove-a-kind-you-imported-by-mistake).

## Remove a kind you imported by mistake

To stop importing a kind, take it out of `kinds.libsonnet` and sync with
`--prune`:

```console
catalog-importer sync --config importer.jsonnet --prune
```

`--prune` deletes catalog types that this config (its `sync_id`) created but no
longer has. The type goes in one request, and incident.io archives its entries
in the background, so it's quick however big the type is.

Don't remove the entries by filtering them out and syncing: the importer then
deletes them one at a time, which can take hours for a large type.
