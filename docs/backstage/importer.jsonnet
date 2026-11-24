local kinds = import 'kinds.libsonnet';

// Where to load Backstage entities from.
//
// The local source reads the example catalog-info.yaml next to this file, so
// you can try a sync before connecting to Backstage. Once that works, swap it
// for the backstage source below.
//
// https://github.com/incident-io/catalog-importer/blob/master/docs/sources.md#backstage
local sources = [
  {
    'local': {
      files: ['catalog-info.yaml'],
    },
  },

  /*
  {
    backstage: {
      endpoint: 'https://backstage.example.com/api/catalog/entities/by-query',

      // Only fetch the kinds listed in kinds.libsonnet. Add more conditions to
      // narrow it further, such as 'kind=component,metadata.namespace=default'.
      filter: kinds.filter,

      // A static token from Backstage's backend.auth.externalAccess config,
      // sent as "Authorization: Bearer <token>".
      token: '$(BACKSTAGE_TOKEN)',
      sign_jwt: false,

      // Or, for a legacy external access secret, remove sign_jwt and the
      // importer signs the base64 token into a short-lived JWT for you.

      // If Backstage sits behind a proxy that wants its own credentials, send
      // them as extra headers.
      // headers: {
      //   'X-Proxy-Secret': '$(BACKSTAGE_PROXY_SECRET)',
      // },
    },
  },
  */
];

{
  // Every catalog type this config creates is tagged with this ID, so the
  // importer only ever changes types it owns.
  sync_id: 'backstage',

  pipelines: [
    (import 'pipelines/entries.jsonnet')(sources, kinds),
    import 'pipelines/component-type.jsonnet',
    import 'pipelines/lifecycle.jsonnet',
  ] + (if kinds.has('API') then [import 'pipelines/api-type.jsonnet'] else []),
}
