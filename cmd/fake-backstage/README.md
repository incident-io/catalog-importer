# fake-backstage

A small server that answers the Backstage catalog API calls the importer's
[`backstage` source](../../docs/sources.md#backstage) makes, with seeded data. Use it to
try a config, or to check the [Backstage docs](../../docs/backstage/README.md), without a
real Backstage.

```console
$ go run ./cmd/fake-backstage --apis 5000
fake-backstage 10:15:02 serving 5042 entities: map[API:5003 Component:10 Domain:3 Group:6 Location:1 Resource:4 System:6 Template:1 User:8]
fake-backstage 10:15:02 listening on http://127.0.0.1:7007/api/catalog/entities/by-query (auth=none)
```

Then point a `backstage` source at `http://127.0.0.1:7007/api/catalog/entities/by-query`.
It logs every request with how many entities it matched, so you can see what a filter
fetches.

## Endpoints

- `GET /api/catalog/entities/by-query`, with `limit`, `cursor` and `filter`. The cursor
  carries the original filter, as in Backstage.
- `GET /api/catalog/entities`, with `limit`, `offset` and `filter`.

Filters follow the Backstage rules: conditions in one `filter` are comma-separated, a key
repeated in one filter matches any of its values, different keys must all match, and
several `filter` parameters match if any one does. Keys and values are
case-insensitive, `relations.<type>` matches relation targets, and a key without `=`
matches entities that have it.

## Data

A fictional engineering organisation:

- Groups: an Engineering department with Platform, Payments, Growth and Data teams, and
  SRE under Platform. Groups and Users mix `backstage.io/v1alpha1` and
  `backstage.io/v1beta1`.
- Users with `spec.memberOf`, written as both `payments` and
  `group:default/payments`. Growth lists its members on the Group instead, in
  `spec.members`.
- Components, Systems, Domains and Resources, with `spec.owner` in every reference form
  Backstage allows.
- An `example.com/on-call-team` annotation on most Components, naming the Group that
  gets paged. Platform owns `auth-service` and `api-gateway`, but SRE is on call for
  them.
- `--apis` generated API entities, plus a Location and a Template, which most real
  catalogs have and few people want in incident.io.

Every entity gets a `uid`, an `etag` and `relations` computed from its spec, as Backstage
returns them.

## Flags

| Flag | Default | |
| --- | --- | --- |
| `--addr` | `127.0.0.1:7007` | Address to listen on |
| `--apis` | `500` | How many API entities to generate |
| `--extra` | | YAML or JSON file of extra entities to serve, such as a `catalog-info.yaml`. An entity with the same kind, namespace and name as a seeded one replaces it. Repeatable |
| `--auth` | `none` | `static` wants `Authorization: Bearer <token>`. `legacy` wants a JWT signed with the base64 `--token`, as the importer sends when `sign_jwt` is left on |
| `--token` | | The token or secret for `--auth`. Also read from `FAKE_BACKSTAGE_TOKEN` |
| `--require-header` | | Also require a header, as `Name=value`, like a proxy in front of Backstage. Repeatable |
| `--quiet` | | Don't log each request |
